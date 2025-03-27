package Opts

import (
	"blockEmulator/AutoTx"
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Monosulfide"
	"blockEmulator/Proposals"
	"blockEmulator/Relay"
	"blockEmulator/config"
	"blockEmulator/consensus_shard/pow"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"blockEmulator/storage"
	"log"
	"math/rand/v2"
	"strconv"
	"time"
)

type NewEpochOpt struct {
	con   Interfaces.Consensus
	block Block.Block
}

func (op *NewEpochOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	op.con.GetProposalBuffer().SetPriority(message.NewEpoch, Proposals.Emergency)
	return message.NewEpoch
}

func (op *NewEpochOpt) Schedule(...*[]byte) {
	EncodeIdBlock := new([]byte)
	Propose(op.con, message.NewEpoch, EncodeIdBlock)
	//Interfaces.Schedule(op.con,NewEpoch, newBlock.EncodeH())
}

func (op *NewEpochOpt) Prepare(vars []*[]byte) bool {
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

func (op *NewEpochOpt) Verify(vars [][]byte) bool {
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

func (op *NewEpochOpt) Execute() {
	timeStamp := time.Now()
	idChain.IDC.Append(op.block)
	if config.IdConfig.UsingPoW() {
		Interfaces.Con[config.IdMod].(*pow.NakamotoPoW).UpdateDifficulty(op.block.Head())
	}
	//idChain.RunningNode = idChain.IDC.NodeMap[idChain.RunningNode.StrKey()] // refresh self status
	idChain.RunningNode.PrintNode()
	EpochInit()
	log.Printf("--->>> update blockchain%v", time.Since(timeStamp))
	return
}

func EpochInit() {
	timeStamp := time.Now()
	inited := false
	if config.PyrConf.Enable == true {
		RefreshShard()
		inited = true
	} else if config.FideConf.Enable == true {
		RefreshFideShard()
		inited = true
	} else if config.RelayConf.Enable == true {
		RefreshRelayShard()
		inited = true
	}
	if idChain.IsIdMainNode() && inited {
		log.Println("Start packaging txs")
		go AutoTx.Manager.MsgSendingControl()
	}
	log.Printf("New main node is: %v", idChain.IDC.Main().IpAddr)
	log.Printf("---->>>>RandGen timeCost:%v", time.Since(timeStamp))
	go EpochCountDown(config.EpochTime)

	// todo
	//go Interfaces.Con[config.IdMod].EnableViewChange(config.ViewChangeTime) //ms

	return
}

func RefreshRelayShard() {
	shardAmount := config.RelayConf.ShardAmount

	shardMaps := make([]map[string]*idChain.Node, shardAmount)
	Relay.GlobalShards = make([]*Relay.Shard, config.FideConf.ShardAmount)
	// init shards
	for i := 0; i < shardAmount; i++ {
		shardMaps[i] = make(map[string]*idChain.Node)
		port, _ := strconv.Atoi(launch.Listener.GetListenPort())
		Relay.GlobalShards[i] = Relay.NewRelayShard(port, i)
	}

	// assign nodes to shards.
	randNum := idChain.IDC.GetRand()
	for strPubKey, node := range idChain.IDC.NodeMap {
		shardId, _ := crypt.HashToRange(randNum, node.NodeId, node.Port(), shardAmount)
		if node.Port() == config.ListenPort && config.EnableSpy {
			shardId = config.SpyAtShard
		}
		node.ShardID = uint64(shardId)
		shardMaps[shardId][strPubKey] = node
	}
	currShardId := idChain.IDC.NodeMap[idChain.RunningNode.StrKey()].ShardID
	Relay.LocalShard = Relay.GlobalShards[currShardId]

	// selecting main node in each shard
	for i := 0; i < shardAmount; i++ {
		if len(shardMaps[i]) == 0 {
			log.Println("WARNING::not enough nodes.")
			continue
		}
		shardI := Relay.GlobalShards[i]
		shardI.NodeMap = shardMaps[i]
		shardI.SelectMainNode()
	}

	Interfaces.Con[config.RelayMod].SetDomain(Relay.LocalShard)
	log.Printf("Nodes in shard(%v):", currShardId)
	for _, node := range Relay.GlobalShards[currShardId].NodeMap {
		log.Printf("\t%v\n", node.IpAddr)
	}
	log.Printf("Main node in shard is %v, threshold = %v",
		Relay.LocalShard.Main().IpAddr,
		Interfaces.Con[config.RelayMod].GetDomain().Threshold(),
	)

	AutoTx.Manager = AutoTx.NewTxManager(Relay.LocalShard.Chain.TxPool)
	if idChain.RunningNode == Relay.LocalShard.Main() {
		// log record
		storage.StateLogger.ResetRow(int(currShardId))
		go storage.StateLogger.Run()
		storage.StateLogger.Writef("%v", idChain.RunningNode.IpAddr)

		Interfaces.Con[config.RelayMod].Enable()
		go Interfaces.Operations[message.RelayTx].Schedule()
	}

}

func RefreshFideShard() {
	shardAmount := config.FideConf.ShardAmount
	shardMaps := make([]map[string]*idChain.Node, shardAmount)
	Monosulfide.GlobalShards = make([]*Monosulfide.Shard, config.FideConf.ShardAmount)
	// init shards
	for i := 0; i < shardAmount; i++ {
		shardMaps[i] = make(map[string]*idChain.Node)
		Monosulfide.GlobalShards[i] = Monosulfide.NewFideShard(uint64(i))
	}

	// assign nodes to shards.
	randNum := idChain.IDC.GetRand()
	for strPubKey, node := range idChain.IDC.NodeMap {
		shardId, _ := crypt.HashToRange(randNum, node.NodeId, node.Port(), shardAmount)
		if node.Port() == config.ListenPort && config.EnableSpy {
			shardId = config.SpyAtShard
		}
		node.ShardID = uint64(shardId)
		shardMaps[shardId][strPubKey] = node
	}
	currShardId := idChain.IDC.NodeMap[idChain.RunningNode.StrKey()].ShardID
	Monosulfide.LocalShard = Monosulfide.GlobalShards[currShardId]
	Monosulfide.LocalShard.Chain.EnableStorage(idChain.RunningNode.Port())

	// selecting main node in each shard
	for i := 0; i < shardAmount; i++ {
		if len(shardMaps[i]) == 0 {
			log.Println("WARNING::not enough nodes.")
			continue
		}
		shardI := Monosulfide.GlobalShards[i]
		shardI.NodeMap = shardMaps[i]
		shardI.SelectMainNode()
	}

	Interfaces.Con[config.FideMod].SetDomain(Monosulfide.LocalShard)
	log.Printf("Nodes in shard(%v):", currShardId)
	for _, node := range Monosulfide.GlobalShards[currShardId].NodeMap {
		log.Printf("\t%v\n", node.IpAddr)
	}

	log.Printf("Main node in shard is %v, threshold = %v",
		Monosulfide.LocalShard.Main().IpAddr,
		Interfaces.Con[config.FideMod].GetDomain().Threshold(),
	)

	Interfaces.Communications[Interfaces.Ping].Request() // connect to other nodes .
	AutoTx.Manager = AutoTx.NewTxManager(Monosulfide.LocalShard.Chain.TxPool)
}

func RefreshShard() {
	// constructing the pyramid shards
	shardAmount := config.PyrConf.ShardAmount
	pyramid.GlobalPyrShards = make([]*pyramid.PyrShard, shardAmount)
	shardMaps := make([]map[string]*idChain.Node, shardAmount)
	for i := 0; i < shardAmount; i++ { // init shards
		port, _ := strconv.Atoi(launch.Listener.GetListenPort())
		pyramid.GlobalPyrShards[i] = pyramid.NewPyramidShard(uint64(port), uint64(i))
		shardMaps[i] = make(map[string]*idChain.Node)
	}
	randNum := idChain.IDC.GetRand()
	for strPubKey, node := range idChain.IDC.NodeMap {
		shardId, _ := crypt.HashToRange(randNum, node.NodeId, node.Port(), shardAmount)
		if node.Port() == config.ListenPort && config.EnableSpy {
			shardId = config.SpyAtShard
		}
		node.ShardID = uint64(shardId)
		shardMaps[shardId][strPubKey] = node
	}
	for i := 0; i < shardAmount; i++ {
		// selecting main node in each shard
		if len(shardMaps[i]) == 0 {
			log.Println("WARNING::not enough nodes.")
			continue
		}
		shardI := pyramid.GlobalPyrShards[i]
		shardI.NodeMap = shardMaps[i]
		shardI.SelectMainNode()
	}
	currShardId := idChain.IDC.NodeMap[idChain.RunningNode.StrKey()].ShardID
	storage.StateLogger.ResetRow(int(currShardId))
	go storage.StateLogger.Run()
	pyramid.LocalShard = pyramid.GlobalPyrShards[currShardId]
	pyramid.LocalShard.Chain.InitOriginBlocks()
	pyramid.LocalShard = pyramid.GlobalPyrShards[currShardId]
	Interfaces.Con[config.PyrMod].SetDomain(pyramid.LocalShard)
	log.Printf("Nodes in shard(%v):", currShardId)
	for _, node := range pyramid.GlobalPyrShards[currShardId].NodeMap {
		log.Printf("\t%v\n", node.IpAddr)
	}
	log.Printf("Main node in shard is %v, threshold = %v", pyramid.LocalShard.Main().IpAddr, Interfaces.Con[config.PyrMod].GetDomain().Threshold())
	AutoTx.Manager = AutoTx.NewTxManager(pyramid.LocalShard.Chain.TxPool)

	if pyramid.IsPyrMainNode() {
		storage.StateLogger.Writef("%v", idChain.RunningNode.IpAddr)
		Interfaces.Con[config.PyrMod].Enable()
		go Interfaces.Operations[message.InternalTx].Schedule()
		if pyramid.LocalShard.IsBShard() {
			go NxtCrossTx()
		}
	}
}

func EpochCountDown(t time.Duration) {
	if config.IdConfig.UsingPbft() {
		time.Sleep(t)
		if idChain.IsIdMainNode() {
			Interfaces.Operations[message.NewEpoch].Schedule()
		}
	}
	if config.IdConfig.UsingPoW() {
		Interfaces.Operations[message.NewEpoch].Schedule()
		log.Printf("Mining on &%v", idChain.IDC.Chain.TopBlock().Hash().String())
	}
}
