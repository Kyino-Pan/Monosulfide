package crypt

import (
	"crypto/sha256"
	"math/big"
)

type Hash struct{ Hash string }

func (h Hash) Bytes() []byte {
	if h.Hash == "" {
		return nil
	}
	return []byte(h.Hash)
}

func (h Hash) String() string {
	return h.Hash
}

func NewHash(byteHash []byte) *Hash {
	return &Hash{
		Hash: string(byteHash),
	}
}

func EmptyHash() *Hash {
	return &Hash{Hash: ""}
}

func doubleSHA256(blockHeaderBytes []byte) [32]byte {
	firstHash := sha256.Sum256(blockHeaderBytes)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash
}

func IsValidBlock(blockHeaderBytes []byte, target *big.Int) bool {
	hash := doubleSHA256(blockHeaderBytes)
	// 将哈希值转换为大整数（注意：比特币中的哈希通常以小端格式存储，转换时要留意字节顺序）
	hashInt := new(big.Int).SetBytes(hash[:])
	// 比较计算结果和目标值
	return hashInt.Cmp(target) < 0
}

// BigToCompact 将 target（*big.Int）转换为比特币中的压缩表示（bits）。
func BigToCompact(target *big.Int) uint32 {
	// 如果目标值为 0，直接返回 0
	if target.Sign() == 0 {
		return 0
	}

	// 计算 target 需要多少字节来表示
	size := uint32((target.BitLen() + 7) / 8)
	var compact uint32

	// 拷贝一份 target，避免修改原值
	tmp := new(big.Int).Set(target)

	if size <= 3 {
		// 如果 target 占用的字节数不足 3 字节，将其左移以填充 3 字节
		compact = uint32(tmp.Uint64() << (8 * (3 - size)))
	} else {
		// 如果超过 3 字节，右移使其只保留最高的 3 字节
		tmp.Rsh(tmp, uint(8*(size-3)))
		compact = uint32(tmp.Uint64())
	}

	// 如果 compact 的最高有效位（第 24 位）被置位，
	// 需要右移 8 位，并使 size 自增，防止溢出
	if (compact & 0x00800000) != 0 {
		compact >>= 8
		size++
	}

	// 将 size 存入 compact 的最高 8 位
	compact |= size << 24

	return compact
}

// CompactToBig 将压缩表示（bits）还原为目标值（*big.Int）。
func CompactToBig(compact uint32) *big.Int {
	// compact 的最高 8 位为 size，剩下 23 位为系数
	size := uint32(compact >> 24)
	word := compact & 0x007fffff

	bn := new(big.Int)
	if size <= 3 {
		// 当 size 小于等于 3 时，需要右移以恢复原始值
		bn.SetUint64(uint64(word) >> (8 * (3 - size)))
	} else {
		// 否则将 word 左移恢复原始字节数
		bn.SetUint64(uint64(word))
		bn.Lsh(bn, uint(8*(size-3)))
	}
	return bn
}
