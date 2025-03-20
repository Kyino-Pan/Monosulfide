package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"log"
)

var CoSiCommit = message.MessageType("CoSiCommit")
var CrossTxT = "CrossTxT"

type CrossPrePreOpt struct {
	con       Interfaces.Consensus
	localId   int
	tempBlock Block.Block
}

func (op *CrossPrePreOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	return message.CrossPrePre
}

func (op *CrossPrePreOpt) Propose(...*[]byte) {
	byteArray := new([]byte)
	//log.Printf("Propose")
	//log.Printf("amount::%v", pyramid.LocalShard.UnconfirmedCrossTxsLen())
	Propose(op.con, message.CrossPrePre, byteArray)
}
func (op *CrossPrePreOpt) PrepareAfterLock(vars []*[]byte) bool {
	shard := pyramid.LocalShard
	op.localId = shard.Id
	//shard.Lock()
	newBlock := shard.Chain.GenerateCrossBlock()
	op.tempBlock = newBlock
	byteBlock := newBlock.Encode()
	vars[0] = &byteBlock
	log.Printf("Finish cross block")
	return true
}

func (op *CrossPrePreOpt) Verify(vars [][]byte) bool {
	//pyramid.LocalShard.Lock()
	byteBlock := vars[0]
	block := new(Block.StdBlock).Decode(byteBlock)
	flag := pyramid.LocalShard.Chain.VerifyBlock(block)
	if flag == true {
		op.tempBlock = block
	}
	//log.Printf("Verified\n")
	//log.Printf("threshold = %v", op.con.GetDomain().Threshold())
	return flag
}

func (op *CrossPrePreOpt) Execute() {
	// vars[0] = &byteBlock
	log.Printf("CPrePreExe::block len:%v", len(op.tempBlock.Body().Txs()))
	op.con.GetDomain().SetProcessingBlock(op.tempBlock)
	op.tempBlock = nil
	if pyramid.IsPyrMainNode() {
		Interfaces.Communications[Interfaces.CrossPrepare].Request()
		op.con.GetProposalBuffer().SetThreshold(Proposals.CrossCommitPriority)
	} else {
		//op.con.CoSiCommit()
	}
	return
}
