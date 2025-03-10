package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"sync"
)

type PingCom struct {
	con  Interfaces.Consensus
	cnt  int
	lock sync.Mutex
}

func (com *PingCom) Type() Interfaces.CommType {
	return Interfaces.Ping
}

func (com *PingCom) Request(...*[]byte) bool {
	com.lock.Lock()
	defer com.lock.Unlock()
	if config.SpiConf.Enable {
		for _, n := range idChain.IDC.NodeMap {
			com.con.SendMsg(&message.Message{
				Type:       com.Type().RequestType(),
				Content:    nil,
				RemoteInfo: n.IpAddr,
			})
		}
	}
	return true
}

func (com *PingCom) HandleRequest(*message.Message) bool {
	return true
}

func (com *PingCom) Response(...*[]byte) bool {
	return true
}

func (com *PingCom) HandleResponse(*message.Message) {
}

func (com *PingCom) Reset() Interfaces.CommType {
	com.con = Interfaces.Con[config.IdMod]
	com.cnt = 0
	return com.Type()
}
