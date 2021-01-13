package checker

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
	"github.com/trezor/trezord-go/server/checker/xsy_proto"
	"github.com/trezor/trezord-go/server/checker/xsy_proto/messages"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func checkTronSignTx(hexBody string) error {

	msgXSYTronSignTx := messages.XSYTronSignTxReq{}
	resultRet, errRet := xsy_proto.DecodeCallMessage(
		hexBody,
		messages.MessageType_MessageType_XSYTronSignTxReq,
		&msgXSYTronSignTx,
	)
	// 解析出错，那么跳过这次
	if errRet != nil {
		return nil
	}

	// 消息类型不匹配，则返回nil
	if resultRet.MessageType != int32(messages.MessageType_MessageType_XSYTronSignTxReq) {
		return nil
	}

	// 消息类型是Failure，则返回nil
	if resultRet.Failure != nil {
		return nil
	}

	return checkTronSignReq(msgXSYTronSignTx)
}

func checkTronSignReq(msg messages.XSYTronSignTxReq) error {

	param := msg.GetSignReq()
	if param == nil {
		return nil
	}

	random := param.GetRandom()
	if random == 0 {
		return fmt.Errorf("random [%d] cannot be 0", random)
	}

	var tronSignReq TronSignReq
	err := getDb().Where(&TronSignReq{Random: random}).First(&tronSignReq).Error

	fromAddress := addressN2str(msg.GetAddressFrom())

	// random不存在，则数据入库，不检查是否重放
	if err != nil && err == gorm.ErrRecordNotFound {
		newData := TronSignReq{
			FromAddress: fromAddress,
			Symbol:      param.GetSymbol(),
			To:          param.GetTo(),
			Amount:      param.GetAmount(),
			Random:      param.GetRandom(),
			CreateTime:  time.Now().Unix(),
		}
		newErr := getDb().Create(&newData).Error
		if newErr != nil {
			return fmt.Errorf(newErr.Error())
		}
		return nil
	}

	if err != nil {
		return err
	}

	// 已存在random,但是其他参数有一个不同，则认为是重放
	if tronSignReq.FromAddress != fromAddress ||
		tronSignReq.To != param.GetTo() ||
		tronSignReq.Amount != param.GetAmount() ||
		tronSignReq.Symbol != param.GetSymbol() {
		return fmt.Errorf("replay, random [%d]", random)
	}
	// 10分钟内，重发相同random的，认为可能有问题（链上不一定有数据）
	if time.Now().Unix() -  tronSignReq.CreateTime < 600 {
		return fmt.Errorf("replay, internal time too short, random [%d]", random)
	}

	// 通过查询to地址,从 tronSignReq.CreateTime 之后的历史交易，有相同金额的转账，认为是重放
	var isReplay bool
	if strings.ToLower(param.GetSymbol()) == "trx" {
		isReplay, err = IsReplayTrx(param.GetTo(), param.GetAmount(), tronSignReq.CreateTime)
	} else {
		isReplay, err = IsReplayTrc20(param.GetTo(), param.GetAmount(), tronSignReq.CreateTime)
	}

	if err != nil {
		return err
	}

	if isReplay {
		return fmt.Errorf("replay random [%d]", random)
	}

	return nil
}

// 分页查询，时间大于记录时间，到账的地址是toAddress的trc20交易，
// 有相同金额的，认为的重放，未找到不认为是重放，认为是交易失败后再次发起的
// 中间任何一次err，都返回 err, 认为检查失败，不发送签名请求

// 第一页查询传递参数，之后根据返回的结果确定下一页的链接
func IsReplayTrc20(toAddress, amount string, createTime int64) (bool, error) {

	minTimestamp := strconv.FormatInt(createTime*1000, 10) // 波场的接口查询时间戳毫秒级
	// 默认每页20条
	q1 := url.Values{
		"only_confirmed": {"true"},
		"only_to":        {"true"},
		"order_by":       {"block_timestamp,asc"},
		"min_timestamp":  {minTimestamp},
	}

	nextLink := fmt.Sprintf(`https://api.trongrid.io/v1/accounts/%v/transactions/trc20?%v`, toAddress, q1.Encode())

	for {

		httpResponse, err := http.Get(nextLink)
		if err != nil {
			return false, err
		}

		respBytes, err := ioutil.ReadAll(httpResponse.Body)
		_ = httpResponse.Body.Close()
		if err != nil {
			return false, err
		}

		var trc20Result Trc20Result
		err = json.Unmarshal(respBytes, &trc20Result)
		if err != nil {
			return false, err
		}

		if !trc20Result.Success {
			return false, errors.New("获取交易记录失败")
		}
		for _, v := range trc20Result.Data {

			vDecimal, err := decimal.NewFromString(v.Value)
			if err != nil {
				return false, err
			}

			vDecimal = vDecimal.Shift(-v.TokenInfo.Decimals)
			if vDecimal.String() == amount {
				return true, nil
			}
		}

		if trc20Result.Meta.Fingerprint == "" {
			break
		}
		if trc20Result.Meta.Links.Next == "" {
			break
		}

		nextLink = trc20Result.Meta.Links.Next
	}

	return false, nil
}

