package Comm

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"log"
)

var (
	Register Interfaces.CommType = "Register"
)

type RegisterCom struct {
	con      Interfaces.Consensus
	tempAddr string
}

func (com *RegisterCom) Type() Interfaces.CommType {
	return Register
}

func (com *RegisterCom) Request(...*[]byte) bool {
	con := com.con
	if launch.Listener.GetListenPort() == config.ListenPort {
		idChain.InitNode(true)
		return true
	}
	// The first node will listen
	// has connected to the local primary node
	con.SendMsg(&message.Message{
		Type:       Register.RequestType(),
		Content:    *message.NewStrContent(launch.Listener.GetLocalAddr()),
		RemoteInfo: config.DnsAddr + ":" + config.ListenPort,
	})
	//log.Println("RegisterRequest sent.")
	return true
}

func (com *RegisterCom) HandleRequest(msg *message.Message) bool {
	remoteId := msg.RemoteInfo
	contents := msg.GetContents()
	remoteAddr := string(contents[0])
	com.tempAddr = remoteAddr
	tx := Tx.GenerateRegisterTx(remoteId, remoteAddr, idChain.IDC.Nonce())
	idChain.IDC.Chain.RecordTx(tx)
	byteTx := tx.Encode()
	Interfaces.Communications[RegisterBroadcast].Request(&byteTx)
	com.Response()
	return true
}

func (com *RegisterCom) Response(...*[]byte) bool {
	log.Print("Registered")
	content := *idChain.IDC.ChainInfo()
	com.con.SendMsg(&message.Message{
		Type:       Register.ResponseType(),
		Content:    content,
		RemoteInfo: com.tempAddr,
	})
	com.tempAddr = ""
	return true
}

func (com *RegisterCom) HandleResponse(msg *message.Message) {
	contents := msg.GetContents()
	blockAmount := crypt.BytesToUint(contents[0])
	i := 1
	for ; i <= int(blockAmount); i++ {
		block := new(Block.StdBlock).Decode(contents[i])
		idChain.IDC.Append(block)
	}
	idChain.InitNode(false)
	launch.ClearMsgBuffer()
	log.Printf("--->>>	Initialize Finished")
}

func (com *RegisterCom) Reset() Interfaces.CommType {
	com.con = Interfaces.Con[config.IdMod]
	return com.Type()
}
