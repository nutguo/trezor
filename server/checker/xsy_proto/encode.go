package xsy_proto

import (
	"fmt"
	"github.com/trezor/trezord-go/server/checker/xsy_proto/messages"
	"google.golang.org/protobuf/proto"
)

func EncodeCallData(paramMessageType messages.MessageType, paramMessage proto.Message) (string, error) {
	retBytes, retErr := proto.Marshal(paramMessage)
	if retErr != nil {
		return "", retErr
	}
	retString := fmt.Sprintf("%04x%08x%x", int32(paramMessageType), len(retBytes), retBytes)
	return retString, nil
}
