package idChain

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/blockchain"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/message"
	"crypto/rsa"
	"encoding/json"
	"log"
	"sort"
	"sync"
)

var IDC *IdChain = nil

type IdChain struct {
	NodeMap   map[string]*Node
	roundId   uint64
	viewId    uint64
	randNum   uint64
	mainNode  *Node
	Seq       uint64
	chainLock sync.Mutex
	Chain     *blockchain.Chain
	nonce     uint64
	lock      sync.Mutex

	tempChainInfo *message.Content
	tempBlock     Block.Block
	tmpTopHash    crypt.Hash
}

func (idc *IdChain) SetViewId(u uint64) {
	idc.viewId = u
}

func (idc *IdChain) Addr() string {
	return config.IdRunningAddr
}

func (idc *IdChain) Main() *Node {
	return IDC.mainNode
}

func (idc *IdChain) ProcessingBlock() Block.Block {
	return idc.tempBlock
}

func (idc *IdChain) SetProcessingBlock(block Block.Block) {
	idc.tempBlock = block
}

func (idc *IdChain) AppendNode(nodeAddr string, nodeID *rsa.PublicKey) {
	idc.chainLock.Lock()
	defer idc.chainLock.Unlock()
	nodePubKey := string(crypt.EncodePublicKey(nodeID))
	if idc.NodeMap[nodePubKey] == nil {
		idc.NodeMap[nodePubKey] = &Node{
			NodeId:     nodeID,
			IpAddr:     nodeAddr,
			Activating: false,
			Sleeping:   true,
		}
		log.Printf("New node(%v) appended.\n", idc.LegalNodeAmount())
	} else {
		log.Printf("Node exist(%v).\n", idc.LegalNodeAmount())
	}
}

// unchecked
func (idc *IdChain) Threshold() uint64 {
	maliciousAmount := (idc.RunningNodeAmount()+2)/3 - 1
	return idc.RunningNodeAmount() - maliciousAmount
}

func (idc *IdChain) GlobalThreshold() uint64 {
	maliciousAmount := (idc.LegalNodeAmount()+2)/3 - 1
	return idc.LegalNodeAmount() - maliciousAmount
}

func (idc *IdChain) RunningNodeAmount() uint64 {
	ret := uint64(0)
	for _, n := range idc.NodeMap {
		if n.IsRunning() {
			ret++
		}
	}
	return ret
}

func (idc *IdChain) LegalNodeAmount() uint64 {
	ret := uint64(0)
	for _, n := range idc.NodeMap {
		if n.IsLegal() {
			ret++
		}
	}
	return ret
}

// activate all nodes except the faulty nodes.
func (idc *IdChain) ActivateAll() {
	for _, node := range idc.NodeMap {
		if node.Activating == false && node.Sleeping == true {
			node.Activating = true
		}
	}
}

func (idc *IdChain) ActivateConfirm() {
	for _, node := range idc.NodeMap {
		if node.Activating == true && node.Sleeping == true {
			node.Sleeping = false
		}
	}
}

func (idc *IdChain) State() map[string]uint {
	ret := make(map[string]uint)
	for key, node := range idc.NodeMap {
		ret[key] = node.State()
	}
	return ret
}

func (idc *IdChain) Encode() []byte {
	shardInfo, err := json.Marshal(idc)
	if err != nil {
		log.Panic(err)
		return nil
	}
	return shardInfo
}

func (idc *IdChain) Nonce() uint64 {
	ret := idc.nonce
	idc.nxtNonce()
	return ret
}

func (idc *IdChain) nxtNonce() {
	idc.nonce++
}

func (idc *IdChain) Append(block Block.Block) {
	// view id and randNum will be updated during append for ID-blocks.
	idc.Chain.Append(block)
	for _, tx := range block.Body().Txs() {
		if tx.Type == Tx.RegisterTx {
			id := tx.Sender
			addr := tx.Recipient
			if idc.NodeMap[id] == nil {
				idc.AppendNode(addr, crypt.DecodePublicKey([]byte(id)))
			}
		}
	}
	idc.ActivateAll()
	idc.ActivateConfirm()
	if len(idc.NodeMap) == 1 {
		for _, node := range idc.NodeMap {
			IDC.mainNode = node
		}
	} else {
		if b, ok := block.(*Block.StdBlock); ok {
			randByte := b.B.Interface
			if randByte != nil {
				rand := crypt.BytesToUint(randByte)
				idc.randNum = rand
			}
			idc.SelectMainNode()
		} else {
			log.Panic("Should implement block's specific method")
		}
	}
	idc.viewId = 0
	idc.roundId++
}

