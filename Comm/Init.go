package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/config"
)

func Deploy(com Interfaces.Communication) {
	comType := com.Reset()
	Interfaces.ComTypes[comType.RequestType()] = comType
	Interfaces.ComTypes[comType.ResponseType()] = comType
	Interfaces.Communications[comType] = com
}

func Init() {
	Deploy(new(RegisterCom))
	Deploy(new(RegisterBroadCom))
	Deploy(new(ViewChangeCom))
	Deploy(new(MigrateProCom))

	if config.PyrConf.Enable {
		Deploy(new(SyncIBlockCom))
		Deploy(new(CrossLockCom))
		Deploy(new(CrossPrepareCom))
		Deploy(new(CrossReplyCom))
	}

	if config.SpiConf.Enable {
		Deploy(new(SyncSBlockCom))
		Deploy(new(MainBeginCom))
		Deploy(new(PingCom))
	}
}
