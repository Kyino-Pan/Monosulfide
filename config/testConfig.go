package config

import (
	"time"
)

var (
	ShardAmount         = 2
	IdChainConsensus    = Pbft
	IntraShardConsensus = Pbft
	CrossShardConsensus = UniRelay
	//CrossShardConsensus = Pyramid
)

const (
	EpochTime = time.Duration(999999) * time.Millisecond
	InitDelay = time.Duration(2) * time.Second

	TxInjectInterval = 50 * time.Millisecond
	InjectSpeed      = 2048  // the transaction inject speed（per message)
	TotalDataSize    = 30000 // the total number of txs

	MaxBlockSize = 2048
	BatchSize    = 16000 // supervisor read a batch of txs then send them, it should be larger than inject speed

	EnableSpy     = false // The following 2 settings are enabled only if EnableSpy is true
	SpyAtShard    = 0     // The first node in the system will be the spy.
	SpyIsMainNode = false

	TpsTest = false
)
