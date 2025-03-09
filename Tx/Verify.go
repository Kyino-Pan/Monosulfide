package Tx

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
	"log"
	"math/big"
)

// 假设账户结构包含余额、Nonce
type Account struct {
	Balance *big.Int
	Nonce   uint64
}

// IsValid 作为交易校验的入口
// 需要传入一个 *trie.Database，用来读写账户状态；
// go-ethereum/trie.Database 的定义可参考 go-ethereum 源码

// GetOrCreateAccount 从 Trie 数据库中获取指定地址的账户；若不存在则初始化一个
func GetOrCreateAccount(db *trie.Database, blockHash []byte, addr common.Address) (*Account, error) {
	acct, err := readAccountFromTrie(db, addr)
	if err != nil && err.Error() != "not found" {
		// 如果读取时出现了真正的读写错误，应当返回
		return nil, err
	}
	if acct == nil {
		// 说明数据库中找不到这个地址的账户，新建并写入
		acct = &Account{
			Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil), // 1e20
			Nonce:   0,
		}
		st, err := trie.New(trie.TrieID(common.BytesToHash(blockHash)), db)
		if err != nil {
			return nil, err
		}
		err = writeAccountToTrie(st, addr.Bytes(), acct)
		if err != nil {
			return nil, err
		}
	}
	return acct, nil
}

// readAccountFromTrie 从 *trie.Database 中读取并 gob 解码指定地址的账户
func readAccountFromTrie(db *trie.Database, addr common.Address) (*Account, error) {
	// 通过 addr.Hash() 作为 key，从底层数据库取出序列化数据
	encoded, err := db.Node(addr.Hash())
	if err != nil {
		return nil, err
	}
	if encoded == nil {
		// 数据不存在：此处返回 nil, nil 表示“读不到账户”
		return nil, nil
	}
	// 使用 gob 解码
	var acct Account
	dec := gob.NewDecoder(bytes.NewReader(encoded))
	if err := dec.Decode(&acct); err != nil {
		return nil, err
	}
	return &acct, nil
}

// writeAccountToTrie 将账户用 gob 编码后写入 *trie.Database
func writeAccountToTrie(trie *trie.Trie, accAddr []byte, acct *Account) error {
	// gob 编码
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(acct); err != nil {
		return err
	}
	// 使用地址的哈希作为 key，写入底层数据库
	err := trie.Update(accAddr, acct.Balance.Bytes())
	if err != nil {
		log.Println(err)
	}
	return err
}

// CheckNonce 用于检查发送方交易的 Nonce 是否符合预期
func CheckNonce(acct *Account, txNonce uint64) error {
	if txNonce != acct.Nonce {
		return fmt.Errorf("nonce mismatch: expected=%d, got=%d", acct.Nonce, txNonce)
	}
	return nil
}

// CheckBalance 用于检查账户余额是否足够
func CheckBalance(acct *Account, value *big.Int) error {
	if acct.Balance.Cmp(value) < 0 {
		return errors.New("balance mismatch")
	}
	return nil
}
