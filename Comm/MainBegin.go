package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Monosulfide"
	"blockEmulator/config"
	"blockEmulator/message"
	"sync"
	"time"
)

// MainBeginCom is used to sync the nodes at the beginning of the tx consensus.
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
		go Interfaces.Operations[message.FideTx].Schedule()
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

func (com *MainBeginCom) Reset(con Interfaces.Consensus) Interfaces.CommType {
	com.con = con
	com.cnt = 0
	return com.Type()
}
