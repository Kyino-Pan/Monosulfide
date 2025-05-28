package config

import (
	"time"
)

var (
	ShardAmount         = 4
	IdChainConsensus    = Pbft
	IntraShardConsensus = Pbft
	CrossShardConsensus = UniRelay
	//CrossShardConsensus = ClassicRely
	//CrossShardConsensus = Pyramid
)

const (
	TxInjectInterval = 50 * time.Millisecond
	InjectSpeed      = 2048     // 每秒交易注入速率 (transactions per second)
	TotalDataSize    = 10000000 // the total number of txs

	MaxBlockSize = 2048
	BatchSize    = 16000 // supervisor read a batch of txs then send them, it should be larger than inject speed

	EnableSpy     = false // The following 2 settings are enabled only if EnableSpy is true
	SpyAtShard    = 0     // The first node in the system will be the spy.
	SpyIsMainNode = false

	TpsTest = false
)
