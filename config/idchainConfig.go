package config

import (
	"math/big"
	"strings"
)

type IDChainConfig struct {
	UsingPBFT    bool
	UsingPoW     bool
	Difficulty   *big.Int
	UpdatePeriod uint64
}

var IdConfig = NewIDChainConfig()

func NewIDChainConfig() *IDChainConfig {
	initDiffStr := "000004" + strings.Repeat("0", 58)
	initDifficulty, ok := new(big.Int).SetString(initDiffStr, 16)
	if !ok {
		panic("解析初始难度失败")
	}
	return &IDChainConfig{
		Difficulty:   initDifficulty,
		UsingPBFT:    false,
		UsingPoW:     true,
		UpdatePeriod: 8, // 每UpdatePeriod个块更新一次diff
	}
}
