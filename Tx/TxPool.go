package Tx

import (
	"blockEmulator/config"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

type Pool struct {
	TxLists    [][]*List // txs[i][j] is the pointer to the array of Transactions
	local      uint
	localTxs   *List
	lock       sync.RWMutex
	localIndex int
	controls   []bool // This param is ONLY used by pyramid consensus
}

func NewTxPool(localIndex int) *Pool {
	var amount = config.ShardAmount
	ret := &Pool{
		TxLists: make([][]*List, amount),
		local:   uint(localIndex),
	}
	for i := 0; i < amount; i++ {
		ret.TxLists[i] = make([]*List, amount)
		for j := 0; j < amount; j++ {
			ret.TxLists[i][j] = NewTxList()
		}
	}
	ret.localTxs = ret.TxLists[localIndex][localIndex]
	ret.localIndex = localIndex
	if config.PyrConf.Enable {
		ret.controls = config.PyrConf.ShardDistribution[localIndex]
	}
	return ret
}

func (pool *Pool) Append(tx *Transaction) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	s := tx.SInShard()
	r := tx.RInShard()
	// tx.Time = time.Now()
	if config.PyrConf.Enable {
		if config.PyrConf.InRoute(pool.localIndex, tx.RInShard(), tx.SInShard()) {
			// 如果localShard是relay路径上的
			txs := SplitRelay(tx)
			local := pool.localIndex
			for _, tx := range txs {
				s := tx.SInShard()
				r := tx.RInShard()
				if config.PyrConf.ShardDistribution[local][s] || config.PyrConf.ShardDistribution[local][r] {
					list := pool.TxLists[s][r]
					list.append(tx)
				}
			}
		} else if pool.controls[s] || pool.controls[r] { // drop other shards' tx.
			list := pool.TxLists[s][r]
			list.append(tx)
		}
	} else if config.FideConf.Enable {
		list := pool.TxLists[s][r]
		list.append(tx)
	} else {
		list := pool.TxLists[s][r]
		list.append(tx)
	}
}

func (pool *Pool) PackageInnerTxs() []*Transaction {
	pool.lock.RLock()
	defer pool.lock.RUnlock()
	var transactions []*Transaction
	for _, tx := range pool.localTxs.txs {
		transactions = append(transactions, tx)
		if len(transactions) >= config.MaxBlockSize {
			break
		}
	}
	//pool.Print()
	return transactions
}

func (pool *Pool) PackageRelayTxs() ([][]*Transaction, int) {
	pool.lock.RLock()
	defer pool.lock.RUnlock()
	var candidateGroups = make([][]*Transaction, config.FideConf.ShardAmount)
	var bestIndex = pool.FindBest()
	remain := 0
	for dest, list := range pool.TxLists[bestIndex] {
		// 遍历从本地发出的交易
		var txs = make([]*Transaction, 0)
		for _, tx := range list.txs {
			txs = append(txs, tx)
		}
		candidateGroups[dest] = txs
		remain += len(txs)
	}

	// 使用轮询方式从各候选组中选取交易，保证分布均匀
	var transactions = make([][]*Transaction, config.FideConf.ShardAmount)
	cnt := 0
	round := 0
	for cnt < config.MaxBlockSize {
		roundAdd := false
		for dest, list := range candidateGroups {
			if len(list) > round {
				transactions[dest] = append(transactions[dest], list[round])
				roundAdd = true
				cnt++
			}
			if cnt >= config.MaxBlockSize {
				break
			}
		}
		if roundAdd == false {
			break
		}
		round++
	}
	log.Printf("%v/%v (S%v) packaged", cnt, pool.Amount(), bestIndex)
	return transactions, bestIndex
}

func (pool *Pool) PackageCrossTxs() []*Transaction {
	pool.lock.RLock()
	defer pool.lock.RUnlock()
	var transactions []*Transaction
	if !config.PyrConf.Enable {
		log.Panic("This func can only be used by pyramid, or fatal bug may exist.")
	}
	// 收集所有满足条件的交易组
	var candidateGroups [][]*Transaction
	for i, txLists := range pool.TxLists {
		for j, txList := range txLists {
			if i == j {
				// 跳过自身的交易列表
				continue
			}
			if pool.controls[i] && pool.controls[j] {
				var txs []*Transaction
				for _, tx := range txList.txs {
					txs = append(txs, tx)
				}
				candidateGroups = append(candidateGroups, txs)
			}
		}
	}
	// 使用轮询方式从各候选组中选取交易，保证分布均匀
	round := 0
	for len(transactions) < config.MaxBlockSize {
		roundAdded := false
		// 遍历所有候选组
		for _, group := range candidateGroups {
			if round < len(group) {
				transactions = append(transactions, group[round])
				roundAdded = true
				// 达到区块大小限制时退出
				if len(transactions) >= config.MaxBlockSize {
					break
				}
			}
		}
		// 如果这一轮没有添加任何交易，则退出循环
		if !roundAdded {
			break
		}
		round++
	}
	return transactions
}

