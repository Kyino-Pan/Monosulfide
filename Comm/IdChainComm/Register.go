package IdChainComm

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

type RegisterCom struct {
	con      Interfaces.Consensus
	tempAddr string
}

func (com *RegisterCom) Type() Interfaces.CommType {
	return Interfaces.Register
}

func (com *RegisterCom) Request(...*[]byte) bool {
	con := com.con
	if launch.Listener.GetListenPort() == config.MainPort {
		// If is the first node in the system.
		idChain.InitNode(true)
		return true
	}
	con.SendMsg(&message.Message{
		Type:       Interfaces.Register.RequestType(),
		Content:    *message.NewStrContent(launch.Listener.GetLocalAddr()),
		RemoteInfo: config.DnsAddr + ":" + config.MainPort,
	})
	log.Printf("Sending register request to %s", config.DnsAddr+":"+config.MainPort)
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
		Type:       Interfaces.Register.ResponseType(),
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

func (com *RegisterCom) Reset(con Interfaces.Consensus) Interfaces.CommType {
	com.con = con
	return com.Type()
}
