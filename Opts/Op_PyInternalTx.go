package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"time"
)

type InternalTxOpt struct {
	con Interfaces.Consensus
}

func (op *InternalTxOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	con.GetProposalBuffer().SetPriority(message.InternalTx, Proposals.NormalPriority)
	return message.InternalTx
}

func (op *InternalTxOpt) Propose(...*[]byte) {
	byteArray := new([]byte)
	time.Sleep(1 * time.Second)
	Propose(op.con, message.InternalTx, byteArray)
}

func (op *InternalTxOpt) PrepareAfterLock(vars []*[]byte) bool {
	shard := pyramid.LocalShard
	tempBlock := shard.Chain.GenerateInternalBlock()
	if tempBlock == nil {
		// generate failed.
		Interfaces.Operations[message.InternalTx].Propose()
		return false
	}
	byteBlock := tempBlock.Encode()
	pyramid.LocalShard.SetProcessingBlock(tempBlock)
	vars[0] = &byteBlock
	return true
}

func (op *InternalTxOpt) Verify(vars [][]byte) bool {
	//pyramid.LocalShard.Lock()
	byteBlock := vars[0]
	block := new(Block.StdBlock).Decode(byteBlock)
	flag := pyramid.LocalShard.Chain.VerifyBlock(block)
	if flag == true {
		pyramid.LocalShard.SetProcessingBlock(block)
	}
	//log.Printf("Verified\n")
	//log.Printf("threshold = %v", op.con.GetDomain().Threshold())
	return flag
}

func (op *InternalTxOpt) Execute() {
	shard := pyramid.LocalShard
	Interfaces.Communications[Interfaces.SyncIBlock].Request()
	//shard.Unlock()
	shard.Append(shard.ProcessingBlock())
	shard.SetProcessingBlock(nil)
	if pyramid.IsPyrMainNode() {
		Interfaces.Operations[message.InternalTx].Propose()
	}
	return
}
