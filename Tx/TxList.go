package Tx

import (
	"blockEmulator/crypt"
	"sync"
)

type List struct {
	txs     map[crypt.Hash]*Transaction
	mapLock sync.RWMutex
}

func NewTxList() *List {
	return &List{
		txs: make(map[crypt.Hash]*Transaction),
	}
}

func (tl *List) append(tx *Transaction) {
	tl.mapLock.Lock()
	defer tl.mapLock.Unlock()
	tl.txs[tx.Hash()] = tx
}

func (tl *List) Len() int {
	tl.mapLock.RLock()
	defer tl.mapLock.RUnlock()
	return len(tl.txs)
}

func (tl *List) remove(tx *Transaction) {
	tl.mapLock.Lock()
	defer tl.mapLock.Unlock()
	delete(tl.txs, tx.Hash())
}

func (tl *List) TxExist(tx *Transaction) bool {
	tl.mapLock.RLock()
	defer tl.mapLock.RUnlock()
	return tl.txs[tx.Hash()] != nil
}
