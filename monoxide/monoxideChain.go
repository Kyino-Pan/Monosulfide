package monoxide

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/storage"
	"bytes"
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
	Weight       map[crypt.Hash]int
	Path         map[crypt.Hash][]crypt.Hash
	TopBlockHash map[int]crypt.Hash
	BlockGs      map[int]Block.Block
	nonce        uint64
}

func (ch *Chain) Id() int {
	return ch.ownerShard.sid
}

func (ch *Chain) GenerateBlock() []Block.Block {
	ch.lock.RLock()
	defer ch.lock.RUnlock()
	ret := make([]Block.Block, 0)
	for sid := 0; sid < config.MonoxideConf.ShardAmount; sid++ {
		IntraTxs := make([]*Tx.Transaction, 0)
		OutTxs := make([]*Tx.Transaction, 0)
		for i := 0; i < config.MonoxideConf.ShardAmount; i++ {
			// package txs to different shards from txPool
			if i == sid {
				IntraTxs = ch.TxPool.PackageTxsFrom(sid)
				continue
			}
		}
		tempB := &Block.RelayBlock{
			H: &Block.RelayHead{
				Nonce:           uint64(sid),
				ParentHash:      ch.TopBlockHash[sid],
				ConfirmedTxRoot: Tx.GenTxRoot(IntraTxs),
				OutTxRoot:       Tx.GenTxRoot(OutTxs),
				Timestamp:       time.Now(),
			},
			B: &Block.RelayBody{
				Intra: IntraTxs,
				Out:   OutTxs,
			},
		}
		ret = append(ret, tempB)
	}
	return ret
}

func (ch *Chain) Append(block *Block.RelayBlock) {
	ch.lock.Lock()
	defer ch.lock.Unlock()
	//log.Printf("Append")
	//ch.TxPool.Print()
	isLegal := ch.VerifyBlock(block)
	sid := block.Nonce()
	if isLegal {
		ch.storage.AddBlock(block)
		ch.Blocks[block.Hash()] = block
		tempBlock := block
		ch.Path[tempBlock.H.ParentHash] = append(ch.Path[tempBlock.H.ParentHash], block.Hash())
		for ch.BlockGs[int(sid)] != tempBlock {
			ch.Weight[tempBlock.Hash()] += 1
			tempBlock = ch.Blocks[tempBlock.H.ParentHash].(*Block.RelayBlock)
		}
		if block.H.ParentHash == ch.TopBlockHash[int(sid)] {
			ch.TopBlockHash[int(sid)] = block.Hash()
			ch.TxPool.RemoveTxs(block.B.Intra)
		} else {
			top, fork := ch.findTop(sid)
			if fork != nil {
				rollBlock := ch.Blocks[ch.TopBlockHash[int(sid)]]
				for rollBlock != fork {
					for _, tx := range rollBlock.Body().Txs() {
						ch.TxPool.Append(tx)
					}
					rollBlock = ch.Blocks[rollBlock.(*Block.RelayBlock).H.ParentHash]
				}
				rollBlock = top
				for rollBlock != fork {
					ch.TxPool.RemoveTxs(rollBlock.Body().Txs())
					rollBlock = ch.Blocks[rollBlock.(*Block.RelayBlock).H.ParentHash]
				}
			} else {
				if top != ch.Blocks[ch.TopBlockHash[int(sid)]] {
					log.Panic()
				}
				ch.TopBlockHash[int(sid)] = block.Hash()
				ch.TxPool.RemoveTxs(block.B.Intra)
			}
			ch.TopBlockHash[int(sid)] = top.Hash()
		}
	} else {
		log.Panic()
	}
}

func (ch *Chain) VerifyBlock(block *Block.RelayBlock) bool {
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

//func (ch *Chain) LockMoney(commit []*Tx.Transaction) {
//	ch.TxPool.RemoveTxs(commit)
//	//todo
//}

func (ch *Chain) findTop(sid uint64) (Block.Block, Block.Block) {
	log.Printf("findTop sid:%v", sid)
	tempBlock := ch.Blocks[ch.TopBlockHash[int(sid)]]
	forkPoint := Block.Block(nil)
	currPath := make(map[crypt.Hash]bool)
	for ch.BlockGs[int(sid)] != tempBlock {
		currPath[tempBlock.Hash()] = true
		tempBlock = ch.Blocks[tempBlock.(*Block.RelayBlock).H.ParentHash]
	}
	//tempBlock = ch.BlockGs[int(sid)]
	for len(ch.Path[tempBlock.Hash()]) != 0 {
		maxW := 0
		tarH := tempBlock.Hash()
		for _, h := range ch.Path[tempBlock.Hash()] {
			if ch.Weight[h] > maxW {
				tarH = h
				maxW = ch.Weight[h]
			} else if ch.Weight[h] == maxW {
				if currPath[h] == true {
					tarH = h
				}
			}
		}
		if currPath[tarH] == false && forkPoint == nil {
			forkPoint = tempBlock
		}
		tempBlock = ch.Blocks[tarH]
	}
	log.Println("findTop ed")
	return tempBlock, forkPoint
}

func NewRelayChain(port uint64, shard *Shard) *Chain {
	ret := &Chain{
		storage:      storage.NewStorage(strconv.FormatUint(port, 10), uint64(shard.sid)),
		lock:         sync.RWMutex{},
		ownerShard:   shard,
		Blocks:       make(map[crypt.Hash]Block.Block),
		BlockGs:      make(map[int]Block.Block),
		TopBlockHash: make(map[int]crypt.Hash),
		Path:         make(map[crypt.Hash][]crypt.Hash),
		Weight:       make(map[crypt.Hash]int),
	}

	for sid := 0; sid < config.MonoxideConf.ShardAmount; sid++ {
		baseBlock := &Block.RelayBlock{
			H: &Block.RelayHead{
				Nonce:      uint64(sid),
				ParentHash: idChain.IDC.Chain.TopBlockHash[0],
			},
			B: &Block.RelayBody{
				Intra: nil,
				Out:   nil,
			},
		}
		ret.BlockGs[sid] = baseBlock
		ret.Blocks[baseBlock.Hash()] = baseBlock
		ret.TopBlockHash[sid] = baseBlock.Hash()
	}
	ret.TxPool = Tx.NewTxPool(shard.sid)
	return ret
}
