package Block

import (
	"blockEmulator/Tx"
	"blockEmulator/crypt"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"time"
)

type (
	SpiralBlock struct {
		H *SpiralHead
		B *SpiralBody
	}
	SpiralHead struct {
		Nonce        uint64
		ParentHash   crypt.Hash // 上一区块Hash
		IntraTxRoot  []byte     // 片内交易的MerkleRoot
		SubBlockRoot []byte     // 子Cross块的MerkleRoot
		StateRoot    []byte
		Time         time.Time
	}
	SpiralBody struct {
		Intra     []*Tx.Transaction
		SubBlocks []*SubBlock
	}
	SubBlock struct {
		CHead *CBlockHead
		CBody *CBlockBody
	}
	CBlockHead struct {
		RemoteBlockHash []byte
		TxToot          []byte
	}
	CBlockBody struct {
		Txs []*Tx.Transaction
	}
)

func (block *SpiralBlock) Light() Block {
	block.B.Intra = nil
	for _, b := range block.B.SubBlocks {
		if b.CBody != nil {
			b.CBody.Txs = nil
		}
	}
	return block
}

func (b SubBlock) Hash() crypt.Hash {
	return *crypt.NewHash(crypt.GetDigest(b.CHead))
}

func (head *SpiralHead) TxRoot() []byte {
	return head.IntraTxRoot
}

func (sBody SpiralBody) Txs() []*Tx.Transaction {
	return sBody.Intra
}

func (head *SpiralHead) ParentHashes() map[int]crypt.Hash {
	mp := make(map[int]crypt.Hash)
	if head == nil {
		return mp
	}
	mp[0] = head.ParentHash
	return mp
}

func (block *SpiralBlock) Nonce() uint64 {
	return block.H.Nonce
}

func (sBody SpiralBody) EncodeB() []byte {
	return crypt.GetDigest(sBody)
}

func (head *SpiralHead) EncodeH() []byte {
	return crypt.GetDigest(head)
}

func (block *SpiralBlock) Head() Head {
	return block.H
}

func (block *SpiralBlock) Body() Body {
	return block.B
}

func (block *SpiralBlock) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(block)
	if err != nil {
		log.Panic(err)
	}
	tempBlock := new(SpiralBlock).Decode(buff.Bytes())
	if tempBlock == nil {
		return nil
	}
	return buff.Bytes()
}

func (block *SpiralBlock) Decode(byteBlock []byte) Block {
	if byteBlock == nil {
		return _emptySpiralBlock()
	}
	decoder := gob.NewDecoder(bytes.NewReader(byteBlock))
	err := decoder.Decode(block)
	if err != nil {
		log.Panic(err)
	}
	return block
}

func _emptySpiralBlock() Block {
	return &SpiralBlock{
		H: &SpiralHead{
			ParentHash:   *crypt.EmptyHash(),
			IntraTxRoot:  nil,
			SubBlockRoot: nil,
			StateRoot:    nil,
			Time:         time.Now(),
		},
		B: &SpiralBody{
			Intra:     nil,
			SubBlocks: nil,
		},
	}
}

func (block *SpiralBlock) Hash() crypt.Hash {
	return *crypt.NewHash(crypt.GetDigest(block.H))
}

func (block *SpiralBlock) Print() {
	h := block.H
	fmt.Println("--head--")
	fmt.Println(h.Nonce)
	fmt.Println(h.ParentHash.Bytes())
	fmt.Println(h.Time)
	fmt.Println("--body--")
	b := block.B
	fmt.Println("\t--sub block--")
	for i, s := range b.SubBlocks {
		fmt.Printf("\t--%v", i)
		fmt.Print("\t")
		fmt.Println(s.CHead != nil)
		fmt.Print("\t\t")
		fmt.Println(s.CBody != nil)
	}
	fmt.Println("--")
	fmt.Println(b.Intra != nil)
	fmt.Println("------")
	time.Sleep(40 * time.Millisecond)
}

func EmptySubBlock() *SubBlock {
	return &SubBlock{
		CHead: nil,
		CBody: &CBlockBody{Txs: nil},
	}
}
