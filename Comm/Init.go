package Comm

import (
	"blockEmulator/Comm/IdChainComm"
	"blockEmulator/Comm/MonosulfideComm"
	"blockEmulator/Comm/MonoxideComm"
	"blockEmulator/Comm/PyramidComm"
	"blockEmulator/Interfaces"
	"blockEmulator/config"
)

func Deploy(com Interfaces.Communication, con Interfaces.Consensus) {
	comType := com.Reset(con)
	Interfaces.ComTypes[comType.RequestType()] = comType
	Interfaces.ComTypes[comType.ResponseType()] = comType
	Interfaces.Communications[comType] = com
}

func Init() {
	id := Interfaces.Con[config.IdMod]
	Deploy(new(IdChainComm.RegisterCom), id)
	Deploy(new(IdChainComm.RegisterBroadCom), id)
	Deploy(new(ViewChangeCom), id)
	Deploy(new(IdChainComm.MigrateProCom), id)
	Deploy(new(PoWBroadcastCom), id)

	if config.PyrConf.Enable {
		pyr := Interfaces.Con[config.PyrMod]
		Deploy(new(PyramidComm.SyncIBlockCom), pyr)
		Deploy(new(PyramidComm.CrossLockCom), pyr)
		Deploy(new(PyramidComm.CrossPrepareCom), pyr)
		Deploy(new(PyramidComm.CrossReplyCom), pyr)
		Deploy(new(MainBeginCom), pyr)
		Deploy(new(PingCom), pyr)
	}

	if config.FideConf.Enable {
		monosulfide := Interfaces.Con[config.FideMod]
		Deploy(new(MonosulfideComm.SyncFideBlockComm), monosulfide)
		Deploy(new(MainBeginCom), monosulfide)
		Deploy(new(PingCom), monosulfide)
	}

	if config.MonoxideConf.Enable {
		monoxide := Interfaces.Con[config.RelayMod]
		Deploy(new(MonoxideComm.SyncSBlockCom), monoxide)
		Deploy(new(MainBeginCom), monoxide)
	}
}
