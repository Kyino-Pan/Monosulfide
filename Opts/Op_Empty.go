package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/message"
)

var Empty message.RequestType = "Empty"

type EmptyOpt struct {
	con   Interfaces.Consensus
	block Block.Block
}

func (op *EmptyOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	con.GetProposalBuffer().SetPriority(Empty, Proposals.Emergency)
	return Empty
}

func (op *EmptyOpt) Propose(...*[]byte) {
	op.con.Propose(Empty)
	//op.con.Propose(Empty, newBlock.EncodeH())
}
func (op *EmptyOpt) PrepareAfterLock([]*[]byte) bool {
	return true
}

func (op *EmptyOpt) Verify([][]byte) bool {
	return true
}
func (op *EmptyOpt) Execute() {
	return
}
