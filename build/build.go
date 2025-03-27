package build

import (
	"blockEmulator/Comm"
	"blockEmulator/Interfaces"
	"blockEmulator/Opts"
	"blockEmulator/config"
	"blockEmulator/consensus_shard"
	"blockEmulator/launch"
	"blockEmulator/message"
)

import (
	"time"
)

func Run(uint64) {
	launch.TcpListen()
	consensus_shard.Init()
	Comm.Init()
	Opts.Init()
	go consensus_shard.Selector() // Core logic loop
	go Trigger()                  // Propose Begin
	go launch.Listener.Hearing()  //
	go killerTicker()
	NodeKiller()
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

func Trigger() {
	time.Sleep(time.Second * 2) // waiting for main node init.
	if launch.Listener.GetListenPort() == config.ListenPort {
		//go NetTester()
		time.Sleep(config.InitDelay)
		Interfaces.Operations[message.NewEpoch].Propose()
	}
}
