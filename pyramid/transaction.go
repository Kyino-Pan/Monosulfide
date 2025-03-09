package pyramid

import (
	"blockEmulator/config"
	"blockEmulator/crypt"
	_ "blockEmulator/networks"
	"log"
	"math/big"
	"strconv"
)

type Transaction struct {
	Sender   string
	Receiver string
	Value    *big.Int
	Nonce    uint64
}

func NewTransaction(sender, recipient string, value *big.Int, nonce uint64) *Transaction {
	return &Transaction{
		Sender:   sender,
		Receiver: recipient,
		Value:    value,
		Nonce:    nonce,
	}
}

func (tx *Transaction) Hash() crypt.Hash {
	return crypt.Hash{Hash: string(crypt.GetDigest(tx))}
}

func (tx *Transaction) SInShard() int {
	addr := tx.Sender
	last16Addr := addr[len(addr)-8:]
	num, err := strconv.ParseUint(last16Addr, 16, 64)
	if err != nil {
		log.Panic(err)
	}
	return int(num) % config.PyrConf.ShardAmount
}

func (tx *Transaction) RInShard() int {
	addr := tx.Receiver
	last16Addr := addr[len(addr)-8:]
	num, err := strconv.ParseUint(last16Addr, 16, 64)
	if err != nil {
		log.Panic(err)
	}
	return int(num) % config.PyrConf.ShardAmount
}

func (tx *Transaction) RelatedShards() []int {
	s := tx.SInShard()
	r := tx.RInShard()
	ret := make([]int, 0)
	for i := 0; i < config.PyrConf.ShardAmount; i++ {
		if config.PyrConf.ShardDistribution[i][r] || config.PyrConf.ShardDistribution[i][s] {
			//if r == s && s != i {
			// // this will skip relate i-shard's internal txs.
			//	continue
			//}
			ret = append(ret, i)
		}
	}
	return ret
}
