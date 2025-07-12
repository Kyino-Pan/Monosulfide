package config

import (
	"os"
	"time"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

const (
	Pbft         = 0
	PoW          = 1
	ClassicRelay = 1000
	Pyramid      = 1001
	UniRelay     = 1002
)

var TX_TEST = true

const (
	EpochTime = time.Duration(2999) * time.Second
	InitDelay = time.Duration(8) * time.Second

	ExitDelay = time.Duration(8) * time.Second
	MainPort  = "20000"

	ViewChangeTime   = 80000                                      //ms
	FileInput        = `../2000000to2999999_BlockTransaction.csv` //the raw BlockTransaction data path
	DockerInput      = `./2000000to2999999_BlockTransaction.csv`
	OutputPath       = "./output/"
	PrivateKeyPEM    = "./privateKeyPEM.txt"
	PublicKeyPEM     = "./publicKeyPEM.txt"
	MsgSplitter      = byte('~')
	ViewTrigger      = "VIEW_TRIGGER"
	SleepMin         = 64
	PrefixMsgTypeLen = 40

	IdMod    = 0
	PyrMod   = 1
	FideMod  = 2
	RelayMod = 3
	BcMod    = 1024

	RealRand = false
)

var (
	Localhost           = getEnv("LOCALHOST", "127.0.0.1")
	DnsAddr             = getEnv("DNSADDR", "127.0.0.1")
	ListenPort          = getEnv("LISTEN_PORT", "20000")
	IdRunningAddr       = "IdRunningAddr"      // message will be delivered to every non-sleeping nodes.
	IdLegalAddr         = "IdLegalAddr"        // message will be delivered to EVERY node.
	PyrRunningAddr      = "pyramidRunningAddr" //
	FideRunningAddr     = "FideRunningAddr"    //
	MonoxideRunningAddr = "MonoxideRunningAddr"
	PyramidMainAddr     = "PyramidMainAddr" //
	PyramidRelateI      = "PyramidRelateI"  // related shards mainNodes

	ManagerFinished = false
	STOPPER         = make(chan bool, 1)
	SuccessByte     = []byte("successByte")
	FailByte        = []byte("failByte")
)
