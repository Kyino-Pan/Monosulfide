package Tx

import (
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/message"
	"crypto/sha256"
	"log"
	"math/big"
	"strconv"
	"time"
)

var SendTxs = message.MessageType(message.PyrPrefix + "SendTxs")
var (
	RegisterTx = 0
	PryTx      = 1
	RelayTx    = 2
)

type Transaction struct {
	Sender    string
	Recipient string
	Value     *big.Int
	Nonce     uint64
	Type      int
	Interface []byte
	Time      time.Time
}

func GenMPTRoot(hashes [][]byte) []byte {
	// 构建默克尔树
	if hashes == nil || len(hashes) == 0 {
		return nil
	}
	for len(hashes) > 1 {
		// 如果数量为奇数，重复最后一个哈希值补足
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}
		// 合并每对哈希值并计算其哈希
		var newLevel [][]byte
		for i := 0; i < len(hashes); i += 2 {
			combined := append(hashes[i], hashes[i+1]...)
			newHash := sha256.Sum256(combined)
			newLevel = append(newLevel, newHash[:])
		}
		hashes = newLevel
	}
	return hashes[0]
}

func GenTxRoot(txs []*Transaction) []byte {
	// 提取交易哈希值
	var hashes [][]byte
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash().Bytes())
	}
	return GenMPTRoot(hashes)
}

func NewTransaction(sender, recipient string, value *big.Int, nonce uint64, txType int) *Transaction {
	return &Transaction{
		Sender:    sender,
		Recipient: recipient,
		Value:     value,
		Nonce:     nonce,
		Type:      txType,
		Interface: nil,
	}
}

func GenerateRegisterTx(id, addr string, nonce uint64) *Transaction {
	return NewTransaction(id, addr, nil, nonce, RegisterTx)
}

func (tx *Transaction) Hash() crypt.Hash {
	return *crypt.NewHash(crypt.GetDigest(tx))
}

func (tx *Transaction) RInShard() int {
	if tx.Type == RegisterTx {
		return 0
	}
	return _inShard_(tx.Recipient)
}

func (tx *Transaction) SInShard() int {
	if tx.Type == RegisterTx {
		return 0
	}
	return _inShard_(tx.Sender)
}

func _inShard_(addr string) int {
	last16Addr := addr[len(addr)-8:]
	num, err := strconv.ParseUint(last16Addr, 16, 64)
	if err != nil {
		log.Panic(err)
	}
	if config.PyrConf.Enable {
		return int(num) % config.PyrConf.ShardAmount
	} else if config.FideConf.Enable {
		return int(num) % config.FideConf.ShardAmount
	} else if config.RelayConf.Enable {
		return int(num) % config.RelayConf.ShardAmount
	} else {
		log.Printf("WARNING::tx belonging is asked without config")
		return 0
	}
}

func (tx *Transaction) RelatedShards() []int {
	s := tx.SInShard()
	r := tx.RInShard()
	ret := make([]int, 0)
	if config.PyrConf.Enable {
		for i := 0; i < config.PyrConf.ShardAmount; i++ {
			if config.PyrConf.ShardDistribution[i][r] || config.PyrConf.ShardDistribution[i][s] {
				// 如果是i is b-shard且控制控制r或s
				ret = append(ret, i)
			} else if config.PyrConf.InRoute(i, r, s) {
				// 如果i在r->s的relay路径上
				ret = append(ret, i)
			}
		}
	} else if config.FideConf.Enable {
		ret = append(ret, s)
		//if s != r {
		//	ret = append(ret, r)
		//}
	}
	return ret
}

func SplitRelay(rtx *Transaction) []*Transaction {
	path := config.PyrConf.GetRoute(rtx.SInShard(), rtx.RInShard())
	rtxs := make([]*Transaction, 0)
	for i := 1; i < len(path); i++ {
		curr := path[i]
		pre := path[i-1]
		rtxs = append(rtxs, genRtx(rtx, pre, curr))
	}
	return rtxs
}

func genRtx(rtx *Transaction, pre int, sid int) *Transaction {
	adds := config.PyrConf.DefaultAddr
	return NewTransaction(adds[pre], adds[sid], rtx.Value, rtx.Nonce, RelayTx)
}

func IsVaildTxSet(all []*Transaction) bool {
	//todo
	return true
}
