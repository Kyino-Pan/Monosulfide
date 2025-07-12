package engine

import (
	"blockEmulator/Comm"
	"blockEmulator/Interfaces"
	"blockEmulator/Opts"
	"blockEmulator/config"
	"blockEmulator/consensus_shard"
	"blockEmulator/launch"
	"blockEmulator/message"
	"log"
)

import (
	"time"
)

func Run() {
	launch.TcpListen()
	consensus_shard.Init()
	Comm.Init()
	Opts.Init()

	go consensus_shard.Selector() // Core logic loop
	go Trigger()                  // Schedule Begin
	go launch.Listener.Hearing()  //
	go killerTicker()
	NodeKiller()
}

// killerTicker and NodeKiller are used to handle zombie threads.
var killer = make(chan bool, 1)

func killerTicker() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		killer <- true
	}
}

func NodeKiller() {
	t := config.TotalDataSize / 10
	for {
		select {
		case <-config.STOPPER:
			t--
			log.Printf("Receive STOPPER")
			return
		case <-killer:
			t--
			if t == 0 {
				log.Println("killer killing")
				return
			}
		}
	}
}

func Trigger() {
	time.Sleep(time.Second * 2) // waiting for main node init.
	if launch.Listener.GetListenPort() == config.MainPort {
		//go NetTester()
		time.Sleep(config.InitDelay)
		Interfaces.Operations[message.EpochReset].Schedule()
	}
}
