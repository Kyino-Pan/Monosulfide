package Spiral

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
	Chain       *SpiChain
	rwLock      sync.RWMutex
	Id          int
	mainNode    *idChain.Node
	tempBlock   *Block.SpiralBlock
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
	sh.tempBlock = block.(*Block.SpiralBlock)
}

func (sh *Shard) GetViewId() uint64 {
	//TODO implement me
	panic("implement me")
}

func (sh *Shard) SetViewId(u uint64) {
	//TODO implement me
	panic("implement me")
}

func (sh *Shard) Addr() string {
	//TODO implement me
	panic("implement me")
}

func (sh *Shard) Append(block Block.Block) {
	if b, ok := block.(*Block.SpiralBlock); ok {
		sh.Chain.Append(b)
	} else {
		log.Panic()
	}
}

func (sh *Shard) SelectMainNode() *idChain.Node {
	randNum := idChain.IDC.GetRand()
	newIdMainNodeID := crypt.PubKey2Str(idChain.SelectRandomKey(sh.NodeMap, randNum))
	sh.mainNode = sh.NodeMap[newIdMainNodeID]
	if config.Debugging == true && sh.Id == config.DebugNodeAtShard && config.DebugIsMainNode {
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

func NewSpiralShard(shardId uint64) *Shard {
	ret := &Shard{
		NodeMap:     nil,
		GlobalState: nil,
		Chain:       nil,
		rwLock:      sync.RWMutex{},
		Id:          int(shardId),
	}
	ret.Chain = NewSpiChain(ret)
	return ret
}

func (sh *Shard) BlockExist(hash crypt.Hash) bool {
	if LocalShard.Chain.Blocks[hash] != nil {
		return true
	}
	return false
}
