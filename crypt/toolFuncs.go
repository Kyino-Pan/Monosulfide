package crypt

import (
	"blockEmulator/config"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"strconv"
)

func UintToBytes(num uint64) *[]byte {
	// 创建一个足够存放uint64类型数据的字节切片（uint64是8字节）
	buf := make([]byte, 8)

	// 将uint64数据编码到字节切片中（大端模式）
	binary.BigEndian.PutUint64(buf, num)
	return &buf
}

func BytesToUint(buf []byte) uint64 {
	decodedNum := binary.BigEndian.Uint64(buf)
	return decodedNum
}

func HashToRange(value uint64, key *rsa.PublicKey, p string, n int) (int, error) {
	// Convert uint64 to bytes
	if config.RealRand {
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
	} else {
		port, _ := strconv.Atoi(p)
		return (port / 10) % n, nil
	}
}
