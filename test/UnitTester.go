package test

import (
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/launch"
	"blockEmulator/message"
	"time"
)

func Trigger() {
	time.Sleep(time.Second * 2) // waiting for main node init.
	if launch.Listener.GetListenPort() == config.ListenPort {
		//go NetTester()
		time.Sleep(config.InitDelay)
		Interfaces.Operations[message.NewEpoch].Propose()
	}
}
