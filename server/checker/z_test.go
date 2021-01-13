package checker

import (
	"github.com/trezor/trezord-go/server/checker/xsy_proto"
	"github.com/trezor/trezord-go/server/checker/xsy_proto/messages"
	"log"
	"testing"
)

func init() {
	//Init(`D:\trezor\trezord-go\trezord.db`, &log.Logger{})
	Init(`/tmp/trezord.db`, &log.Logger{})
}

func newXSYEthereumSignTx() *messages.XSYEthereumSignTx {
	tmpSting := "string"
	tmpUint32 := uint32(1)
	tmpUint32Arr := []uint32{1}
	tmpByteArr := []byte{'1'}

	msgXSYEthereumSignTx := messages.XSYEthereumSignTx{}
	msgXSYEthereumSignTx.SignRsp = &messages.XSYSignCommonRsp{
		Address:   &tmpSting,
		Signature: tmpByteArr,
	}
	msgXSYEthereumSignTx.SignReq = &messages.XSYSignCommonReq{
		Symbol: &tmpSting,
		Amount: &tmpSting,
		To:     &tmpSting,
		Random: &tmpUint32,
	}
	msgXSYEthereumSignTx.Data = &messages.EthereumSignTx{
		AddressN:         tmpUint32Arr,
		Nonce:            tmpByteArr,
		GasPrice:         tmpByteArr,
		GasLimit:         tmpByteArr,
		To:               &tmpSting,
		Value:            tmpByteArr,
		DataInitialChunk: tmpByteArr,
		DataLength:       &tmpUint32,
	}

	return &msgXSYEthereumSignTx
}

func Test_001(t *testing.T) {
	tmpSymbol := "USDT"
	tmpAmount := "123"
	tmpTo := "to001"
	tmpRandom := uint32(1234)
	msgXSYEthereumSignTx := newXSYEthereumSignTx()
	msgXSYEthereumSignTx.SignReq = &messages.XSYSignCommonReq{
		Symbol: &tmpSymbol,
		Amount: &tmpAmount,
		To:     &tmpTo,
		Random: &tmpRandom,
	}

	callStr, errEncode := xsy_proto.EncodeCallData(messages.MessageType_MessageType_XSYEthereumSignTx, msgXSYEthereumSignTx)
	if errEncode != nil {
		log.Fatal("errEncode", errEncode)
	}

	errCheck := CheckCall(callStr)
	if errCheck != nil {
		log.Fatal("errCheck", errCheck)
	}
}

func TestTronReplay(t *testing.T) {
	tmpSymbol := "trx"
	tmpAmount := "2"
	tmpTo := "TGWmXKaMPV8LRQKr8hWsjRPdEizSfPJz7y"
	tmpRandom := uint32(123412)

	msgXSYTronSignTxReq := messages.XSYTronSignTxReq{
		AddressFrom: []uint32{44 | 0x80000000, 1 | 0x80000000, 0 | 0x80000000, 0, 1},
	}
	msgXSYTronSignTxReq.SignReq = &messages.XSYSignCommonReq{
		Symbol: &tmpSymbol,
		Amount: &tmpAmount,
		To:     &tmpTo,
		Random: &tmpRandom,
	}

	callStr, errEncode := xsy_proto.EncodeCallData(messages.MessageType_MessageType_XSYTronSignTxReq, &msgXSYTronSignTxReq)
	if errEncode != nil {
		log.Fatal("errEncode", errEncode)
	}

	errCheck := CheckCall(callStr)
	if errCheck != nil {
		log.Fatal("errCheck:", errCheck)
	}
}
