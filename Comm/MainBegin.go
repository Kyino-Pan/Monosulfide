package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Monosulfide"
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
	if config.FideConf.Enable {
		for _, sh := range Monosulfide.GlobalShards {
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
	if com.cnt == config.FideConf.ShardAmount {
		Interfaces.Con[config.FideMod].Enable()
		go Interfaces.Operations[message.FideTx].Propose()
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
