package config

import "time"

const (
	Debugging        = true
	DebugNodeAtShard = 0
	DebugIsMainNode  = false
	TPS_TEST         = false
)

var TxBegin time.Time

// 4个shard，S1>S2>S
