package config

import "time"

// 流量计算
var (
	CalcComm = false
	TxBegin  time.Time
	CommCalc = 0.0 //KB
)

// 网络延迟配置
var (
	// NDelay 在DT未配置时的默认网络延迟时间
	NDelay   = time.Duration(0) * time.Millisecond // ms
	DT       = make(map[int]time.Duration)         // DT[i]为shard i的延迟时间
	EnableDT = true
)

func InitNetDelay() {
	DT[0] = 100 * time.Millisecond
	DT[1] = 100 * time.Millisecond
	DT[2] = 100 * time.Millisecond
	DT[3] = 100 * time.Millisecond
	DT[4] = 100 * time.Millisecond
	DT[5] = 100 * time.Millisecond
	DT[6] = 100 * time.Millisecond
	DT[7] = 100 * time.Millisecond
	DT[8] = 100 * time.Millisecond
	DT[9] = 100 * time.Millisecond
	DT[10] = 100 * time.Millisecond
	DT[11] = 100 * time.Millisecond
	DT[12] = 100 * time.Millisecond
	DT[13] = 100 * time.Millisecond
	DT[14] = 100 * time.Millisecond
	DT[15] = 100 * time.Millisecond
	DT[16] = 100 * time.Millisecond
}
