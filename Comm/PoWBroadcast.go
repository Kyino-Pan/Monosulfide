package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/message"
	"sync"
)

type PoWBroadcastCom struct {
	con  Interfaces.Consensus
	lock sync.Mutex
}

func (com *PoWBroadcastCom) Type() Interfaces.CommType {
	return Interfaces.PoWBroadcast
}

func (com *PoWBroadcastCom) Request(vars ...*[]byte) bool {
	com.lock.Lock()
	defer com.lock.Unlock()
	content := message.Content(*vars[0])
	for _, n := range com.con.GetDomain().GetMap() {
		com.con.SendMsg(&message.Message{
			Type:       com.Type().RequestType(),
			Content:    content,
			RemoteInfo: n.IpAddr,
		})
	}
	return true
}

func (com *PoWBroadcastCom) HandleRequest(msg *message.Message) bool {
	contents := msg.GetContents()
	ReqType := string(contents[0])
	opt := Interfaces.Operations[message.RequestType(ReqType)]
	vars := make([][]byte, 0)
	for _, v := range contents[1:] {
		vars = append(vars, v)
	}
	success := opt.Verify(vars)
	if !success {
		return false
	}
	opt.Execute()
	return true
}

func (com *PoWBroadcastCom) Response(...*[]byte) bool {
	panic("should not reach here")
	return true
}

func (com *PoWBroadcastCom) HandleResponse(*message.Message) {
	panic("should not reach here")
}

func (com *PoWBroadcastCom) Reset() Interfaces.CommType {
	com.con = Interfaces.Con[config.IdMod]
	return com.Type()
}
