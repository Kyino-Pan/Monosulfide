package Opts

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"time"
)

type CrossCommitOpt struct {
	con Interfaces.Consensus
}

func (op *CrossCommitOpt) Schedule(vars ...*[]byte) {
	// vars=[[seq]]
	Propose(op.con, message.CrossCommit, vars...)
}
func (op *CrossCommitOpt) Prepare([]*[]byte) bool {
	return true
}

func (op *CrossCommitOpt) Verify([][]byte) bool {
	// the bytes are verified in the pre-pre phase.
	return true
}

func (op *CrossCommitOpt) Execute() {
	block := op.con.GetDomain().ProcessingBlock()
	hash := block.Hash().Bytes()
	Interfaces.Communications[Interfaces.CrossReply].Request(&hash)
	pyramid.LocalShard.Append(block)
	if pyramid.IsPyrMainNode() {
		Interfaces.Communications[Interfaces.CrossLock].HandleResponse(&message.Message{
			Type:       "",
			Content:    *message.NewByteContent(&config.FailByte),
			RemoteInfo: idChain.RunningNode.StrKey(),
		},
		)
		go NxtCrossTx()
	}
}

func NxtCrossTx() {
	time.Sleep(config.SleepMin * 32 * time.Millisecond)
	Interfaces.Communications[Interfaces.CrossLock].Request()
}

func (op *CrossCommitOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	op.con.GetProposalBuffer().SetPriority(message.CrossCommit, Proposals.CrossCommitPriority)
	op.con = Interfaces.Con[config.PyrMod]
	return message.CrossCommit
}
