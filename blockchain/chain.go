package blockchain

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/crypt"
	"blockEmulator/storage"
	"bytes"
	"fmt"
	"log"
	"time"
)

type Chain struct {
	//db           ethdb.Database     // the leveldb database to store in the disk, for status trie
	TopBlockHash map[int]crypt.Hash // the top block in this blockchain
	Storage      *storage.Storage   // Storage is the bolt-db to store the blocks
	TxPool       *Tx.Pool
	Blocks       map[crypt.Hash]Block.Block
}

func NewChain(name string, shard uint64) *Chain {
	pc := &Chain{
		Storage:      storage.NewStorage(name, shard),
		TxPool:       Tx.NewTxPool(0),
		Blocks:       make(map[crypt.Hash]Block.Block),
		TopBlockHash: make(map[int]crypt.Hash),
	}
	return pc
}

func (c *Chain) RecordTx(tx *Tx.Transaction) {
	c.TxPool.Append(tx)
}

func (c *Chain) GenerateIdBlock(randNum *[]byte) Block.Block {
	InnerTxs := c.TxPool.PackageInnerTxs()
	if len(InnerTxs) < 1 {
		time.Sleep(10 * time.Millisecond)
		InnerTxs = c.TxPool.PackageInnerTxs()
	}
	head := &Block.StdHead{
		ParentHashes: make(map[int]crypt.Hash),
		MerkleRoot:   Tx.GenTxRoot(InnerTxs),
		Nonce:        c.Blocks[c.TopBlockHash[c.Id()]].Nonce() + 1,
	}
	head.ParentHashes[c.Id()] = c.TopBlockHash[c.Id()]
	body := &Block.StdBody{
		Transactions: InnerTxs,
	}
	body.Interface = *randNum

	block := new(Block.StdBlock).Init(head, body)
	return block
}
func (c *Chain) Append(block Block.Block) {
	//block.print()
	Txs := block.Body().Txs()
	if Txs != nil {
		//log.Printf("Tx amount: %v", c.TxPool.TxLists[0][0].Len())
		c.TxPool.RemoveTxs(Txs)
		//log.Printf("Remaining: %v", c.TxPool.TxLists[0][0].Len())
	}
	c.TopBlockHash[0] = block.Hash()
	c.Blocks[block.Hash()] = block
	currBlock := block
	{
		// test code
		for {
			if currBlock == nil {
				// 到达创世区块
				break
			}
			if b, ok := currBlock.(*Block.StdBlock); ok {
				preHash := b.H.ParentHashes[c.Id()]
				fmt.Printf("-> %p\t %d\n", currBlock, len(b.B.Txs()))
				currBlock = c.Blocks[preHash]
			}
		}
	}
	c.Storage.AddBlock(block)
	return
}

func (c *Chain) Id() int {
	return 0
}

func (c *Chain) resultValidity(tx *Tx.Transaction) bool {
	//todo
	return true
}

func (c *Chain) GetBlock(hash []byte) Block.Block {
	if b := c.Blocks[*crypt.NewHash(hash)]; b != nil {
		return b
	}
	byteBlock, _ := c.Storage.GetBlock(hash)
	b := new(Block.StdBlock).Decode(byteBlock)
	return b
}

func (c *Chain) GetBlocks() []Block.Block {
	currHash := c.TopBlockHash[c.Id()]
	var blocks []Block.Block
	for {
		bb, err := c.Storage.GetBlock(currHash.Bytes())
		block := new(Block.StdBlock).Decode(bb)
		if block == nil {
			log.Panic("block is null")
		}
		if err != nil {
			log.Printf("无法获取区块: %v", err)
			break
		}
		// 处理区块的逻辑（例如打印区块信息）
		fmt.Printf("区块编号: %d, 区块哈希: %v, 交易数量：%v\n", block.Nonce(), block.Hash().Bytes(), len(block.Body().Txs()))
		// 如果当前区块是创世区块，其父哈希可能为零
		blocks = append(blocks, block)
		if b, ok := block.(*Block.StdBlock); ok {
			if len(b.H.ParentHashes) == 0 {
				//fmt.Println("已到达创世区块")
				break
			}
			// 更新当前哈希为父区块哈希
			currHash = b.H.ParentHashes[0]
		} else {
			log.Panic()
		}
	}
	return blocks
}

func (c *Chain) Verify(block Block.Block) bool {
	// todo
	return true
}

func (c *Chain) TopBlock() Block.Block {
	return c.Blocks[c.TopBlockHash[c.Id()]]
}

func (c *Chain) VerifyBlock(block Block.Block) bool {
	if b, ok := block.(*Block.StdBlock); ok {
		if !bytes.Equal(b.H.MerkleRoot, Tx.GenTxRoot(b.B.Txs())) {
			log.Println("mpt root error.")
			return false
		}
		//todo
		//	other validity checks.
		//	add here
		return true
	} else {
		log.Panic("Undefined behavior.")
		return false
	}
}
