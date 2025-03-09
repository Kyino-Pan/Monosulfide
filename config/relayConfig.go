package config

var RelayConf = InitRelayConfig()

type RelayConfig struct {
	ShardAmount int
	Enable      bool
}

func InitRelayConfig() *RelayConfig {
	return &RelayConfig{
		ShardAmount: 8,
		Enable:      false,
	}
}
