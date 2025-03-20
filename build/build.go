package build

import (
	"blockEmulator/Comm"
	"blockEmulator/Opts"
	"blockEmulator/config"
	"blockEmulator/consensus_shard"
	"blockEmulator/launch"
)

import (
	"blockEmulator/test"
	"time"
)

func Run(mod uint64) {
	launch.TcpListen()
	consensus_shard.Init()
	Comm.Init()
	Opts.Init()
	go consensus_shard.Selector() // Core logic loop
	go test.Trigger()             // Propose Begin
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
