package pyramid

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"sync"
	"time"
)

var GlobalPyrShards []*PyrShard = nil
var LocalShard *PyrShard

type PyrShard struct {
	NodeMap       map[string]*idChain.Node // participants of this shard
	Chain         *PyrChain
	mainNode      *idChain.Node
	shardLock     sync.Mutex
	Id            int
	RelatedIShard []int
	RelatedBShard []int
	tempBlock     Block.Block
}

func (sh *PyrShard) GetBlock(bHash []byte) ([]byte, error) {
	return sh.Chain.storage.GetBlock(bHash)
}

// implement Domain Interface

func (sh *PyrShard) GetViewId() uint64 {
	//TODO implement me

	panic("implement me")
}

func (sh *PyrShard) SetViewId(u uint64) {
	//TODO implement me

	panic("implement me")
}

func (sh *PyrShard) Addr() string {
	return config.PyrRunningAddr
}

func (sh *PyrShard) ProcessingBlock() Block.Block {
	return sh.tempBlock
}
func (sh *PyrShard) SetProcessingBlock(block Block.Block) {
	sh.tempBlock = block
}

func (sh *PyrShard) Main() *idChain.Node {
	return sh.mainNode
}

func (sh *PyrShard) SelectMainNode() *idChain.Node {
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

func NewPyramidShard(port, shardId uint64) *PyrShard {
	ret := &PyrShard{
		NodeMap:       nil,
		Chain:         nil,
		mainNode:      nil,
		Id:            int(shardId),
		RelatedIShard: make([]int, 0),
		RelatedBShard: make([]int, 0),
	}
	for i := 0; i < config.PyrConf.ShardAmount; i++ {
		if config.PyrConf.ShardDistribution[i][shardId] == true && uint64(i) != shardId {
			ret.RelatedBShard = append(ret.RelatedBShard, i)
		}
		if config.PyrConf.ShardDistribution[shardId][i] == true && uint64(i) != shardId {
			ret.RelatedIShard = append(ret.RelatedIShard, i)
		}
	}
	ret.Chain = NewPyrChain(port, ret)
	return ret
}

func IsPyrMainNode() bool {
	return GlobalPyrShards[idChain.RunningNode.ShardID].mainNode == idChain.RunningNode
}

func (sh *PyrShard) state() map[string]uint {
	ret := make(map[string]uint)
	keys := idChain.GetSortedKeySlice(sh.NodeMap)
	if keys == nil {
		return nil
	}
	for i := 0; i < keys.Len(); i++ {
		k := string(crypt.EncodePublicKey((*keys)[i]))
		ret[k] = sh.NodeMap[k].State()
	}
	return ret
}

func (sh *PyrShard) NodeAmount() int {
	return len(sh.NodeMap)
}

func (sh *PyrShard) Threshold() uint64 {
	maliciousAmount := (sh.NodeAmount()+2)/3 - 1
	return uint64(sh.NodeAmount() - maliciousAmount)
}

func (sh *PyrShard) WaitForInternalTxs() {
	beginT := time.Now()
	for time.Since(beginT).Seconds() < 3.0 {
		if sh.internalTxsAmount() < 5 {
			time.Sleep(188 * time.Millisecond)
		} else {
			break
		}
	}
}

func (sh *PyrShard) internalTxsAmount() int {
	return sh.Chain.TxPool.TxLists[sh.Id][sh.Id].Len()
}

func (sh *PyrShard) UnconfirmedCrossTxsLen() int {
	cnt := 0
	for i := 0; i < config.PyrConf.ShardAmount; i++ {
		if i == sh.Id {
			continue
		}
		cnt += sh.Chain.TxPool.TxLists[i][sh.Id].Len()
		cnt += sh.Chain.TxPool.TxLists[sh.Id][i].Len()
	}
	cnt -= sh.UnconfirmedInternal()
	return cnt
}

func (sh *PyrShard) RecordTx(tx *Tx.Transaction) {
	sh.Chain.RecordTx(tx)
}

func (sh *PyrShard) Append(block Block.Block) {
	// Append will save the block to disk AND REMOVE TXS IN MEM.
	sh.Chain.Append(block, sh.Id)
	// So after append the block.txs will be nil.
}

func (sh *PyrShard) UnconfirmedInternal() int {
	return sh.Chain.TxPool.TxLists[sh.Id][sh.Id].Len()
}

func (sh *PyrShard) Lock() {
	sh.shardLock.Lock()
}

func (sh *PyrShard) Unlock() {
	sh.shardLock.Unlock()
}

func (sh *PyrShard) IsBShard() bool {
	return len(sh.RelatedIShard) > 0
}

// Controls returns whether shard storage the chain of shard-r
func (sh *PyrShard) Controls(r int) bool {
	return config.PyrConf.ShardDistribution[sh.Id][r]
}

func (sh *PyrShard) AppendIShBlock(block Block.Block, shardId int) {
	//sh.PrintTxs()
	sh.Chain.AppendIShardIBlock(block, shardId)
	//sh.PrintTxs()
}

func (sh *PyrShard) GenStateRoot() []byte {
	//todo
	return []byte("STATE_ROOT")
}

func GetBlock(hash crypt.Hash) Block.Block {
	return LocalShard.Chain.Blocks[hash] //todo
}

func ShardAddr(id int) string {
	// returning the addr of the main node in shard[id]
	return GlobalPyrShards[id].Main().IpAddr
}
