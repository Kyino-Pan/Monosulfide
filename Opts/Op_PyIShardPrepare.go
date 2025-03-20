package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"log"
)

type IShardPrepareOpt struct {
	con       Interfaces.Consensus
	tempBlock Block.Block
}

func (op *IShardPrepareOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	op.tempBlock = nil
	return message.IShardPrepare
}

func (op *IShardPrepareOpt) Propose(vars ...*[]byte) {
	// vars = [[byteBlock]]
	op.Verify(Interfaces.TransVars(vars))
	Propose(op.con, message.IShardPrepare, vars...)
	return
}

func (op *IShardPrepareOpt) PrepareAfterLock([]*[]byte) bool {
	return true
}

func (op *IShardPrepareOpt) Verify(vars [][]byte) bool {
	block := new(Block.StdBlock).Decode(vars[0])
	suc := pyramid.LocalShard.Chain.VerifyBlock(block)
	if suc {
		op.tempBlock = block
		return true
	} else {
		return false
	}
}

func (op *IShardPrepareOpt) Execute() {
	op.con.GetDomain().SetProcessingBlock(op.tempBlock)
	op.tempBlock = nil
	log.Printf("Processing block.len()=%v",
		len(op.con.GetDomain().ProcessingBlock().Body().Txs()))
	if pyramid.IsPyrMainNode() {
		Interfaces.Communications[Interfaces.CrossPrepare].Response(&config.SuccessByte)
	}
}
