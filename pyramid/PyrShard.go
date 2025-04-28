package pyramid

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"sync"
	"time"
)

var GlobalShards []*Shard = nil
var LocalShard *Shard

type Shard struct {
	NodeMap map[string]*idChain.Node // participants of this shard

	Chain         *PyrChain
	mainNode      *idChain.Node
	shardLock     sync.Mutex
	sid           int
	RelatedIShard []int
	RelatedBShard []int
	tempBlock     Block.Block
}

func (sh *Shard) Reset(port int, sid int) {
	sh.NodeMap = nil
	sh.Chain = nil
	sh.mainNode = nil
	sh.sid = sid
	sh.RelatedIShard = make([]int, 0)
	sh.RelatedBShard = make([]int, 0)
	for i := 0; i < config.PyrConf.ShardAmount; i++ {
		if config.PyrConf.ShardDistribution[i][sid] == true && i != sid {
			sh.RelatedBShard = append(sh.RelatedBShard, i)
		}
		if config.PyrConf.ShardDistribution[sid][i] == true && i != sid {
			sh.RelatedIShard = append(sh.RelatedIShard, i)
		}
	}
	sh.Chain = NewPyrChain(uint64(port), sh)
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

func (sh *Shard) GetBlock(bHash []byte) ([]byte, error) {
	return sh.Chain.storage.GetBlock(bHash)
}

// implement Domain Interface

func (sh *Shard) GetViewId() uint64 {
	//TODO implement me

	panic("implement me")
}

func (sh *Shard) SetViewId(u uint64) {
	//TODO implement me

	panic("implement me")
}

func (sh *Shard) BroadAddr() string {
	return config.PyrRunningAddr
}

func (sh *Shard) ProcessingBlock() Block.Block {
	return sh.tempBlock
}
func (sh *Shard) SetProcessingBlock(block Block.Block) {
	sh.tempBlock = block
}

func (sh *Shard) Main() *idChain.Node {
	return sh.mainNode
}

func (sh *Shard) SelectMain() *idChain.Node {
	randNum := idChain.IDC.GetRand()
	newIdMainNodeID := crypt.PubKey2Str(idChain.SelectRandomKey(sh.NodeMap, randNum))
	sh.mainNode = sh.NodeMap[newIdMainNodeID]
	if config.EnableSpy == true && sh.Id() == config.SpyAtShard && config.SpyIsMainNode {
		for _, node := range sh.NodeMap {
			if node.Port() == config.ListenPort {
				sh.mainNode = node
				break
			}
		}
	}
	return sh.mainNode
}

func NewPyramidShard(port, shardId uint64) *Shard {
	ret := &Shard{
		NodeMap:       nil,
		Chain:         nil,
		mainNode:      nil,
		sid:           int(shardId),
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
	return GlobalShards[idChain.RunningNode.ShardID].mainNode == idChain.RunningNode
}

func (sh *Shard) state() map[string]uint {
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

func (sh *Shard) NodeAmount() int {
	return len(sh.NodeMap)
}

func (sh *Shard) Threshold() uint64 {
	maliciousAmount := (sh.NodeAmount()+2)/3 - 1
	return uint64(sh.NodeAmount() - maliciousAmount)
}

func (sh *Shard) WaitForInternalTxs() {
	beginT := time.Now()
	for time.Since(beginT).Seconds() < 3.0 {
		if sh.internalTxsAmount() < 5 {
			time.Sleep(188 * time.Millisecond)
		} else {
			break
		}
	}
}

func (sh *Shard) internalTxsAmount() int {
	return sh.Chain.TxPool.TxLists[sh.Id()][sh.Id()].Len()
}

func (sh *Shard) UnconfirmedCrossTxsLen() int {
	cnt := 0
	for i := 0; i < config.PyrConf.ShardAmount; i++ {
		if i == sh.Id() {
			continue
		}
		cnt += sh.Chain.TxPool.TxLists[i][sh.Id()].Len()
		cnt += sh.Chain.TxPool.TxLists[sh.Id()][i].Len()
	}
	cnt -= sh.UnconfirmedInternal()
	return cnt
}

func (sh *Shard) RecordTx(tx *Tx.Transaction) {
	sh.Chain.RecordTx(tx)
}

func (sh *Shard) Append(block Block.Block) {
	// Append will save the block to disk AND REMOVE TXS IN MEM.
	sh.Chain.Append(block, sh.Id())
	// So after append the block.txs will be nil.
}

func (sh *Shard) UnconfirmedInternal() int {
	return sh.Chain.TxPool.TxLists[sh.Id()][sh.Id()].Len()
}

func (sh *Shard) Lock() {
	sh.shardLock.Lock()
}

func (sh *Shard) Unlock() {
	sh.shardLock.Unlock()
}

func (sh *Shard) IsBShard() bool {
	return len(sh.RelatedIShard) > 0
}

// Controls returns whether shard storage the chain of shard-r
func (sh *Shard) Controls(r int) bool {
	return config.PyrConf.ShardDistribution[sh.Id()][r]
}

func (sh *Shard) AppendIShBlock(block Block.Block, shardId int) {
	//sh.PrintTxs()
	sh.Chain.AppendIShardIBlock(block, shardId)
	//sh.PrintTxs()
}

func (sh *Shard) GenStateRoot() []byte {
	//todo
	return []byte("STATE_ROOT")
}

func GetBlock(hash crypt.Hash) Block.Block {
	return LocalShard.Chain.Blocks[hash] //todo
}

func ShardAddr(id int) string {
	// returning the addr of the main node in shard[id]
	return GlobalShards[id].Main().IpAddr
}

func NxtCrossTx() {
	time.Sleep(config.SleepMin * 32 * time.Millisecond)
	Interfaces.Communications[Interfaces.CrossLock].Request()
}
