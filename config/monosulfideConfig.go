package config

var (
	FideConf        *FideConfig
	FideShardAmount = 1
)

const ClassRelay = false

func InitFideConfig() *FideConfig {
	return &FideConfig{
		Enable:      true,
		ShardAmount: FideShardAmount,
		Threshold:   FideShardAmount - (FideShardAmount+2)/3 + 1,
	}
}

type FideConfig struct {
	Enable      bool
	ShardAmount int
	Threshold   int
}
