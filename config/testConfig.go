package config

import (
	"time"
)

var (
	ShardAmount         = 8
	IdChainConsensus    = Pbft
	IntraShardConsensus = Pbft
	//CrossShardConsensus = UniRelay
	CrossShardConsensus = ClassicRelay
	//CrossShardConsensus = Pyramid
)

const (
	TxInjectInterval = 50 * time.Millisecond
	InjectSpeed      = 2000 // per second

	// 每秒交易注入速率 (transactions per second)
	TotalDataSize = 100000 // the total number of txs
	PoWExpTime    = 3000 * time.Millisecond

	MaxBlockSize = 2048
	BatchSize    = 16000 // supervisor read a batch of txs then send them, it should be larger than inject speed

	EnableSpy     = false // The following 2 settings are enabled only if EnableSpy is true
	SpyAtShard    = 0     // The first node in the system will be the spy.
	SpyIsMainNode = false

	TpsTest = false
)
