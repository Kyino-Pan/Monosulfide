package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"log"
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
	// Request is only called by main nodes.
	log.Printf("MainBegin sent")
	for _, sh := range Interfaces.GlobalShards {
		com.con.SendMsg(&message.Message{
			Type:       com.Type().RequestType(),
			Content:    nil,
			RemoteInfo: sh.Main().IpAddr,
		})
	}
	return true
}

func (com *MainBeginCom) HandleRequest(*message.Message) bool {
	// only main node reach here
	com.lock.Lock()
	defer com.lock.Unlock()
	com.cnt++
	if com.cnt == config.ShardAmount {
		switch config.CrossShardConsensus {
		case config.Pyramid:
			Interfaces.Con[config.PyrMod].Enable()
			go Interfaces.Operations[message.InternalTx].Schedule()
			if pyramid.LocalShard.IsBShard() {
				go pyramid.NxtCrossTx()
			}
		case config.UniRelay:
			Interfaces.Con[config.FideMod].Enable()
			go Interfaces.Con[config.FideMod].Tic()
			//go Interfaces.Operations[message.FideTx].Schedule()
			config.TxBegin = time.Now()
			config.CalcComm = true
		case config.ClassicRelay:
			Interfaces.Con[config.RelayMod].Enable()
			go Interfaces.Con[config.RelayMod].Tic()
			//if Interfaces.LocalShard.Main() == idChain.RunningNode {
			//	go Interfaces.Operations[message.RelayTx].Schedule()
			//}
		}
	}
	return true
}

func (com *MainBeginCom) Response(...*[]byte) bool {
	log.Panic("SHOULD NOT BE RECEIVED")
	return true
}

func (com *MainBeginCom) HandleResponse(*message.Message) {
	log.Panic("SHOULD NOT BE RECEIVED")
}

func (com *MainBeginCom) Reset(con Interfaces.Consensus) Interfaces.CommType {
	com.con = con
	com.cnt = 0
	return com.Type()
}
