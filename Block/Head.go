package Block

import (
	"blockEmulator/Tx"
	"blockEmulator/crypt"
	"encoding/json"
	"log"
)

type StdHead struct {
	ParentHashes map[int]crypt.Hash
	MerkleRoot   []byte
	StateRoot    []byte
	Nonce        uint64
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
