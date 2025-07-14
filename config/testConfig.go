package config

import (
	"time"
)

var (
	ShardAmount         = 16
	IdChainConsensus    = Pbft
	IntraShardConsensus = Pbft
	CrossShardConsensus = UniRelay
	//CrossShardConsensus = ClassicRelay
	//CrossShardConsensus = Pyramid
)
var PoWExpTime = 2000 * time.Millisecond

var phi = 1.0

const (
	TxInjectInterval = 50 * time.Millisecond

	// 每秒交易注入速率 (transactions per second)
	TotalDataSize = 100000 // the total number of txs

	BatchSize = 16000 // supervisor read a batch of txs then send them, it should be larger than inject speed

	EnableSpy     = false // The following 2 settings are enabled only if EnableSpy is true
	SpyAtShard    = 0     // The first node in the system will be the spy.
	SpyIsMainNode = false

	TpsTest = false
)

var MaxBlockSize = 4196
var InjectSpeed = float64(MaxBlockSize) / float64(PoWExpTime/time.Second) * phi // per second
// var InjectSpeed = 10000
