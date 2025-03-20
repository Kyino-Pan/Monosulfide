package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/Relay"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"log"
)

type RelayTxOpt struct {
	con       Interfaces.Consensus
	tempBlock Block.Block
}

func (op *RelayTxOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	con.GetProposalBuffer().SetPriority(message.RelayTx, Proposals.NormalPriority)
	return message.RelayTx
}

func (op *RelayTxOpt) Propose(...*[]byte) {
	byteArray := new([]byte)
	log.Printf("Propose")
	//pyramid.LocalShard.WaitForRelayTxs()
	Propose(op.con, message.RelayTx, byteArray)
}

func (op *RelayTxOpt) PrepareAfterLock(vars []*[]byte) bool {
	shard := Relay.LocalShard
	op.tempBlock = shard.Chain.GenerateBlock()
	if op.tempBlock == nil {
		// generate failed.
		Interfaces.Operations[message.RelayTx].Propose()
		return false
	}
	byteBlock := op.tempBlock.Encode()
	vars[0] = &byteBlock
	return true
}

func (op *RelayTxOpt) Verify(vars [][]byte) bool {
	//pyramid.LocalShard.Lock()
	byteBlock := vars[0]
	b := new(Block.StdBlock).Decode(byteBlock)
	if block, ok := b.(*Block.RelayBlock); ok {
		flag := Relay.LocalShard.Chain.VerifyBlock(block)
		if flag == true {
			op.tempBlock = block
		} else {
			log.Println(message.RelayTx, "Block Verification Failed")
		}
		return flag
	} else {
		log.Panic()
	}
	return false
}

func (op *RelayTxOpt) Execute() {
	shard := Relay.LocalShard
	shard.Append(op.tempBlock)
	//shard.Unlock()
	if Relay.LocalShard.Main() == idChain.RunningNode {
		Interfaces.Operations[message.RelayTx].Propose()
	}
	return
}
