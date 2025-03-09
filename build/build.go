package build

import (
	"blockEmulator/Comm"
	"blockEmulator/Opts"
	"blockEmulator/config"
	"blockEmulator/consensus_shard"
	"blockEmulator/consensus_shard/pbft"
	"blockEmulator/launch"
)

import (
	"blockEmulator/test"
	"time"
)

func Run(mod uint64) {
	launch.TcpListen()
	pbft.Init()
	Comm.Init()
	Opts.Init()
	go consensus_shard.Selector()
	//go pyramidNode.PryCommunicating()
	go test.Trigger()
	go launch.Listener.Hearing()
	//go pyramidNode.TcpListen()
	go killerTicker()
	NodeKiller()
	//go networks.NetTester()
}

func killerTicker() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		killer <- true
	}
}

var killer = make(chan bool, 1)

func NodeKiller() {
	t := 600
	for {
		select {
		case <-config.STOPPER:
			t--
			return
		case <-killer:
			t--
			if t == 0 {
				return
			}
		}
	}
}
