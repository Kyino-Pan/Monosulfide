package config

var (
	SpiConf           *SpiralConfig
	SpiralShardAmount = 8
)

const ClassRelay = false

func InitSpiralConfig() *SpiralConfig {
	return &SpiralConfig{
		Enable:      true,
		ShardAmount: SpiralShardAmount,
		Threshold:   SpiralShardAmount - (SpiralShardAmount+2)/3 + 1,
	}
}

type SpiralConfig struct {
	Enable      bool
	ShardAmount int
	Threshold   int
}
