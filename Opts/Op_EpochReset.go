package Opts

import (
	"blockEmulator/AutoTx"
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Monosulfide"
	"blockEmulator/Proposals"
	"blockEmulator/config"
	"blockEmulator/consensus_shard/pow"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"blockEmulator/monoxide"
	"blockEmulator/pyramid"
	"blockEmulator/storage"
	"log"
	"math/rand/v2"
	"strconv"
	"time"
)

type EpochResetOpt struct {
	con   Interfaces.Consensus
	block Block.Block
}

func (op *EpochResetOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	op.con.GetProposalBuffer().SetPriority(message.EpochReset, Proposals.Now)
	return message.EpochReset
}

func (op *EpochResetOpt) Schedule(...*[]byte) {
	EncodeIdBlock := new([]byte)
	Propose(op.con, message.EpochReset, EncodeIdBlock)
	//Interfaces.Schedule(op.con,EpochReset, newBlock.EncodeH())
}

func (op *EpochResetOpt) Prepare(vars []*[]byte) bool {
	idChain.IDC.ActivateAll()
	random := rand.Uint64()
	randByte := crypt.UintToBytes(random)
	newBlock := idChain.IDC.Chain.GenerateIdBlock(randByte)
	if config.IdConfig.UsingPoW() {
		for !crypt.IsValidBlock(newBlock.Head().EncodeH(), config.IdConfig.PowConf.Difficulty) {
			newBlock = idChain.IDC.Chain.GenerateIdBlock(randByte)
		}
	}
	byteBlock := newBlock.Encode()
	vars[0] = &byteBlock
	op.Verify(Interfaces.TransVars(vars))
	return true
}

func (op *EpochResetOpt) Verify(vars [][]byte) bool {
	block := new(Block.StdBlock).Decode(vars[0])
	if idChain.RunningNode.IsPreparing() {
		op.block = block
		return true
	}
	result := idChain.IDC.Chain.Verify(block)
	if result == true {
		op.block = block
		for _, tx := range block.Body().Txs() {
			id := tx.Sender
			addr := tx.Recipient
			idChain.IDC.AppendNode(addr, crypt.DecodePublicKey([]byte(id)))
		}
		idChain.IDC.ActivateAll()
		return true
	} else {
		log.Printf("ID block error")
		return false
	}
}

func (op *EpochResetOpt) Execute() {
	timeStamp := time.Now()
	log.Println("EpochResetOpt execute begin")
	idChain.IDC.Append(op.block)
	if config.IdConfig.UsingPoW() {
		Interfaces.Con[config.IdMod].(*pow.NakamotoPoW).UpdateDifficulty(op.block.Head())
	}
	idChain.RunningNode.PrintNode()
	EpochInit()
	go EpochCountDown(config.EpochTime)
	log.Printf("--->>> update blockchain%v", time.Since(timeStamp))
	return
}

func CrossBegin() {
	if config.EnableDelayTable {
		config.NDelay = config.DT[int(idChain.RunningNode.ShardID)]
	}
	if idChain.RunningNode == Interfaces.LocalShard.Main() || config.MonoxideConf.Enable || config.FideConf.Enable {
		Interfaces.Communications[Interfaces.MainBegin].Request()
	}
}

func EpochInit() {
	timeStamp := time.Now()
	inited := true
	sNum := config.ShardAmount
	shardMaps := make([]map[string]*idChain.Node, config.ShardAmount)
	globalShards := make([]Interfaces.Domain, sNum)
	conId := -1
	for i := 0; i < sNum; i++ {
		shardMaps[i] = make(map[string]*idChain.Node)
		port, _ := strconv.Atoi(launch.Listener.GetListenPort())
		switch config.CrossShardConsensus {
		case config.Pyramid:
			globalShards[i] = new(pyramid.Shard)
			conId = config.PyrMod
		case config.UniRelay:
			globalShards[i] = new(Monosulfide.Shard)
			conId = config.FideMod
		case config.ClassicRelay:
			globalShards[i] = new(monoxide.Shard)
			conId = config.RelayMod
		default:
			log.Printf("Warning::No cross-consensus")
			inited = false
		}
		globalShards[i].Reset(port, i)
	}
	Interfaces.LocalShard = RDomain(globalShards, config.ShardAmount)
	Interfaces.Con[conId].SetDomain(Interfaces.LocalShard)
	Interfaces.GlobalShards = globalShards

	switch config.CrossShardConsensus {
	case config.Pyramid:
		pyramid.LocalShard = Interfaces.LocalShard.(*pyramid.Shard)
		pyramid.GlobalShards = make([]*pyramid.Shard, sNum)
		for i, s := range Interfaces.GlobalShards {
			pyramid.GlobalShards[i] = s.(*pyramid.Shard)
		}
	case config.UniRelay:
		Monosulfide.LocalShard = Interfaces.LocalShard.(*Monosulfide.Shard)
		Monosulfide.GlobalShards = make([]*Monosulfide.Shard, sNum)
		for i, s := range Interfaces.GlobalShards {
			Monosulfide.GlobalShards[i] = s.(*Monosulfide.Shard)
		}
	case config.ClassicRelay:
		monoxide.LocalShard = Interfaces.LocalShard.(*monoxide.Shard)
		monoxide.GlobalShards = make([]*monoxide.Shard, sNum)
		for i, s := range Interfaces.GlobalShards {
			monoxide.GlobalShards[i] = s.(*monoxide.Shard)
		}
	default:
		log.Printf("Warning::No cross-consensus")
	}
	if idChain.IsIdMainNode() && inited {
		log.Println("Start packaging txs")
		go AutoTx.Manager.MsgSendingControl()
	}
	log.Printf("New main node is: %v", idChain.IDC.Main().IpAddr)
	log.Printf("---->>>>RandGen timeCost:%v", time.Since(timeStamp))

	//go Interfaces.Con[config.IdMod].EnableViewChange(config.ViewChangeTime) //ms
	return
}

