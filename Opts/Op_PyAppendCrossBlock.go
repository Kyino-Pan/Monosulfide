package Opts

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"log"
	"sync"
)

type AppendCBlockOpt struct {
	con  Interfaces.Consensus
	lock sync.Mutex
}

func (op *AppendCBlockOpt) Schedule(vars ...*[]byte) {
	Propose(op.con, message.AppendCBlock, vars...)
	return
}
func (op *AppendCBlockOpt) Prepare([]*[]byte) bool {
	return true
}

func (op *AppendCBlockOpt) Verify([][]byte) bool {
	if op.con.GetDomain().ProcessingBlock() == nil {
		log.Panic()
	}
	return true
}

func (op *AppendCBlockOpt) Execute() {
	Interfaces.Communications[Interfaces.SyncIBlock].Request()
	pyramid.LocalShard.Append(pyramid.LocalShard.ProcessingBlock())
	op.con.GetDomain().SetProcessingBlock(nil)
	if pyramid.IsPyrMainNode() {
		Interfaces.Communications[Interfaces.CrossLock].HandleResponse(&message.Message{
			Type:       "",
			Content:    *message.NewByteContent(&config.FailByte),
			RemoteInfo: idChain.RunningNode.StrKey(),
		}) // telling itself to unblock inner tx.
	}
	return
}

func (op *AppendCBlockOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	op.con.GetProposalBuffer().SetPriority(message.AppendCBlock, Proposals.CrossCommitPriority)
	return message.AppendCBlock
}
