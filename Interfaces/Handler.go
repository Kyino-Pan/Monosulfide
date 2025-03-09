package Interfaces

import (
	"blockEmulator/message"
)

type Handler interface {
	MakeMsg(...interface{}) *message.Message
	Handle(*message.Message)
}

// todo
type MsgHandler struct {
}

func (mh *MsgHandler) MakeMsg(vars ...interface{}) *message.Message {
	return nil
}
func (mh *MsgHandler) Handle(msg *message.Message) {

}
