package config

import "time"

const (
	Debugging        = false
	DebugNodeAtShard = 0
	DebugIsMainNode  = false
	TPS_TEST         = false
)

var (
	CalcComm = false
	TxBegin  time.Time
	CommCalc = 0.0 //KB
)
