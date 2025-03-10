package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Spiral"
	"blockEmulator/config"
	"blockEmulator/message"
	"sync"
	"time"
)

type MainBeginCom struct {
	con  Interfaces.Consensus
	cnt  int
	lock sync.Mutex
}

func (com *MainBeginCom) Type() Interfaces.CommType {
	return Interfaces.MainBegin
}

func (com *MainBeginCom) Request(...*[]byte) bool {
	com.lock.Lock()
	defer com.lock.Unlock()
	if config.SpiConf.Enable {
		for _, sh := range Spiral.GlobalShards {
			com.con.SendMsg(&message.Message{
				Type:       com.Type().RequestType(),
				Content:    nil,
				RemoteInfo: sh.Main().IpAddr,
			})
		}
	}
	return true
}

func (com *MainBeginCom) HandleRequest(*message.Message) bool {
	// only main node reach here
	com.lock.Lock()
	defer com.lock.Unlock()
	com.cnt++
	if com.cnt == config.SpiConf.ShardAmount {
		Interfaces.Con[config.SpiMod].EnablePropose()
		go Interfaces.Operations[message.SpiralTx].Propose()
		config.TxBegin = time.Now()
		config.CalcComm = true
	}
	return true
}

func (com *MainBeginCom) Response(...*[]byte) bool {
	return true
}

func (com *MainBeginCom) HandleResponse(*message.Message) {
}

func (com *MainBeginCom) Reset() Interfaces.CommType {
	com.con = Interfaces.Con[config.PyrMod]
	com.cnt = 0
	return com.Type()
}
