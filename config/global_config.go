package config

import (
	"time"
)

const (
	Pbft        = 0
	PoW         = 1
	ClassicRely = 1000
	Pyramid     = 1001
	UniRelay    = 1002
)

const (
	ExitDelay = time.Duration(3) * time.Second

	ViewChangeTime   = 80000                                      //ms
	FileInput        = `../2000000to2999999_BlockTransaction.csv` //the raw BlockTransaction data path
	Localhost        = "127.0.0.1"
	OutputPath       = "./output/"
	DnsAddr          = "127.0.0.1"
	ListenPort       = "20000"
	IdRunningAddr    = "IdRunningAddr"      // message will be delivered to every non-sleeping nodes.
	IdLegalAddr      = "IdLegalAddr"        // message will be delivered to EVERY node.
	PyrRunningAddr   = "pyramidRunningAddr" //
	FideRunningAddr  = "FideRunningAddr"    //
	PyramidMainAddr  = "PyramidMainAddr"    //
	PyramidRelateI   = "PyramidRelateI"     // related shards mainNodes
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
	ManagerFinished = false
	STOPPER         = make(chan bool, 1)
	SuccessByte     = []byte("successByte")
	FailByte        = []byte("failByte")
)
