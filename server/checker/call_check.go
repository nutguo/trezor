package checker

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/trezor/trezord-go/server/checker/xsy_proto"
	"github.com/trezor/trezord-go/server/checker/xsy_proto/messages"
)

func CheckCall(hexBody string) (retErr error) {

	// 以太坊链
	retErr = check_XSYEthereumSignTx(hexBody)
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

	return checkSignReq(msgSignReq)
}

func checkSignReq(param *messages.XSYSignCommonReq) error {
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
	if findErr == nil {
		return fmt.Errorf("random [%d] already exist", random)
	}

	if findErr != gorm.ErrRecordNotFound {
		return fmt.Errorf(findErr.Error())
	}

	newData := SignCommonReq{
		Symbol: param.GetSymbol(),
		To:     param.GetTo(),
		Amount: param.GetAmount(),
		Random: param.GetRandom(),
	}
	newErr := getDb().Create(&newData).Error
	if newErr != nil {
		return fmt.Errorf(newErr.Error())
	}
	return nil
}
