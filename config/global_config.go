package config

import "time"

const (
	InjectSpeed   = 2048  // the transaction inject speed（per message)
	TotalDataSize = 30000 // the total number of txs
	BlockSize     = 2048
	BatchSize     = 16000 // supervisor read a batch of txs then send them, it should be larger than inject speed

	EpochTime        = time.Duration(999999) * time.Millisecond // ms
	InitDelay        = time.Duration(4) * time.Second           // s
	TxInjectInterval = 50 * time.Millisecond
	ExitDelay        = time.Duration(3) * time.Second

	ViewChangeTime   = 80000                                      //ms
	FileInput        = `../2000000to2999999_BlockTransaction.csv` //the raw BlockTransaction data path
	Localhost        = "127.0.0.1"
	OutputPath       = "./output/"
	DnsAddr          = "127.0.0.1"
	ListenPort       = "20000"
	MonitorAddr      = Localhost + ":" + ListenPort
	IdRunningAddr    = "IdRunningAddr"      // message will be delivered to every non-sleeping nodes.
	IdLegalAddr      = "IdLegalAddr"        // message will be delivered to EVERY node.
	PyrRunningAddr   = "pyramidRunningAddr" //
	FideRunningAddr  = "FideRunningAddr"    //
	PyramidMainAddr  = "PyramidMainAddr"    //
	PyramidRelateI   = "PyramidRelateI"     // related shards mainNodes
	PrivateKeyPEM    = "./privateKeyPEM.txt"
	PublicKeyPEM     = "./publicKeyPEM.txt"
	ShardPrefix      = "ShardPrefix"
	MsgSplitter      = byte('~')
	ViewTrigger      = "VIEW_TRIGGER"
	SleepMin         = 64
	PrefixMsgTypeLen = 40
	IdMod            = 0
	PyrMod           = 1
	FideMod          = 2
	RelayMod         = 3
	EoFDelay         = 16 * time.Second
	RealRand         = false
)

var (
	SuccessByte     = []byte("successByte")
	FailByte        = []byte("failByte")
	ManagerFinished = false
	STOPPER         = make(chan bool, 1)
)
