package Opts

import (
	"blockEmulator/AutoTx"
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/Relay"
	"blockEmulator/Spiral"
	"blockEmulator/config"
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

func (op *NewEpochOpt) Propose(...*[]byte) {
	EncodeIdBlock := new([]byte)
	op.con.Propose(message.NewEpoch, EncodeIdBlock)
	//op.con.Propose(NewEpoch, newBlock.EncodeH())
}

func (op *NewEpochOpt) PrepareAfterLock(vars []*[]byte) bool {
	idChain.IDC.ActivateAll()
	random := rand.Uint64()
	randByte := crypt.UintToBytes(random)
	newBlock := idChain.IDC.Chain.GenerateIdBlock(randByte)
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
	result := idChain.IDC.Chain.Verify(block) //todo
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
	//idChain.RunningNode = idChain.IDC.NodeMap[idChain.RunningNode.StrKey()] // refresh self status
	idChain.RunningNode.PrintNode()
	EpochInit()
	log.Printf("--->>> update blockchain%v", time.Since(timeStamp))
	return
}

func EpochInit() {
	timeStamp := time.Now()
	if config.PyrConf.Enable == true {
		RefreshShard()
	} else if config.SpiConf.Enable == true {
		RefreshSpiralShard()
	} else if config.RelayConf.Enable == true {
		RefreshRelayShard()
	}
	if idChain.IsIdMainNode() {
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
	Relay.GlobalShards = make([]*Relay.Shard, config.SpiConf.ShardAmount)
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
		if node.Port() == config.ListenPort && config.Debugging {
			shardId = config.DebugNodeAtShard
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

		Interfaces.Con[config.RelayMod].EnablePropose()
		go Interfaces.Operations[message.RelayTx].Propose()
	}

}

func RefreshSpiralShard() {
	shardAmount := config.SpiConf.ShardAmount
	shardMaps := make([]map[string]*idChain.Node, shardAmount)
	Spiral.GlobalShards = make([]*Spiral.Shard, config.SpiConf.ShardAmount)
	// init shards
	for i := 0; i < shardAmount; i++ {
		shardMaps[i] = make(map[string]*idChain.Node)
		Spiral.GlobalShards[i] = Spiral.NewSpiralShard(uint64(i))
	}

	// assign nodes to shards.
	randNum := idChain.IDC.GetRand()
	for strPubKey, node := range idChain.IDC.NodeMap {
		shardId, _ := crypt.HashToRange(randNum, node.NodeId, node.Port(), shardAmount)
		if node.Port() == config.ListenPort && config.Debugging {
			shardId = config.DebugNodeAtShard
		}
		node.ShardID = uint64(shardId)
		shardMaps[shardId][strPubKey] = node
	}
	currShardId := idChain.IDC.NodeMap[idChain.RunningNode.StrKey()].ShardID
	Spiral.LocalShard = Spiral.GlobalShards[currShardId]
	Spiral.LocalShard.Chain.EnableStorage(idChain.RunningNode.Port())

	// selecting main node in each shard
	for i := 0; i < shardAmount; i++ {
		if len(shardMaps[i]) == 0 {
			log.Println("WARNING::not enough nodes.")
			continue
		}
		shardI := Spiral.GlobalShards[i]
		shardI.NodeMap = shardMaps[i]
		shardI.SelectMainNode()
	}

	Interfaces.Con[config.SpiMod].SetDomain(Spiral.LocalShard)
	log.Printf("Nodes in shard(%v):", currShardId)
	for _, node := range Spiral.GlobalShards[currShardId].NodeMap {
		log.Printf("\t%v\n", node.IpAddr)
	}

	log.Printf("Main node in shard is %v, threshold = %v",
		Spiral.LocalShard.Main().IpAddr,
		Interfaces.Con[config.SpiMod].GetDomain().Threshold(),
	)

	Interfaces.Communications[Interfaces.Ping].Request() // connect to other nodes .
	AutoTx.Manager = AutoTx.NewTxManager(Spiral.LocalShard.Chain.TxPool)
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
		if node.Port() == config.ListenPort && config.Debugging {
			shardId = config.DebugNodeAtShard
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
		Interfaces.Con[config.PyrMod].EnablePropose()
		go Interfaces.Operations[message.InternalTx].Propose()
		if pyramid.LocalShard.IsBShard() {
			go NxtCrossTx()
		}
	}
}

func EpochCountDown(t time.Duration) {
	time.Sleep(t)
	if idChain.IsIdMainNode() {
		Interfaces.Operations[message.NewEpoch].Propose()
	}
}
