package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/config"
	"blockEmulator/consensus_shard/pow"
	"blockEmulator/message"
	"blockEmulator/monoxide"
	"log"
	"sync"
	"time"
)

type RelayTxOpt struct {
	con        Interfaces.Consensus
	tempBlocks []Block.Block
	opLock     sync.Mutex
}

func (op *RelayTxOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	op.tempBlocks = make([]Block.Block, 0)
	con.GetProposalBuffer().SetPriority(message.RelayTx, Proposals.NormalPriority)
	return message.RelayTx
}

func (op *RelayTxOpt) Schedule(...*[]byte) {
	byteArray := make([]*[]byte, config.MonoxideConf.ShardAmount)
	log.Printf("Schedule")
	Propose(op.con, message.RelayTx, byteArray...)
}

func (op *RelayTxOpt) Prepare(vars []*[]byte) bool {
	shard := monoxide.LocalShard
	blocks := shard.Chain.GenerateBlock()
	if len(blocks) == 0 {
		// generate failed.
		log.Println("Prepare fail")
		//Interfaces.Operations[message.RelayTx].Schedule()
		return false
	}
	for i, block := range blocks {
		byteBlock := block.Encode()
		vars[i] = &byteBlock
	}
	log.Printf("Prepare")

	return true
}

func (op *RelayTxOpt) Verify(vars [][]byte) bool {
	op.opLock.Lock()
	//log.Printf("Verify")
	for _, byteBlock := range vars {
		b := new(Block.RelayBlock).Decode(byteBlock)
		if block, ok := b.(*Block.RelayBlock); ok {
			flag := monoxide.LocalShard.Chain.VerifyBlock(block)
			if flag == true {
				op.tempBlocks = append(op.tempBlocks, block)
			} else {
				log.Panic(message.RelayTx, "Block Verification Failed")
				//op.opLock.Unlock()
				return false
			}
		} else {
			log.Panic()
		}
	}
	return true
}

func (op *RelayTxOpt) Execute() {
	defer op.opLock.Unlock()
	//log.Printf("Execute ,temp :%v", len(op.tempBlocks))
	shard := monoxide.LocalShard
	for _, block := range op.tempBlocks {
		cnt := 0
		for _, tx := range block.Body().Txs() {
			if shard.GetTxPool().TxExist(tx) {
				cnt++
			}
		}
		//log.Printf("%v / %v", cnt, len(block.Body().Txs()))
		shard.Append(block)
		//log.Printf("Append block%v", i)
	}
	log.Printf("%v remains.", shard.GetTxPool().Amount())
	if shard.GetTxPool().Amount() == 0 {
		space := 0
		for _, b := range shard.Chain.Blocks {
			space += len(b.Encode())
		}
		cnt := 0
		for sid := range config.MonoxideConf.ShardAmount {
			topH := shard.Chain.TopBlockHash[sid]
			tempBlock := shard.Chain.Blocks[topH]
			for tempBlock != shard.Chain.BlockGs[sid] {
				cnt++
				tempBlock = shard.Chain.Blocks[tempBlock.(*Block.RelayBlock).H.ParentHash]
			}
			cnt++
		}
		log.Printf("SPACE(%v blocks)(%v in pivot) :: %v\n", len(shard.Chain.Blocks), cnt, space)
		log.Printf("TIME :: %v", time.Since(op.con.(*pow.EasyPoW).StartTime))
		config.STOPPER <- true
	}
	op.tempBlocks = make([]Block.Block, 0)
	// log.Printf("Executed")
	return
}
