package Block

import (
	"blockEmulator/Tx"
	"blockEmulator/crypt"
	"bytes"
	"encoding/gob"
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
		ShardIdx     int
		ParentHash   crypt.Hash // 上一区块Hash
		IntraTxRoot  []byte     // 片内交易的MerkleRoot
		SubBlockRoot []byte     // 子Cross块的MerkleRoot
		StateRoot    []byte
		Timestamp    time.Time
	}
	FideBody struct {
		SubBlocks []*SubBlock
	}
	SubBlock struct {
		CHead *CBlockHead
		CBody *CBlockBody
	}
	CBlockHead struct {
		RemoteBlockHash crypt.Hash
		TxToot          []byte
	}
	CBlockBody struct {
		Txs []*Tx.Transaction
	}
)

func (sBody FideBody) Txs() []*Tx.Transaction {
	ret := make([]*Tx.Transaction, 0)
	for _, subBlock := range sBody.SubBlocks {
		if subBlock != nil {
			if subBlock.CBody != nil {
				ret = append(ret, subBlock.CBody.Txs...)
			}
		}
	}
	return ret
}

func (block *FideBlock) Ref(i int) crypt.Hash {
	return block.B.SubBlocks[i].CHead.RemoteBlockHash
}

func (head *FideHead) GetNonce() uint64 {
	return head.Nonce
}

func (head *FideHead) Time() time.Time {
	return head.Timestamp
}

func (block *FideBlock) Light() Block {
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

func (head *FideHead) ParentHashes() map[int]crypt.Hash {
	mp := make(map[int]crypt.Hash)
	if head == nil {
		return mp
	}
	mp[0] = head.ParentHash
	return mp
}

func (block *FideBlock) Nonce() uint64 {
	return block.H.GetNonce()
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
			Timestamp:    time.Now(),
		},
		B: &FideBody{
			SubBlocks: nil,
		},
	}
}

func (block *FideBlock) Hash() crypt.Hash {
	return *crypt.NewHash(crypt.GetDigest(block.H))
}

func EmptySubBlock() *SubBlock {
	return &SubBlock{
		CHead: nil,
		CBody: &CBlockBody{Txs: nil},
	}
}
