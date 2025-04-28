package PyramidComm

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"log"
	"sync"
)

type SyncIBlockCom struct {
	con    Interfaces.Consensus
	reqCnt map[int]map[crypt.Hash]uint64
	blocks map[int]map[crypt.Hash]Block.Block
	finish map[int]map[crypt.Hash]bool
	lock   sync.Mutex
}

func (com *SyncIBlockCom) Type() Interfaces.CommType {
	return Interfaces.SyncIBlock
}

func (com *SyncIBlockCom) Request(...*[]byte) bool {
	con := com.con
	bIndexes := pyramid.LocalShard.RelatedBShard
	content := new(message.Content)
	block := pyramid.LocalShard.ProcessingBlock()
	if pyramid.IsPyrMainNode() {
		byteBlock := block.Encode()
		content = message.NewByteContent(&byteBlock) //main node will broadcast the block
	} else {
		byteHash := block.Hash().Bytes()
		content = message.NewByteContent(&byteHash).AppendByteContent(
			crypt.UintToBytes(uint64(len(block.Body().Txs())))) //replica node will broadcast the blockHash
	}
	for _, idx := range bIndexes {
		for _, node := range pyramid.GlobalShards[idx].NodeMap {
			addr := node.IpAddr
			con.SendMsg(&message.Message{
				Type:       Interfaces.SyncIBlock.RequestType(),
				Content:    *content,
				RemoteInfo: addr,
			})
		}
	}
	return true
}

func (com *SyncIBlockCom) HandleRequest(msg *message.Message) bool {
	com.lock.Lock()
	defer com.lock.Unlock()
	remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	remoteShard := pyramid.GlobalShards[remoteNode.ShardID]
	RemoteSID := remoteShard.Id()
	if com.reqCnt[RemoteSID] == nil {
		com.blocks[RemoteSID] = make(map[crypt.Hash]Block.Block)
		com.reqCnt[RemoteSID] = make(map[crypt.Hash]uint64)
		com.finish[RemoteSID] = make(map[crypt.Hash]bool)
	}
	contents := msg.GetContents()
	var hash crypt.Hash
	localChain := pyramid.LocalShard.Chain
	//blockLen := 0
	if remoteShard.Main() == remoteNode {
		//sender is the main node, the msg contains the block itself.
		byteBlock := contents[0]
		b := new(Block.StdBlock).Decode(byteBlock)
		block, _ := b.(*Block.StdBlock)
		//blockLen = len(block.StdBody().Transactions)
		if localChain.Blocks[block.Hash()] != nil {
			log.Printf("Got an existing block from shard%v, hash = %v", RemoteSID, len(localChain.Blocks[block.Hash()].Body().Txs()))
			hash = block.Hash()
			com.blocks[RemoteSID][hash] = block
		} else {
			if (localChain.TopBlockHash[RemoteSID] != block.H.ParentHashes[RemoteSID]) &&
				!(localChain.TopHashToBeConfirmed[RemoteSID] == true && localChain.TopBlockHash[RemoteSID] == block.Hash()) {
				// 如果这个block没有指向本地副本的topHash，且不是重复收到的block
				log.Printf("shard%v::HASH_ERROR, localTop:%v, errorTop:%v", RemoteSID,
					len(localChain.Blocks[localChain.TopBlockHash[RemoteSID]].Body().Txs()),
					len(block.Body().Txs()))
				return false
			} else {
				hash = block.Hash()
				com.blocks[RemoteSID][hash] = block
			}
		}
	} else {
		byteBlockHash := contents[0]
		//blockLen = int(crypt.BytesToUint(contents[1]))
		hash = *crypt.NewHash(byteBlockHash)
	}
	//storage.CommLogger.Writef("block<%v>Ack%v from %v (%v/%v), block received:%v", blockLen, remoteShard.sid, remoteNode.IpAddr, com.reqCnt[RemoteSID][hash], remoteShard.Threshold(), com.blocks[RemoteSID][hash] != nil)
	if com.finish[RemoteSID][hash] == true {
		//log.Printf("blockAck%v(old) from %v (%v/%v)", remoteShard.sid, remoteNode.IpAddr, com.reqCnt[RemoteSID][hash], remoteShard.Threshold())
		return true
	}
	com.reqCnt[RemoteSID][hash]++
	//log.Printf("block<%v>Ack%v from %v (%v/%v), block received:%v", blockLen, remoteShard.sid, remoteNode.IpAddr, com.reqCnt[RemoteSID][hash], remoteShard.Threshold(), com.blocks[RemoteSID][hash] != nil)

	com.CheckBlock(RemoteSID)
	return true
}

func (com *SyncIBlockCom) CheckBlock(rid int) {
	for hash, cnt := range com.reqCnt[rid] {
		b := com.blocks[rid][hash]
		block, _ := b.(*Block.StdBlock)
		if block == nil || cnt < pyramid.GlobalShards[rid].Threshold() {
			continue
		}
		if pyramid.GetBlock(block.H.ParentHashes[rid]) != nil {
			localShard := pyramid.LocalShard // i-shards internal tx is recorded in local shard txPool
			localShard.Chain.Append(block, rid)
			delete(com.blocks[rid], hash)
			delete(com.reqCnt[rid], hash)
			com.finish[rid][hash] = true
			com.CheckBlock(rid)
			return
		}
	}
}

func (com *SyncIBlockCom) Response(...*[]byte) bool {
	return true
}

func (com *SyncIBlockCom) HandleResponse(*message.Message) {
}

func (com *SyncIBlockCom) Reset(con Interfaces.Consensus) Interfaces.CommType {
	com.con = con
	com.reqCnt = make(map[int]map[crypt.Hash]uint64)
	com.blocks = make(map[int]map[crypt.Hash]Block.Block)
	com.finish = make(map[int]map[crypt.Hash]bool)
	return com.Type()
}
