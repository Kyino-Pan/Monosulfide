package Opts

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/config"
	"blockEmulator/message"
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

func Propose(con Interfaces.Consensus, reqType message.RequestType, vars ...*[]byte) {
	con.GetProposalBuffer().Push(&Proposals.Proposal{
		ReqType: reqType,
		Vars:    vars,
	})
	//log.Printf("%v add to proBuffer, remaining %v", reqType, con.GetProposalBuffer().Amount())
	go con.Propose()
}
