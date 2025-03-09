package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"log"
)

// ----------- broadcast -----------
var RegisterBroadcast Interfaces.CommType = "RegisterBroadcast"

type RegisterBroadCom struct {
	con Interfaces.Consensus
}

func (com *RegisterBroadCom) Type() Interfaces.CommType {
	return RegisterBroadcast
}

func (com *RegisterBroadCom) Request(vars ...*[]byte) bool {
	byteTx := vars[0]
	com.con.SendMsg(&message.Message{
		Type:       com.Type().RequestType(),
		Content:    *message.NewByteContent(byteTx),
		RemoteInfo: config.IdLegalAddr,
	})
	return true
}

func (com *RegisterBroadCom) HandleRequest(msg *message.Message) bool {
	if msg.RemoteInfo == idChain.RunningNode.StrKey() {
		return true
	}
	contents := msg.GetContents()
	tx := Tx.DecodeTx(contents[0])
	log.Printf("Get new registerTx broadcast.")
	idChain.IDC.Chain.RecordTx(tx)
	return true
}

func (com *RegisterBroadCom) Response(...*[]byte) bool {
	log.Panic()
	return true
}

func (com *RegisterBroadCom) HandleResponse(*message.Message) {
	log.Panic()
}

func (com *RegisterBroadCom) Reset() Interfaces.CommType {
	com.con = Interfaces.Con[config.IdMod]
	return com.Type()
}
