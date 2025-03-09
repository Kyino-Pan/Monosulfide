package Block

import (
	"blockEmulator/crypt"
	"bytes"
	"encoding/gob"
	"log"
)

type StdBlock struct {
	H *StdHead
	B *StdBody
	//PreviousBlockHash *Hash
}

func (b *StdBlock) Light() Block {
	b.B.Transactions = nil
	return b
}

func (b *StdBlock) ParentHashes() map[int]crypt.Hash {
	return b.H.ParentHashes
}

func (b *StdBlock) Nonce() uint64 {
	return b.H.Nonce
}

func (b *StdBlock) Head() Head {
	return b.H
}

func (b *StdBlock) Body() Body {
	return b.B
}

func (b *StdBlock) Hash() crypt.Hash {
	return *crypt.NewHash(crypt.GetDigest(b.H))
}

func (b *StdBlock) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(b)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func (b *StdBlock) Decode(byteBlock []byte) Block {
	if byteBlock == nil {
		return _emptyBlock()
	}
	decoder := gob.NewDecoder(bytes.NewReader(byteBlock))
	err := decoder.Decode(b)
	if err != nil {
		log.Panic(err)
	}
	return b
}

func (b *StdBlock) Init(head *StdHead, body *StdBody) Block {
	b.H = head
	b.B = body
	return b
}

func _emptyBlock() *StdBlock {
	return &StdBlock{
		H: &StdHead{
			ParentHashes: nil,
			MerkleRoot:   nil,
		},
		B: &StdBody{
			Transactions: nil,
		},
	}
}