func RDomain[S Interfaces.Domain](globalShards []S, conId int) S {
	// assign nodes to shards.
	sNum := config.ShardAmount
	shardMaps := make([]map[string]*idChain.Node, config.ShardAmount)
	for i := 0; i < sNum; i++ {
		shardMaps[i] = make(map[string]*idChain.Node)
	}
	randNum := idChain.IDC.GetRand()
	for strPubKey, node := range idChain.IDC.NodeMap {
		shardId, _ := crypt.HashToRange(randNum, node.NodeId, node.Port(), sNum)
		if node.Port() == config.MainPort && config.EnableSpy {
			shardId = config.SpyAtShard
		}
		node.ShardID = uint64(shardId)
		shardMaps[shardId][strPubKey] = node
	}
	currShardId := idChain.IDC.NodeMap[idChain.RunningNode.StrKey()].ShardID
	localShard := globalShards[currShardId]

	// selecting main node in each shard
	for i := 0; i < sNum; i++ {
		if len(shardMaps[i]) == 0 {
			log.Println("WARNING::not enough nodes.")
			continue
		}
		shardI := globalShards[i]
		shardI.SetMap(shardMaps[i])
		shardI.SelectMain()
	}
	log.Printf("Nodes in shard(%v):", currShardId)
	for _, node := range globalShards[currShardId].GetMap() {
		log.Printf("\t%v\n", node.IpAddr)
	}

	log.Printf("Main node in shard is %v, threshold = %v",
		localShard.Main().IpAddr,
		localShard.Threshold(),
	)
	AutoTx.Manager = AutoTx.NewTxManager(localShard.GetTxPool())
	if idChain.RunningNode == localShard.Main() {
		// log record
		storage.StateLogger.ResetRow(int(currShardId))
		go storage.StateLogger.Run()
		storage.StateLogger.Writef("%v", idChain.RunningNode.IpAddr)
	}
	return localShard
}

func EpochCountDown(t time.Duration) {
	time.Sleep(t)
	if config.IdConfig.UsingPbft() {
		if idChain.IsIdMainNode() {
			Interfaces.Operations[message.EpochReset].Schedule()
		}
	} else if config.IdConfig.UsingPoW() {
		Interfaces.Operations[message.EpochReset].Schedule()
		log.Printf("Mining on &%v", idChain.IDC.Chain.TopBlock().Hash().String())
	}
}

//func RefreshPyrShard() {
//	// constructing the pyramid shards
//	shardAmount := config.PyrConf.ShardAmount
//	pyramid.GlobalShards = make([]*pyramid.Shard, config.PyrConf.ShardAmount)
//	shardMaps := make([]map[string]*idChain.Node, shardAmount)
//	for i := 0; i < shardAmount; i++ { // init shards
//		port, _ := strconv.Atoi(launch.Listener.GetListenPort())
//		pyramid.GlobalShards[i] = pyramid.NewPyramidShard(uint64(port), uint64(i))
//		shardMaps[i] = make(map[string]*idChain.Node)
//	}
//	randNum := idChain.IDC.GetRand()
//	for strPubKey, node := range idChain.IDC.NodeMap {
//		shardId, _ := crypt.HashToRange(randNum, node.NodeId, node.Port(), shardAmount)
//		if node.Port() == config.MainPort && config.EnableSpy {
//			shardId = config.SpyAtShard
//		}
//		node.ShardID = uint64(shardId)
//		shardMaps[shardId][strPubKey] = node
//	}
//	for i := 0; i < shardAmount; i++ {
//		// selecting main node in each shard
//		if len(shardMaps[i]) == 0 {
//			log.Println("WARNING::not enough nodes.")
//			continue
//		}
//		shardI := pyramid.GlobalShards[i]
//		shardI.NodeMap = shardMaps[i]
//		Interfaces.SelectMain(shardI)
//	}
//	currShardId := idChain.IDC.NodeMap[idChain.RunningNode.StrKey()].ShardID
//	storage.StateLogger.ResetRow(int(currShardId))
//	go storage.StateLogger.Run()
//	pyramid.LocalShard = pyramid.GlobalShards[currShardId]
//	Interfaces.Con[config.PyrMod].SetDomain(pyramid.LocalShard)
//	log.Printf("Nodes in shard(%v):", currShardId)
//	for _, node := range pyramid.GlobalShards[currShardId].NodeMap {
//		log.Printf("\t%v\n", node.IpAddr)
//	}
//	log.Printf("Main node in shard is %v, threshold = %v",
//		pyramid.LocalShard.Main().IpAddr,
//		Interfaces.Con[config.PyrMod].GetDomain().Threshold(),
//	)
//	AutoTx.Manager = AutoTx.NewTxManager(pyramid.LocalShard.Chain.TxPool)
//
//	if pyramid.IsPyrMainNode() {
//		storage.StateLogger.Writef("%v", idChain.RunningNode.IpAddr)
//		Interfaces.Con[config.PyrMod].Enable()
//		go Interfaces.Operations[message.InternalTx].Schedule()
//		if pyramid.LocalShard.IsBShard() {
//			go NxtCrossTx()
//		}
//	}
//}
//
