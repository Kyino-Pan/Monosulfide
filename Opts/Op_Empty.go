package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/message"
)

type EmptyOpt struct {
	con   Interfaces.Consensus
	block Block.Block
}

func (op *EmptyOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	con.GetProposalBuffer().SetPriority(message.Empty, Proposals.Emergency)
	return message.Empty
}

func (op *EmptyOpt) Propose(...*[]byte) {
	Propose(op.con, message.Empty)
	//Interfaces.Propose(op.con,Empty, newBlock.EncodeH())
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
