package Block

import (
	"blockEmulator/Tx"
	"blockEmulator/crypt"
)

type (
	Block interface {
		Head() Head
		Body() Body
		Encode() []byte
		Decode([]byte) Block
		Hash() crypt.Hash
		Nonce() uint64
		Light() Block
	}
	Head interface {
		EncodeH() []byte
		TxRoot() []byte
	}
	Body interface {
		EncodeB() []byte
		Txs() []*Tx.Transaction
	}
)
