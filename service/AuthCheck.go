package service

import (
	"blockEmulator/Interfaces"
	"blockEmulator/consensus_shard/pbft"
	"blockEmulator/message"
)

var (
	AuthCheck        Interfaces.CommType = "AuthCheck"
	AuthCheckRequest message.MessageType = "AuthCheckRequest"
	//AuthCheckResponse message.MessageType = "AuthCheckResponse"
)

type AuthCheckCom struct {
	mod    *pbft.ConPbft
	reqCnt uint64
}

func (com *AuthCheckCom) Request(...*[]byte) bool {
	return true
}

func (com *AuthCheckCom) HandleRequest(msg *message.Message) bool {
	return true
}

func (com *AuthCheckCom) Response(...*[]byte) bool {
	return true
}

func (com *AuthCheckCom) HandleResponse(*message.Message) {

}

func NewAuthCheckCom(con *pbft.ConPbft) *AuthCheckCom {
	return &AuthCheckCom{
		mod:    con,
		reqCnt: 0,
	}
}
