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
	FideBlock struct {
		H *FideHead
		B *FideBody
	}
	FideHead struct {
		Nonce        uint64
		ParentHash   crypt.Hash // 上一区块Hash
		IntraTxRoot  []byte     // 片内交易的MerkleRoot
		SubBlockRoot []byte     // 子Cross块的MerkleRoot
		StateRoot    []byte
		Time         time.Time
	}
	FideBody struct {
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

func (block *FideBlock) Light() Block {
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

func (head *FideHead) TxRoot() []byte {
	return head.IntraTxRoot
}

func (sBody FideBody) Txs() []*Tx.Transaction {
	return sBody.Intra
}

func (head *FideHead) ParentHashes() map[int]crypt.Hash {
	mp := make(map[int]crypt.Hash)
	if head == nil {
		return mp
	}
	mp[0] = head.ParentHash
	return mp
}

func (block *FideBlock) Nonce() uint64 {
	return block.H.Nonce
}

func (sBody FideBody) EncodeB() []byte {
	return crypt.GetDigest(sBody)
}

func (head *FideHead) EncodeH() []byte {
	return crypt.GetDigest(head)
}

func (block *FideBlock) Head() Head {
	return block.H
}

func (block *FideBlock) Body() Body {
	return block.B
}

func (block *FideBlock) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(block)
	if err != nil {
		log.Panic(err)
	}
	tempBlock := new(FideBlock).Decode(buff.Bytes())
	if tempBlock == nil {
		return nil
	}
	return buff.Bytes()
}

func (block *FideBlock) Decode(byteBlock []byte) Block {
	if byteBlock == nil {
		return _emptyFideBlock()
	}
	decoder := gob.NewDecoder(bytes.NewReader(byteBlock))
	err := decoder.Decode(block)
	if err != nil {
		log.Panic(err)
	}
	return block
}

func _emptyFideBlock() Block {
	return &FideBlock{
		H: &FideHead{
			ParentHash:   *crypt.EmptyHash(),
			IntraTxRoot:  nil,
			SubBlockRoot: nil,
			StateRoot:    nil,
			Time:         time.Now(),
		},
		B: &FideBody{
			Intra:     nil,
			SubBlocks: nil,
		},
	}
}

func (block *FideBlock) Hash() crypt.Hash {
	return *crypt.NewHash(crypt.GetDigest(block.H))
}

func (block *FideBlock) Print() {
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
