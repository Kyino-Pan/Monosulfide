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
		delay := config.InitDelay
		time.Sleep(delay)
		Interfaces.Operations[message.NewEpoch].Propose()
	}
}