func (pool *Pool) PackageTxsFrom(sid int) []*Transaction {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	var transactions []*Transaction
	for _, txList := range pool.TxLists[sid] {
		for _, tx := range txList.txs {
			transactions = append(transactions, tx)
		}
	}
	if len(transactions) > config.MaxBlockSize/config.MonoxideConf.ShardAmount {
		transactions = transactions[:config.MaxBlockSize/config.MonoxideConf.ShardAmount]
	}
	return transactions
}

func (pool *Pool) PackageCrossTx(dest int) []*Transaction {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	var transactions []*Transaction
	for _, tx := range pool.TxLists[pool.localIndex][dest].txs {
		transactions = append(transactions, tx)
	}
	return transactions
}

func (pool *Pool) RemoveTxs(txs []*Transaction) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	if txs == nil {
		return
	}
	cnt := 0
	for _, tx := range txs {
		if pool.TxExist(tx) {
			cnt++
		}
		pool.TxLists[tx.SInShard()][tx.RInShard()].remove(tx)
	}
	//log.Printf("%v tx removed. Block contain %v txs", cnt, len(txs))
}

func (pool *Pool) TxExist(tx *Transaction) bool {
	txList := pool.TxLists[tx.SInShard()][tx.RInShard()]
	return txList.TxExist(tx)
}

//func (pool *TxPool) HungryCross() (int, int, int) {
//	tls := pool.TxLists
//	a, b, maxAmount := -1, -1, -1
//	for i := 0; i < len(tls); i++ {
//		for j := i + 1; j < len(tls); j++ {
//			IJAmount := tls[i][j].len() + tls[j][i].len()
//			if maxAmount < IJAmount {
//				maxAmount = IJAmount
//				a, b = i, j
//			}
//		}
//	}
//	return a, b, maxAmount
//}

func (pool *Pool) CheckTxs(txs []*Transaction) bool {
	//todo
	return true
}

func (pool *Pool) Amount() int {
	cnt := 0
	for _, txLists := range pool.TxLists {
		for _, txList := range txLists {
			cnt += len(txList.txs)
		}
	}
	return cnt
}

func (pool *Pool) Print() {
	fmt.Println("----------------")
	for i := 0; i < len(pool.TxLists); i++ {
		for j := 0; j < len(pool.TxLists); j++ {
			if pool.TxLists[i][j].Len() == 0 {
				fmt.Printf("  0 \t")
			} else {
				fmt.Printf("%.1fk\t", float32(pool.TxLists[i][j].Len())/1000)

			}
		}
		fmt.Println()
	}

	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(pool.TxLists)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("%v", buff.Bytes())

	fmt.Println("----------------")

}

func (pool *Pool) FindBest() int {
	maxCnt := -1
	ret := -1
	for s, txLists := range pool.TxLists {
		cnt := 0
		if config.FideConf.Enable && config.FideConf.StorageOptimize == true {
			if !pool.controls[s] {
				continue
			}
		}
		for _, txList := range txLists {
			cnt += len(txList.txs)
		}
		if cnt > maxCnt {
			maxCnt = cnt
			ret = s
		}
	}
	return ret
}

func (pool *Pool) AppendRelay(intra []*Transaction) {
	for _, tx := range intra {
		if tx.Type != RelayDump && tx.SInShard() != tx.RInShard() {
			ntx := &Transaction{
				Sender:    tx.Sender,
				Recipient: tx.Recipient,
				Value:     tx.Value,
				Nonce:     tx.Nonce,
				Type:      RelayDump,
				Interface: tx.Interface,
				Time:      tx.Time,
			}
			pool.Append(ntx)
		}
	}
}

func (pool *Pool) ResetControl() {
	pool.controls = make([]bool, config.FideConf.ShardAmount)
	if config.FideConf.Enable {
		switch config.FideConf.ShardAmount {
		case 8:
			pool.controls[5] = true
			//pool.controls[3]= true
		case 16:
			pool.controls[13] = true
			pool.controls[11] = true
			//pool.controls[12] = true
			//pool.controls[0] = true
		case 32:
			pool.controls[29] = true
			pool.controls[27] = true
			pool.controls[28] = true
			pool.controls[1] = true
			//pool.controls[14] = true
			//pool.controls[0] = true
			//pool.controls[16] = true
			//pool.controls[21] = true
		}
		pool.controls[pool.localIndex] = true
		log.Printf("controls:%v", pool.controls)
	}
}

func (tx *Transaction) Encode() []byte {
	data, err := json.Marshal(tx)
	if err != nil {
		log.Println(err)
	}
	return data
}

func DecodeTx(data []byte) *Transaction {
	ret := &Transaction{
		Sender:    "",
		Recipient: "",
		Value:     nil,
		Nonce:     0,
	}
	err := json.Unmarshal(data, ret)

	if err != nil {
		log.Println(err)
	}
	return ret
}
