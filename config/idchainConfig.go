package config

type IdChainConfig struct {
	IntraConsensus int
	PowConf        *PoWConfig
	ChainId        int
}

func (c IdChainConfig) UsingPoW() bool {
	return c.IntraConsensus == PoW
}

func (c IdChainConfig) UsingPbft() bool {
	return c.IntraConsensus == Pbft
}

var IdConfig = NewIdChainConfig()

func NewIdChainConfig() *IdChainConfig {
	ret := &IdChainConfig{
		IntraConsensus: IdChainConsensus,
		PowConf:        new(PoWConfig).Default(),
		ChainId:        10086,
	}
	return ret
}
