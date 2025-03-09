package Relay

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/storage"
	"bytes"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

type Chain struct {
	storage      *storage.Storage
	TxPool       *Tx.Pool
	lock         sync.RWMutex
	ownerShard   *Shard
	Blocks       map[crypt.Hash]Block.Block
	TopBlockHash map[int]crypt.Hash
	blockBuffer  map[int]map[crypt.Hash]Block.Block
	blockAck     map[int]map[crypt.Hash]int
	nonce        uint64
}

func (ch *Chain) Id() int {
	return ch.ownerShard.Id
}

func (ch *Chain) GenerateBlock() Block.Block {
	ch.lock.RLock()
	defer ch.lock.RUnlock()
	IntraTxs := make([]*Tx.Transaction, 0)
	OutTxs := make([]*Tx.Transaction, 0)
	for i := 0; i < config.RelayConf.ShardAmount; i++ {
		// package txs to different shards from txPool
		if i == LocalShard.Id {
			innerTx := ch.TxPool.PackageInnerTxs()
			IntraTxs = innerTx
			continue
		}
		txs := ch.TxPool.PackageCrossTx(i)
		OutTxs = append(OutTxs, txs...)
	}
	ret := &Block.RelayBlock{
		H: &Block.RelayHead{
			Nonce:           ch.Nonce() + 1,
			ParentHash:      ch.TopBlockHash[ch.Id()],
			ConfirmedTxRoot: Tx.GenTxRoot(IntraTxs),
			OutTxRoot:       Tx.GenTxRoot(OutTxs),
			Time:            time.Time{},
		},
		B: &Block.RelayBody{
			Intra: IntraTxs,
			Out:   OutTxs,
		},
	}
	return ret
}

func (ch *Chain) Append(block *Block.RelayBlock) {
	ch.lock.Lock()
	defer ch.lock.Unlock()
	//ch.TxPool.Print()
	isLegal := ch.VerifyBlock(block)
	if isLegal {
		if block.B.Intra != nil {
			Txs := block.Body().Txs()
			if len(Txs) > 0 {
				ch.TxPool.RemoveTxs(Txs)
			}
			bHash := block.Hash()
			ch.Blocks[bHash] = block
			ch.TopBlockHash[ch.Id()] = bHash
			ch.nonce++
			//Interfaces.Communications[Interfaces.SyncRelayTx].Request()
			//todo
		}
		if block.B.Out != nil {
			txToCommit := make([]*Tx.Transaction, 0)
			for _, txs := range block.B.Out {
				txToCommit = append(txToCommit, txs)
			}
			ch.LockMoney(txToCommit)
		}
		ch.storage.AddBlock(block)
		//ch.TxPool.Print()
	} else {
		log.Panic()
	}
}

func (ch *Chain) VerifyBlock(block *Block.RelayBlock) bool {
	// todo nonce check
	// check pre-block hash
	if block.H.ParentHash != ch.TopBlockHash[ch.Id()] {
		log.Println("ParentHash error")
		return false
	}
	txAll := make([]*Tx.Transaction, 0)
	//check intraTxs
	if !bytes.Equal(Tx.GenTxRoot(block.B.Intra), block.H.ConfirmedTxRoot) {
		log.Println("ConfirmedTxRoot error")
		return false
	}
	txAll = append(txAll, block.B.Intra...)
	if !bytes.Equal(Tx.GenTxRoot(block.B.Out), block.H.OutTxRoot) {
		log.Println("OutTxRoot error")
		return false
	}
	txAll = append(txAll, block.B.Out...)
	//check tx validity
	if !Tx.IsVaildTxSet(txAll) {
		log.Println("Balance not enough")
		return false
	}
	return true
}

func (ch *Chain) Nonce() uint64 {
	return ch.nonce
}

func (ch *Chain) LockMoney(commit []*Tx.Transaction) {
	ch.TxPool.RemoveTxs(commit)
	//todo
}

func NewRelayChain(port uint64, shard *Shard) *Chain {
	ret := &Chain{
		storage:      storage.NewStorage(strconv.FormatUint(port, 10), uint64(shard.Id)),
		lock:         sync.RWMutex{},
		ownerShard:   shard,
		Blocks:       make(map[crypt.Hash]Block.Block),
		TopBlockHash: make(map[int]crypt.Hash),
		blockBuffer:  make(map[int]map[crypt.Hash]Block.Block),
		blockAck:     make(map[int]map[crypt.Hash]int),
	}

	baseBlock := &Block.RelayBlock{
		H: &Block.RelayHead{
			Nonce:      0,
			ParentHash: idChain.IDC.Chain.TopBlockHash[0],
		},
		B: &Block.RelayBody{
			Intra: nil,
			Out:   nil,
		},
	}
	fmt.Println(baseBlock.Hash().Bytes())
	for i := 0; i < config.RelayConf.ShardAmount; i++ {
		ret.blockAck[i] = make(map[crypt.Hash]int)
		ret.blockBuffer[i] = make(map[crypt.Hash]Block.Block)
		ret.TopBlockHash[i] = baseBlock.Hash()
		ret.blockBuffer[i][baseBlock.Hash()] = baseBlock
	}
	ret.TxPool = Tx.NewTxPool(shard.Id)
	return ret
}
