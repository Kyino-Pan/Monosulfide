package Block

import (
	"blockEmulator/Tx"
	"blockEmulator/crypt"
	"encoding/json"
	"log"
	"time"
)

type StdHead struct {
	ParentHashes map[int]crypt.Hash
	MerkleRoot   []byte
	StateRoot    []byte
	Timestamp    time.Time
	Nonce        uint64
	Bits         uint32
}

func (bh *StdHead) Time() time.Time {
	return bh.Timestamp
}

func NewStdHead() *StdHead {
	return &StdHead{
		ParentHashes: make(map[int]crypt.Hash),
		MerkleRoot:   nil,
		StateRoot:    nil,
		Timestamp:    time.Now(),
		Nonce:        0,
	}
}

func (bh *StdHead) TxRoot() []byte {
	return bh.MerkleRoot
}

func (bh *StdHead) EncodeH() []byte {
	bbh, err := json.Marshal(bh)
	if err != nil {
		log.Println(err)
		return nil
	}
	return bbh
}

type StdBody struct {
	Transactions []*Tx.Transaction
	Interface    []byte
}

func (bb StdBody) Txs() []*Tx.Transaction {
	return bb.Transactions
}

func (bb StdBody) EncodeB() []byte {
	return crypt.GetDigest(bb)
}
