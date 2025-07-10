package MonoxideComm

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Monosulfide"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"blockEmulator/storage"
	"log"
	"sync"
)

type SyncSBlockCom struct {
	con    Interfaces.Consensus
	reqCnt map[int]map[crypt.Hash]uint64
	blocks map[int]map[crypt.Hash]Block.Block
	finish map[int]map[crypt.Hash]bool
	lock   sync.Mutex
}

func (com *SyncSBlockCom) Type() Interfaces.CommType {
	return Interfaces.SyncXideBlock
}

func (com *SyncSBlockCom) Request(...*[]byte) bool {
	con := com.con
	localChain := Monosulfide.LocalShard.Chain
	sAmount := config.MonoxideConf.ShardAmount
	content := new(message.Content)
	block := localChain.Blocks[localChain.TopBlockHash[localChain.Id()]].(*Block.FideBlock)
	for sid := 0; sid < sAmount; sid++ {
		if sid == localChain.Id() {
			continue
		}
		// prepare lightened block for shard sid
		tempBlock := &Block.FideBlock{
			H: block.H,
			B: &Block.FideBody{
				Intra:     nil,
				SubBlocks: make([]*Block.SubBlock, config.FideConf.ShardAmount),
			},
		}
		for i, subB := range block.B.SubBlocks {
			tempBlock.B.SubBlocks[i] = &Block.SubBlock{
				CHead: subB.CHead,
			}
			if i == sid {
				tempBlock.B.SubBlocks[i].CBody = subB.CBody
			}
		}
		bb := tempBlock.Encode()
		if idChain.RunningNode == Monosulfide.LocalShard.Main() {
			content = message.NewByteContent(&bb)
		} else {
			byteHash := tempBlock.Hash().Bytes()
			content = message.NewByteContent(&byteHash).
				AppendByteContent(crypt.UintToBytes(block.Nonce()))
		}
		for _, node := range Monosulfide.GlobalShards[sid].NodeMap {
			addr := node.IpAddr
			con.SendMsg(&message.Message{
				Type:       Interfaces.SyncXideBlock.RequestType(),
				Content:    *content,
				RemoteInfo: addr,
			})
		}
	}
	return true
}

func (com *SyncSBlockCom) HandleRequest(msg *message.Message) bool {
	com.lock.Lock()
	defer com.lock.Unlock()
	remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	remoteShard := Monosulfide.GlobalShards[remoteNode.ShardID]
	RemoteSID := remoteShard.Id()
	if com.reqCnt[RemoteSID] == nil {
		com.blocks[RemoteSID] = make(map[crypt.Hash]Block.Block)
		com.reqCnt[RemoteSID] = make(map[crypt.Hash]uint64)
		com.finish[RemoteSID] = make(map[crypt.Hash]bool)
	}
	contents := msg.GetContents()
	var hash crypt.Hash
	var nonce uint64
	localChain := Monosulfide.LocalShard.Chain
	if remoteShard.Main() == remoteNode {
		//sender is the main node, the msg contains the block itself.
		byteBlock := contents[0]
		b := new(Block.FideBlock).Decode(byteBlock)
		block, _ := b.(*Block.FideBlock)
		nonce = block.Nonce()
		if localChain.Blocks[block.Hash()] != nil {
			log.Printf("Got an existing block from shard%v, time = %v", RemoteSID, block.Head().Time().String())
			hash = block.Hash()
			com.blocks[RemoteSID][hash] = block
		} else {
			if localChain.TopBlockHash[RemoteSID] != block.H.ParentHash &&
				localChain.TopBlockHash[RemoteSID] != block.Hash() &&
				com.blocks[RemoteSID][block.H.ParentHash] == nil {
				// 如果这个block没有指向本地副本的topHash，且不是重复收到的block fmt.Print()
				if localChain.Blocks[localChain.TopBlockHash[RemoteSID]] != nil {
					log.Printf("shard%v::HASH_ERROR, localTop:%v, errorTop:%v", RemoteSID,
						localChain.Blocks[localChain.TopBlockHash[RemoteSID]].Head().GetNonce(),
						block.Nonce())
				}
				return false
			} else {
				hash = block.Hash()
				com.blocks[RemoteSID][hash] = block
			}
		}
	} else {
		byteBlockHash := contents[0]
		hash = *crypt.NewHash(byteBlockHash)
		nonce = crypt.BytesToUint(contents[1])
	}
	storage.CommLogger.Writef("Domain%v's block%v ack from %v (%v/%v), received:%v", remoteShard.Id(), nonce, remoteNode.IpAddr, com.reqCnt[RemoteSID][hash], remoteShard.Threshold(), com.blocks[RemoteSID][hash] != nil)
	if com.finish[RemoteSID][hash] == true {
		//log.Printf("blockAck%v(old) from %v (%v/%v)", remoteShard.sid, remoteNode.IpAddr, com.reqCnt[RemoteSID][hash], remoteShard.Threshold())
		return true
	}
	com.reqCnt[RemoteSID][hash]++
	//log.Printf("Domain%v's block%v ack from %v (%v/%v), received:%v", remoteShard.sid, nonce, remoteNode.IpAddr, com.reqCnt[RemoteSID][hash], remoteShard.Threshold(), com.blocks[RemoteSID][hash] != nil)
	com.CheckBlock(RemoteSID)
	return true
}

func (com *SyncSBlockCom) CheckBlock(rid int) {
	for hash, ackAmount := range com.reqCnt[rid] {
		b := com.blocks[rid][hash]
		if b == nil || ackAmount < Monosulfide.GlobalShards[rid].Threshold() {
			continue
		}
		block, _ := b.(*Block.FideBlock)
		if com.finish[rid][hash] == false {
			localShard := Monosulfide.LocalShard // i-shards internal tx is recorded in local shard txPool
			localShard.Append(block)
			//log.Printf("S%v B%v accecpted", rid, b.GetNonce())
			delete(com.blocks[rid], hash)
			delete(com.reqCnt[rid], hash)
			com.finish[rid][hash] = true
			com.CheckBlock(rid)
			return
		}
	}
}

func (com *SyncSBlockCom) Response(...*[]byte) bool {
	return true
}

func (com *SyncSBlockCom) HandleResponse(*message.Message) {
	log.Panic()
}

func (com *SyncSBlockCom) Reset(con Interfaces.Consensus) Interfaces.CommType {
	com.con = con
	com.reqCnt = make(map[int]map[crypt.Hash]uint64)
	com.blocks = make(map[int]map[crypt.Hash]Block.Block)
	com.finish = make(map[int]map[crypt.Hash]bool)
	return com.Type()
}