func (idc *IdChain) SelectMainNode() *Node {
	idc.Lock()
	defer idc.Unlock()
	newIdMainNodeID := crypt.PubKey2Str(SelectRandomKey(idc.NodeMap, idc.randNum))
	idc.mainNode = idc.NodeMap[newIdMainNodeID]
	return idc.mainNode
}

func (idc *IdChain) UpdateRandNum(num uint64) {
	idc.randNum = num
}

func (idc *IdChain) Lock() {
	idc.lock.Lock()
}

func (idc *IdChain) Unlock() {
	idc.lock.Unlock()
}

func (idc *IdChain) GetRand() uint64 {
	return idc.randNum
}

func (idc *IdChain) GetViewId() uint64 {
	return idc.viewId
}

func (idc *IdChain) SetView(vid uint64) {
	idc.viewId = vid
}

// 演示从公钥映射中随机选择一个键
func SelectRandomKey(keysMap map[string]*Node, randomIndex uint64) *rsa.PublicKey {
	keys := GetSortedKeySlice(keysMap)
	// 对键进行排序
	sort.Sort(*keys)
	// 使用randomIndex来选择一个键
	index := randomIndex % uint64(len(*keys))
	return (*keys)[index]
}

func (idc *IdChain) ChainInfo() *message.Content {
	if idc.tmpTopHash == idc.Chain.TopBlockHash[0] {
		return idc.tempChainInfo
	}
	blocks := IDC.Chain.GetBlocks()
	chainInfo := message.NewByteContent(crypt.UintToBytes(uint64(len(blocks))))
	for i := len(blocks) - 1; i >= 0; i-- {
		tempBlock := blocks[i]
		byteBlock := tempBlock.Encode()
		chainInfo.AppendByteContent(&byteBlock)
	}
	idc.tempChainInfo = chainInfo
	idc.tmpTopHash = idc.Chain.TopBlockHash[0]
	return chainInfo
}

func Init(Port string) {
	PriKey = crypt.InitPrivateKey()
	RunningNode = &Node{
		NodeId:     &PriKey.PublicKey,
		ShardID:    0,
		IpAddr:     config.Localhost + ":" + Port,
		Activating: false,
		Sleeping:   false,
		Silence:    0,
	}
	RunningNode.NodeId = &PriKey.PublicKey
	IDC = &IdChain{
		NodeMap:       make(map[string]*Node),
		roundId:       0,
		viewId:        0,
		randNum:       0,
		mainNode:      nil,
		Seq:           0,
		chainLock:     sync.Mutex{},
		Chain:         blockchain.NewChain("IDC_"+Port, uint64(0)),
		nonce:         0,
		lock:          sync.Mutex{},
		tempChainInfo: nil,
		tempBlock:     nil,
		tmpTopHash:    crypt.Hash{},
	}

}

// 自定义rsa.PublicKey的排序，因为它本身不能直接排序
type PublicKeySlice []*rsa.PublicKey

func (pks PublicKeySlice) Len() int {
	return len(pks)
}

func (pks PublicKeySlice) Less(i, j int) bool {
	// 将公钥的N值（模数）转换为字符串进行比较
	return pks[i].N.String() < pks[j].N.String()
}

func (pks PublicKeySlice) Swap(i, j int) {
	pks[i], pks[j] = pks[j], pks[i]
}

// HashToRange hashes a uint64 and rsa.PublicKey to a number in the range (0, n]

func GetSortedKeySlice(keysMap map[string]*Node) *PublicKeySlice {
	if len(keysMap) == 0 {
		return nil
	}
	keys := make(PublicKeySlice, 0, len(keysMap))
	for _, node := range keysMap {
		if node.IsRunning() {
			keys = append(keys, node.NodeId)
		}
	}
	return &keys
}
