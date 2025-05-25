package config

var RelayConf = InitRelayConfig()

type RelayConfig struct {
	ShardAmount int
	Enable      bool
}

func InitRelayConfig() *RelayConfig {
	ret := &RelayConfig{
		ShardAmount: 8,
		Enable:      false,
	}
	if CrossShardConsensus == ClassicRely {
		ret.Enable = true
	}
	return ret
}
