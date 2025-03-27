package config

import (
	"math/big"
	"strings"
)

type PoWConfig struct {
	Difficulty   *big.Int
	UpdatePeriod uint64 // update difficulty every UpdatePeriod block.
}

func (conf *PoWConfig) Default() *PoWConfig {
	initDifficulty, ok := new(big.Int).SetString(DefaultDiff+strings.Repeat("0", 55), 16)
	if !ok {
		panic("解析初始难度失败")
	}
	conf.Difficulty = initDifficulty
	conf.UpdatePeriod = defaultUpdatePeriod
	return conf
}

var (
	DefaultDiff         = "000004000" // 16进制，越小越难
	defaultUpdatePeriod = uint64(8)
)
