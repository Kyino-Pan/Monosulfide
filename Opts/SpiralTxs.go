package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Monosulfide"
	"blockEmulator/Proposals"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"log"
)

type FideTxOpt struct {
	con       Interfaces.Consensus
	tempBlock Block.Block
}

func (op *FideTxOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	con.GetProposalBuffer().SetPriority(message.FideTx, Proposals.NormalPriority)
	return message.FideTx
}

func (op *FideTxOpt) Propose(...*[]byte) {
	byteArray := new([]byte)
	//pyramid.LocalShard.WaitForFideTxs()
	Propose(op.con, message.FideTx, byteArray)
}

func (op *FideTxOpt) PrepareAfterLock(vars []*[]byte) bool {
	shard := Monosulfide.LocalShard
	op.tempBlock = shard.Chain.GenerateBlock()
	if op.tempBlock == nil {
		// generate failed.
		Interfaces.Operations[message.FideTx].Propose()
		return false
	}
	byteBlock := op.tempBlock.Encode()
	vars[0] = &byteBlock
	return true
}

func (op *FideTxOpt) Verify(vars [][]byte) bool {
	//pyramid.LocalShard.Lock()
	byteBlock := vars[0]
	b := new(Block.FideBlock).Decode(byteBlock)
	if block, ok := b.(*Block.FideBlock); ok {
		flag := Monosulfide.LocalShard.Chain.VerifyBlock(block)
		if flag == true {
			op.tempBlock = block
		} else {
			log.Println(message.FideTx, "Block Verification Failed")
		}
		return flag
	} else {
		log.Panic()
	}
	return false
}

func (op *FideTxOpt) Execute() {
	shard := Monosulfide.LocalShard
	shard.Append(op.tempBlock)
	//shard.Unlock()
	if Monosulfide.LocalShard.Main() == idChain.RunningNode {
		Interfaces.Operations[message.FideTx].Propose()
	}
	return
}
