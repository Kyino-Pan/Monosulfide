package config

var (
	FideConf *FideConfig
)

const ClassRelay = false

func InitFideConfig() *FideConfig {
	ret := new(FideConfig)
	if CrossShardConsensus == UniRelay {
		ret.Enable = true
	}
	return ret.UpdateShardAmount(ShardAmount)
}

func (conf *FideConfig) UpdateShardAmount(num int) *FideConfig {
	conf.ShardAmount = num
	conf.Threshold = num - (num+2)/3 + 1
	return conf
}

type FideConfig struct {
	Enable      bool
	ShardAmount int
	Threshold   int
}
