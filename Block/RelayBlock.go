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
	RelayBlock struct {
		H *RelayHead
		B *RelayBody
	}
	RelayHead struct {
		Nonce           uint64
		ParentHash      crypt.Hash // 上一区块Hash
		ConfirmedTxRoot []byte     // 片内交易的MerkleRoot
		OutTxRoot       []byte
		Timestamp       time.Time
	}
	RelayBody struct {
		Intra []*Tx.Transaction
		Out   []*Tx.Transaction
	}
)

func (head *RelayHead) Time() time.Time {
	return head.Timestamp
}

func _emptyRelayBlock() Block {
	return &RelayBlock{
		H: &RelayHead{
			ParentHash:      *crypt.EmptyHash(),
			ConfirmedTxRoot: nil,
			OutTxRoot:       nil,
			Timestamp:       time.Now(),
		},
		B: &RelayBody{
			Intra: nil,
			Out:   nil,
		},
	}
}

func (block *RelayBlock) Light() Block {
	block.B.Out = nil
	block.B.Intra = nil
	return block
}

func (head *RelayHead) TxRoot() []byte {
	return Tx.GenMPTRoot(append(make([][]byte, 0), head.ConfirmedTxRoot, head.OutTxRoot))
}

func (sBody RelayBody) Txs() []*Tx.Transaction {
	return sBody.Intra
}

func (block *RelayBlock) Nonce() uint64 {
	return block.H.Nonce
}

func (sBody RelayBody) EncodeB() []byte {
	return crypt.GetDigest(sBody)
}

func (head *RelayHead) EncodeH() []byte {
	return crypt.GetDigest(head)
}

func (block *RelayBlock) Head() Head {
	return block.H
}

func (block *RelayBlock) Body() Body {
	return block.B
}

func (block *RelayBlock) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(block)
	if err != nil {
		log.Panic(err)
	}
	tempBlock := new(RelayBlock).Decode(buff.Bytes())
	if tempBlock == nil {
		return nil
	}
	return buff.Bytes()
}

func (block *RelayBlock) Decode(byteBlock []byte) Block {
	if byteBlock == nil {
		return _emptyRelayBlock()
	}
	decoder := gob.NewDecoder(bytes.NewReader(byteBlock))
	err := decoder.Decode(block)
	if err != nil {
		log.Panic(err)
	}
	return block
}

func (block *RelayBlock) Hash() crypt.Hash {
	return *crypt.NewHash(crypt.GetDigest(block.H))
}

func (block *RelayBlock) Print() {
	h := block.H
	fmt.Println("--head--")
	fmt.Println(h.Nonce)
	fmt.Println(h.ParentHash.Bytes())
	fmt.Println(h.Time)
	fmt.Println("--body--")
	b := block.B
	fmt.Println("\t--sub block--")
	fmt.Println("--")
	fmt.Println(b.Intra != nil)
	fmt.Println("------")
	time.Sleep(40 * time.Millisecond)
}
