package pyramid

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
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

func originBlock(shardId int) Block.Block {
	b := new(Block.StdBlock).Decode(nil)
	ret := b.(*Block.StdBlock)
	ret.H = &Block.StdHead{
		ParentHashes: make(map[int]crypt.Hash),
		MerkleRoot:   Tx.GenTxRoot(nil),
		StateRoot:    GlobalShards[shardId].GenStateRoot(),
		Nonce:        0,
		Timestamp:    time.Now(),
	}
	ret.H.ParentHashes[shardId] = idChain.IDC.Chain.TopBlock().Hash()
	return ret
}

type PyrChain struct {
	storage              *storage.Storage // storage is the bolt-db to store the blocks
	TxPool               *Tx.Pool
	lock                 sync.RWMutex
	ownerShard           *Shard
	Blocks               map[crypt.Hash]Block.Block
	TopBlockHash         map[int]crypt.Hash
	TopHashToBeConfirmed map[int]bool // 乐观接受
	Nonce                uint64
}

func (pc *PyrChain) Id() int {
	return pc.ownerShard.Id()
}

func NewPyrChain(port uint64, shard *Shard) *PyrChain {
	ret := &PyrChain{
		TopBlockHash:         make(map[int]crypt.Hash),
		TopHashToBeConfirmed: make(map[int]bool),
		storage:              storage.NewStorage(strconv.FormatUint(port, 10), uint64(shard.Id())),
		ownerShard:           shard,
		Blocks:               make(map[crypt.Hash]Block.Block),
		Nonce:                0,
	}
	ret.TxPool = Tx.NewTxPool(shard.sid)
	return ret
}

func (pc *PyrChain) InitOriginBlocks() {
	shard := pc.ownerShard
	orgBlock := originBlock(shard.Id())
	pc.Append(orgBlock, shard.Id())
	log.Println(orgBlock.Hash().Bytes())
	for _, sId := range shard.RelatedIShard {
		// init i-shards' originBlock
		tempOrigin := originBlock(sId)
		pc.Append(tempOrigin, sId)
		log.Println(tempOrigin.Hash().Bytes())
	}
}

func (pc *PyrChain) RecordTx(tx *Tx.Transaction) {
	pc.lock.Lock()
	defer pc.lock.Unlock()
	pc.TxPool.Append(tx)
}

func (pc *PyrChain) GenerateInternalBlock() Block.Block {
	pc.lock.Lock()
	defer pc.lock.Unlock()
	InnerTxs := pc.TxPool.PackageInnerTxs()
	cnt := 3
	for len(InnerTxs) < 1 {
		time.Sleep(128 * time.Millisecond)
		InnerTxs = pc.TxPool.PackageInnerTxs()
		cnt--
		if cnt == 0 {
			// no inner
			return nil
		}
	}
	head := &Block.StdHead{
		ParentHashes: make(map[int]crypt.Hash),
		MerkleRoot:   Tx.GenTxRoot(InnerTxs),
		StateRoot:    pc.ownerShard.GenStateRoot(),
		Nonce:        pc.Nonce + 1,
	}
	//head.ParentHashes[pc.sid()] = pc.TopBlockHash.Bytes()
	body := &Block.StdBody{
		Transactions: InnerTxs,
	}
	head.ParentHashes[pc.Id()] = pc.TopBlockHash[pc.Id()]
	block := new(Block.StdBlock).Init(head, body)
	return block
}

func (pc *PyrChain) GenerateCrossBlock() Block.Block {
	pc.lock.Lock()
	defer pc.lock.Unlock()
	txs := pc.TxPool.PackageCrossTxs()
	cnt := 0
	for len(txs) < 1 {
		time.Sleep(100 * time.Millisecond)
		txs = pc.TxPool.PackageCrossTxs()
		cnt++
		if cnt >= 8 {
			txs = make([]*Tx.Transaction, 0)
			break
		}
	}
	head := &Block.StdHead{
		ParentHashes: make(map[int]crypt.Hash),
		MerkleRoot:   Tx.GenTxRoot(txs),
		StateRoot:    pc.ownerShard.GenStateRoot(),
		Nonce:        pc.Nonce + 1 + 10000,
	}
	body := &Block.StdBody{
		Transactions: txs,
	}
	fmt.Println("----  Gen Block  ----")
	for _, id := range pc.ownerShard.RelatedIShard {
		head.ParentHashes[id] = pc.TopBlockHash[id]
		fmt.Printf("Shard%v's topHash = %v\n", id, pc.TopBlockHash[id])
	}
	fmt.Println("----  End   Gen  ----")

	head.ParentHashes[pc.Id()] = pc.TopBlockHash[pc.Id()]
	block := new(Block.StdBlock).Init(head, body)
	return block
}

