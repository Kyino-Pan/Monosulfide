package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/Spiral"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"log"
)

type SpiralTxOpt struct {
	con       Interfaces.Consensus
	tempBlock Block.Block
}

func (op *SpiralTxOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	con.GetProposalBuffer().SetPriority(message.SpiralTx, Proposals.NormalPriority)
	return message.SpiralTx
}

func (op *SpiralTxOpt) Propose(...*[]byte) {
	byteArray := new([]byte)
	//pyramid.LocalShard.WaitForSpiralTxs()
	op.con.Propose(message.SpiralTx, byteArray)
}

func (op *SpiralTxOpt) PrepareAfterLock(vars []*[]byte) bool {
	shard := Spiral.LocalShard
	op.tempBlock = shard.Chain.GenerateBlock()
	if op.tempBlock == nil {
		// generate failed.
		Interfaces.Operations[message.SpiralTx].Propose()
		return false
	}
	byteBlock := op.tempBlock.Encode()
	vars[0] = &byteBlock
	return true
}

func (op *SpiralTxOpt) Verify(vars [][]byte) bool {
	//pyramid.LocalShard.Lock()
	byteBlock := vars[0]
	b := new(Block.SpiralBlock).Decode(byteBlock)
	if block, ok := b.(*Block.SpiralBlock); ok {
		flag := Spiral.LocalShard.Chain.VerifyBlock(block)
		if flag == true {
			op.tempBlock = block
		} else {
			log.Println(message.SpiralTx, "Block Verification Failed")
		}
		return flag
	} else {
		log.Panic()
	}
	return false
}

func (op *SpiralTxOpt) Execute() {
	shard := Spiral.LocalShard
	shard.Append(op.tempBlock)
	//shard.Unlock()
	if Spiral.LocalShard.Main() == idChain.RunningNode {
		Interfaces.Operations[message.SpiralTx].Propose()
	}
	return
}
