package config

var MonoxideConf = InitMonoxideConfig()

type MonoxideConfig struct {
	ShardAmount int
	Enable      bool
}

func InitMonoxideConfig() *MonoxideConfig {
	ret := &MonoxideConfig{
		ShardAmount: ShardAmount,
		Enable:      false,
	}
	if CrossShardConsensus == ClassicRelay {
		ret.Enable = true
	}
	return ret
}
