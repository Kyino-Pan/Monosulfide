package Monosulfide

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"log"
	"strconv"
	"sync"
)

var LocalShard *Shard
var GlobalShards []*Shard

type Shard struct {
	NodeMap     map[string]*idChain.Node
	GlobalState map[string]Tx.Account
	Chain       *MonosulfideChain
	rwLock      sync.RWMutex
	sid         int
	mainNode    *idChain.Node
	tempBlock   *Block.FideBlock
}

func (sh *Shard) BroadAddr() string {
	return "MONOSULFIDE_" + strconv.Itoa(sh.sid)
}

func (sh *Shard) Reset(_ int, sid int) {
	sh.NodeMap = nil
	sh.GlobalState = nil
	sh.Chain = nil
	sh.rwLock = sync.RWMutex{}
	sh.sid = sid
	sh.Chain = NewFideChain(sh)
}

func (sh *Shard) SetMap(mp map[string]*idChain.Node) {
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
	return sh.NodeMap
}

func (sh *Shard) Main() *idChain.Node {
	return sh.mainNode
}

func (sh *Shard) Threshold() uint64 {
	maliciousAmount := (sh._nodeAmount()+2)/3 - 1
	return uint64(sh._nodeAmount() - maliciousAmount)
}

func (sh *Shard) ProcessingBlock() Block.Block {
	return sh.tempBlock
}

func (sh *Shard) SetProcessingBlock(block Block.Block) {
	sh.tempBlock = block.(*Block.FideBlock)
}

func (sh *Shard) Append(block Block.Block) {
	if b, ok := block.(*Block.FideBlock); ok {
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

func (sh *Shard) _nodeAmount() int {
	return len(sh.NodeMap)
}

func (sh *Shard) BlockExist(hash crypt.Hash) bool {
	if LocalShard.Chain.Blocks[hash] != nil {
		return true
	}
	return false
}
