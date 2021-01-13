package checker

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/trezor/trezord-go/server/checker/xsy_proto"
	"github.com/trezor/trezord-go/server/checker/xsy_proto/messages"
	"math/big"
	"strconv"
)

func CheckCall(hexBody string) (retErr error) {

	// 以太坊链
	retErr = check_XSYEthereumSignTx(hexBody)
	if retErr != nil {
		return retErr
	}

	// 波场
	retErr = checkTronSignTx(hexBody)
	if retErr != nil {
		return retErr
	}

	// TODO 其他链的重放check

	return nil
}

func check_XSYEthereumSignTx(hexBody string) error {

	msgXSYEthereumSignTx := messages.XSYEthereumSignTx{}
	resultRet, errRet := xsy_proto.DecodeCallMessage(
		hexBody,
		messages.MessageType_MessageType_XSYEthereumSignTx,
		&msgXSYEthereumSignTx,
	)
	// 解析出错，那么跳过这次
	if errRet != nil {
		return nil
	}

	// 消息类型不匹配，则返回nil
	if resultRet.MessageType != int32(messages.MessageType_MessageType_XSYEthereumSignTx) {
		return nil
	}

	// 消息类型是Failure，则返回nil
	if resultRet.Failure != nil {
		return nil
	}

	msgSignReq := msgXSYEthereumSignTx.GetSignReq()
	if msgSignReq == nil {
		return nil
	}

	return checkSignReq(msgSignReq, msgXSYEthereumSignTx.GetData())
}

func checkSignReq(param *messages.XSYSignCommonReq, data *messages.EthereumSignTx) error {
	if param == nil {
		return nil
	}

	random := param.GetRandom()
	if random == 0 {
		return fmt.Errorf("random [%d] cannot be 0", random)
	}

	findExist := SignCommonReq{}
	findErr := getDb().Where(&SignCommonReq{
		Random: random,
	}).First(&findExist).Error

	var nonce = new(big.Int).SetBytes(data.Nonce).Uint64()
	fromAddress := addressN2str(data.GetAddressN())

	if findErr != nil && findErr == gorm.ErrRecordNotFound {
		newData := SignCommonReq{
			Symbol:      param.GetSymbol(),
			To:          param.GetTo(),
			Amount:      param.GetAmount(),
			Random:      param.GetRandom(),
			FromAddress: fromAddress,
			Nonce:       uint32(nonce),
		}
		newErr := getDb().Create(&newData).Error
		if newErr != nil {
			return fmt.Errorf(newErr.Error())
		}
		return nil
	}

	if findErr != nil {
		return fmt.Errorf(findErr.Error())
	}
	// 利用同一个地址，一个nonce只能用一次的这个特性
	// nonce, fromAddress, toAddress, Symbol 四个值未变，不认为是重放,
	if findExist.Nonce == uint32(nonce) &&
		findExist.FromAddress == fromAddress &&
		findExist.To == param.GetTo() &&
		findExist.Symbol == param.GetSymbol() &&
		findExist.Amount == param.GetAmount() {
		return nil
	}

	return fmt.Errorf("random [%d] already exist", random)
}

func addressN2str(addressN []uint32) string {
	add := ""
	for _, v := range addressN {
		add = add + strconv.Itoa(int(v)) + "/"
	}

	return add
}