// 分页查询，时间大于记录时间，到账的地址是toAddress的trc交易，
// 有相同金额的，认为的重放，未找到不认为是重放，认为是交易失败后再次发起的
// 中间任何一次err，都返回 err, 认为检查失败，不发送签名请求

// 第一页查询传递参数，之后根据返回的结果确定下一页的链接
func IsReplayTrx(toAddress, amount string, createTime int64) (bool, error) {

	minTimestamp := strconv.FormatInt(createTime*1000, 10) // 波场的接口查询时间戳毫秒级

	// 默认每页20条
	q1 := url.Values{
		"only_confirmed": {"true"},
		"only_to":        {"true"},
		"order_by":       {"block_timestamp,asc"},
		"min_timestamp":  {minTimestamp},
	}

	nextLink := fmt.Sprintf(`https://api.trongrid.io/v1/accounts/%v/transactions?%v`, toAddress, q1.Encode())

	for {

		httpResponse, err := http.Get(nextLink)
		if err != nil {
			return false, err
		}

		respBytes, err := ioutil.ReadAll(httpResponse.Body)
		_ = httpResponse.Body.Close()
		if err != nil {
			return false, err
		}

		var trxResult TrxResult
		err = json.Unmarshal(respBytes, &trxResult)
		if err != nil {
			return false, err
		}

		if !trxResult.Success {
			return false, errors.New("获取交易记录失败")
		}
		for _, v := range trxResult.Data {

			if len(v.RawData.Contract) < 1 {
				continue
			}

			vStr := v.RawData.Contract[0].Parameter.Value.Amount

			vDecimal := decimal.NewFromInt(vStr)
			vDecimal = vDecimal.Shift(-6)
			if vDecimal.String() == amount {
				return true, nil
			}
		}

		if trxResult.Meta.Fingerprint == "" {
			break
		}
		if trxResult.Meta.Links.Next == "" {
			break
		}

		nextLink = trxResult.Meta.Links.Next

	}

	return false, nil
}

type Trc20Result struct {
	Data []struct {
		BlockTimestamp int64  `json:"block_timestamp"`
		From           string `json:"from"`
		To             string `json:"to"`
		TokenInfo      struct {
			Address  string `json:"address"`
			Decimals int32  `json:"decimals"`
			Name     string `json:"name"`
			Symbol   string `json:"symbol"`
		} `json:"token_info"`
		TransactionID string `json:"transaction_id"`
		Type          string `json:"type"`
		Value         string `json:"value"`
	} `json:"data"`
	Meta struct {
		At          int64  `json:"at"`
		Fingerprint string `json:"fingerprint"`
		Links       struct {
			Next string `json:"next"`
		} `json:"links"`
		PageSize int64 `json:"page_size"`
	} `json:"meta"`
	Success bool `json:"success"`
}

type TrxResult struct {
	Data []struct {
		BlockNumber          int64         `json:"blockNumber"`
		BlockTimestamp       int64         `json:"block_timestamp"`
		EnergyFee            int64         `json:"energy_fee"`
		EnergyUsage          int64         `json:"energy_usage"`
		EnergyUsageTotal     int64         `json:"energy_usage_total"`
		InternalTransactions []interface{} `json:"internal_transactions"`
		NetFee               int64         `json:"net_fee"`
		NetUsage             int64         `json:"net_usage"`
		RawData              struct {
			Contract []struct {
				Parameter struct {
					TypeURL string `json:"type_url"`
					Value   struct {
						Amount       int64  `json:"amount"`
						OwnerAddress string `json:"owner_address"`
						ToAddress    string `json:"to_address"`
					} `json:"value"`
				} `json:"parameter"`
				Type string `json:"type"`
			} `json:"contract"`
			Expiration    int64  `json:"expiration"`
			RefBlockBytes string `json:"ref_block_bytes"`
			RefBlockHash  string `json:"ref_block_hash"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"raw_data"`
		RawDataHex string `json:"raw_data_hex"`
		Ret        []struct {
			ContractRet string `json:"contractRet"`
			Fee         int64  `json:"fee"`
		} `json:"ret"`
		Signature []string `json:"signature"`
		TxID      string   `json:"txID"`
	} `json:"data"`
	Meta struct {
		At          int64  `json:"at"`
		Fingerprint string `json:"fingerprint"`
		Links       struct {
			Next string `json:"next"`
		} `json:"links"`
		PageSize int64 `json:"page_size"`
	} `json:"meta"`
	Success bool `json:"success"`
}
