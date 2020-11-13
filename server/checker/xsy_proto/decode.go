package xsy_proto

import (
	"errors"
	"fmt"
	"github.com/trezor/trezord-go/server/checker/xsy_proto/messages"
	"google.golang.org/protobuf/proto"
)

type DecodeCallDataResponse struct {
	MessageType   int32
	MessageLength int32
	MessageBytes  []byte
}

func DecodeCallData(paramData string) (DecodeCallDataResponse, error) {
	retResult := DecodeCallDataResponse{}

	if len(paramData) <= 12 {
		return retResult, errors.New("decode call result error")
	}

	_, ssErr := fmt.Sscanf(paramData, "%04x%08x%x", &retResult.MessageType, &retResult.MessageLength, &retResult.MessageBytes)
	if ssErr != nil {
		return retResult, ssErr
	}

	return retResult, nil
}

type DecodeCallMessageResponse struct {
	DecodeCallDataResponse
	Failure *messages.Failure
}

// paramData 数据
// paramResType 期待返回的type
// paramResMessage 返回的消息体
func DecodeCallMessage(
	paramData string,
	paramResType messages.MessageType,
	paramResMessage proto.Message,
) (DecodeCallMessageResponse, error) {
	retResult := DecodeCallMessageResponse{}

	decodeRet, errDeode := DecodeCallData(paramData)
	if errDeode != nil {
		return retResult, errDeode
	}
	// 更新返回值
	retResult.DecodeCallDataResponse = decodeRet

	// 如果出错
	if decodeRet.MessageType == int32(messages.MessageType_MessageType_Failure) {
		tmpFail := messages.Failure{}
		errUnmarshal := proto.Unmarshal(decodeRet.MessageBytes, &tmpFail)
		if errUnmarshal != nil {
			return retResult, errDeode
		}
		// 更新返回值
		retResult.Failure = &tmpFail
	}

	// 返回的消息与期待消息一致
	if paramResMessage != nil && retResult.Failure == nil {
		if decodeRet.MessageType == int32(paramResType) {
			errUnmarshal := proto.Unmarshal(decodeRet.MessageBytes, paramResMessage)
			if errUnmarshal != nil {
				return retResult, errUnmarshal
			}
		}
	}

	return retResult, nil
}
