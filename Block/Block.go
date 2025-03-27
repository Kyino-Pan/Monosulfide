package Block

import (
	"blockEmulator/Tx"
	"blockEmulator/crypt"
	"time"
)

type (
	Block interface {
		Head() Head
		Body() Body
		Encode() []byte
		Decode([]byte) Block
		Hash() crypt.Hash
		Light() Block
	}
	Head interface {
		EncodeH() []byte
		TxRoot() []byte
		Time() time.Time
		GetNonce() uint64
	}
	Body interface {
		EncodeB() []byte
		Txs() []*Tx.Transaction
	}
)
