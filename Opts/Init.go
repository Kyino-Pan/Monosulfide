package Opts

import (
	"blockEmulator/Interfaces"
	"blockEmulator/config"
)

func Deploy(op Interfaces.Operation, con Interfaces.Consensus) {
	opType := op.Reset(con)
	Interfaces.Operations[opType] = op
}

func Init() {
	id := Interfaces.Con[config.IdMod]
	Deploy(new(NewEpochOpt), id)
	Deploy(new(RemoveNodeOpt), id)

	pyr := Interfaces.Con[config.PyrMod]
	Deploy(new(InternalTxOpt), pyr)
	Deploy(new(CrossPrePreOpt), pyr)
	Deploy(new(IShardPrepareOpt), pyr)
	Deploy(new(CrossCommitOpt), pyr)
	Deploy(new(AppendCBlockOpt), pyr)

	Fide := Interfaces.Con[config.FideMod]
	Deploy(new(FideTxOpt), Fide)
}
