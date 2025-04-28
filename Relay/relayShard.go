package Relay

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"log"
	"sync"
)

var LocalShard *Shard
var GlobalShards []*Shard

type Shard struct {
	NodeMap     map[string]*idChain.Node
	GlobalState map[string]Tx.Account
	Chain       *Chain
	rwLock      sync.RWMutex
	sid         int
	mainNode    *idChain.Node
	tempBlock   *Block.StdBlock
}

func (sh *Shard) SetMap(mp map[string]*idChain.Node) {
	sh.rwLock.Lock()
	defer sh.rwLock.Unlock()
	sh.NodeMap = mp
}

func (sh *Shard) GetTxPool() *Tx.Pool {
	return sh.Chain.TxPool
}

func (sh *Shard) SetMain(node *idChain.Node) {
	sh.mainNode = node
}

func (sh *Shard) Id() int {
	return sh.sid
}

func (sh *Shard) GetMap() map[string]*idChain.Node {
	//TODO implement me
	panic("implement me")
}

func (sh *Shard) Main() *idChain.Node {
	return sh.mainNode
}

func (sh *Shard) Threshold() uint64 {
	maliciousAmount := (sh.NodeAmount()+2)/3 - 1
	return uint64(sh.NodeAmount() - maliciousAmount)
}

func (sh *Shard) ProcessingBlock() Block.Block {
	return sh.tempBlock
}

func (sh *Shard) SetProcessingBlock(block Block.Block) {
	sh.tempBlock = block.(*Block.StdBlock)
}

func (sh *Shard) GetViewId() uint64 {
	//TODO implement me
	panic("implement me")
}

func (sh *Shard) SetViewId(u uint64) {
	//TODO implement me
	panic("implement me")
}

func (sh *Shard) BroadAddr() string {
	//TODO implement me
	panic("implement me")
}

func (sh *Shard) Append(block Block.Block) {
	if b, ok := block.(*Block.RelayBlock); ok {
		sh.Chain.Append(b)
	} else {
		log.Panic()
	}
}

func (sh *Shard) SelectMain() *idChain.Node {
	randNum := idChain.IDC.GetRand()
	newIdMainNodeID := crypt.PubKey2Str(idChain.SelectRandomKey(sh.NodeMap, randNum))
	sh.mainNode = sh.NodeMap[newIdMainNodeID]
	if config.EnableSpy == true && sh.sid == config.SpyAtShard && config.SpyIsMainNode {
		for _, node := range sh.NodeMap {
			if node.Port() == config.ListenPort {
				sh.mainNode = node
				break
			}
		}
	}
	return sh.mainNode
}

func (sh *Shard) NodeAmount() int {
	return len(sh.NodeMap)
}

func (sh *Shard) BlockExist(hash crypt.Hash) bool {
	if LocalShard.Chain.Blocks[hash] != nil {
		return true
	}
	return false
}

func (sh *Shard) Reset(port int, i int) {
	sh.NodeMap = nil
	sh.GlobalState = nil
	sh.Chain = nil
	sh.rwLock = sync.RWMutex{}
	sh.sid = i

	sh.Chain = NewRelayChain(uint64(port), sh)
	return
}
