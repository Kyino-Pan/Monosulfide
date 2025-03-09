package blockchain

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
)

// 自定义rsa.PublicKey的排序，因为它本身不能直接排序
type PublicKeySlice []*rsa.PublicKey

func (pks PublicKeySlice) Len() int {
	return len(pks)
}

func (pks PublicKeySlice) Less(i, j int) bool {
	// 将公钥的N值（模数）转换为字符串进行比较
	return pks[i].N.String() < pks[j].N.String()
}

func (pks PublicKeySlice) Swap(i, j int) {
	pks[i], pks[j] = pks[j], pks[i]
}

// HashToRange hashes a uint64 and rsa.PublicKey to a number in the range (0, n]

func HashToRange(value uint64, key *rsa.PublicKey, n int) (int, error) {
	// Convert uint64 to bytes
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, value)

	// Serialize the RSA PublicKey
	// Assuming the PublicKey consists of the modulus (N) and the public exponent (E)
	keyBytes := key.N.Bytes() // The modulus N
	expBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(expBytes, uint32(key.E))

	// Concatenate all bytes together
	data := append(buf, keyBytes...)
	data = append(data, expBytes...)

	// Hash the bytes
	hash := sha256.Sum256(data)

	// Convert the Hash to a big integer
	hashInt := new(big.Int).SetBytes(hash[:])

	// Reduce the integer to the range [0, n)
	mod := big.NewInt(int64(n))
	result := new(big.Int).Mod(hashInt, mod)

	// Convert the result to an int
	return int(result.Int64()), nil
}