func (pc *PyrChain) VerifyBlock(block Block.Block) bool {
	pc._countRelateTxs(block)
	txs := block.Body().Txs()
	if bytes.Equal(Tx.GenTxRoot(txs), block.Head().TxRoot()) {
		return true
	} else {
		return false
	}
}

func (pc *PyrChain) _countRelateTxs(block Block.Block) {
	//todo
	txs := block.Body().Txs()
	cnt := 0
	for _, tx := range txs {
		// count related txs. This is only used in output.
		s, r := tx.SInShard(), tx.RInShard()
		if pc.ownerShard.Controls(s) && pc.ownerShard.Controls(r) {
			// if tx is with related i-shards
			cnt++
		} else if pc.Id() == s {
			// I'm sender
			pc.resultValidity(tx)
			cnt++
		} else if pc.Id() == r {
			// I'm receiver
			cnt++
		}
	}
	log.Printf("block len:%v, %v related txs", len(block.Body().Txs()), cnt)
}

func (pc *PyrChain) Append(block Block.Block, sid int) {
	pc.lock.Lock()
	defer pc.lock.Unlock()
	isLegal := pc.VerifyBlock(block)
	if isLegal {
		// empty block will NOT be accepted
		Txs := block.Body().Txs()
		pc.storage.AddBlock(block)
		pc.Nonce++
		// update Top Hash
		bHash := block.Hash()
		fmt.Printf("\tAppend: %v", bHash.Bytes())
		pc.Blocks[bHash] = block.Light()
		pc.TopBlockHash[sid] = bHash
		if len(Txs) > 0 {
			pc.TxPool.Print()
			pc.TxPool.RemoveTxs(Txs)
			// update hash map
			s := Txs[0].SInShard()
			r := Txs[0].RInShard()
			if s != r {
				// is cross block
				b, _ := block.(*Block.StdBlock)
				for sid, hash := range b.H.ParentHashes {
					if pc.ownerShard.Controls(sid) {
						if pc.TopBlockHash[sid] == hash {
							pc.TopBlockHash[sid] = block.Hash()
						}
					}
				}
				for _, i := range pc.ownerShard.RelatedIShard {
					pc.printChain(i)
				}
				pc.printChain(pc.Id())
			}
		}
		return
	} else {
		log.Printf("Block is illegal. Block len:%v", len(block.Body().Txs()))
	}
}

func (pc *PyrChain) resultValidity(tx *Tx.Transaction) bool {
	//todo
	return true
}

func (pc *PyrChain) AppendIShardIBlock(IBlock Block.Block, shardId int) {
	pc.lock.Lock()
	defer pc.lock.Unlock()
	Txs := IBlock.Body().Txs()
	pc.TxPool.RemoveTxs(Txs)

	pc.TopBlockHash[shardId] = IBlock.Hash()
	pc.Blocks[IBlock.Hash()] = IBlock.Light()
	if pc.TopBlockHash[shardId] != IBlock.Hash() {
		log.Panic()
	}
	fmt.Printf("blocks amount: %v\n", len(pc.Blocks))
	pc.printChain(shardId)
	pc.storage.AddBlock(IBlock)
	return
}

func (pc *PyrChain) printChain(i int) {
	b := pc.Blocks[pc.TopBlockHash[i]]
	if b == nil {
		log.Printf("block%v is error", i)
	}
	currBlock, _ := b.(*Block.StdBlock)
	fmt.Printf("Shard%v:", i)
	for currBlock.H.ParentHashes != nil {
		fmt.Printf("-> %d\t", currBlock.H.Nonce)
		preHash := currBlock.H.ParentHashes[i]
		if preHash == idChain.IDC.Chain.TopBlockHash[0] {
			break
		}
		currBlock = pc.Blocks[preHash].(*Block.StdBlock)
		if currBlock == nil {
			log.Panic()
		}
	}
	fmt.Println()
}

func (pc *PyrChain) GetBlock(hash []byte) Block.Block {
	byteBlock, err := pc.storage.GetBlock(hash)
	if err != nil {
		log.Panic()
	}
	block := new(Block.StdBlock).Decode(byteBlock)
	return block
}
