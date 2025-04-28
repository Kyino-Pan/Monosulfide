package PyramidComm

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"bytes"
	"log"
	"sync"
)

type CrossPrepareCom struct {
	con      Interfaces.Consensus
	acks     map[int]bool
	bShardId int
	lock     sync.Mutex
}

func (com *CrossPrepareCom) Type() Interfaces.CommType {
	return Interfaces.CrossPrepare
}

func (com *CrossPrepareCom) Request(...*[]byte) bool {
	//vars[0] = &byteBlock
	com.lock.Lock()
	defer com.lock.Unlock()
	localShard := pyramid.LocalShard
	localId := localShard.Id()
	byteBlock := com.con.GetDomain().ProcessingBlock().Encode()
	//log.Printf("%v", len(byteBlock))
	contents := message.NewByteContent(crypt.UintToBytes(uint64(localId))).AppendByteContent(&byteBlock)
	// contents = [ b-shardId, byteBlock ]
	//log.Printf("CPR Len:%v", len(*contents))
	msg := &message.Message{
		Type:       com.Type().RequestType(),
		Content:    *contents,
		RemoteInfo: config.PyramidRelateI,
	}
	com.con.SendMsg(msg)
	return true
}

func (com *CrossPrepareCom) HandleRequest(msg *message.Message) bool {
	com.lock.Lock()
	defer com.lock.Unlock()
	contents := msg.GetContents()
	com.bShardId = int(crypt.BytesToUint(contents[0]))
	block := new(Block.StdBlock).Decode(contents[1])
	suc := pyramid.LocalShard.Chain.VerifyBlock(block)
	if suc {
		Interfaces.Operations[message.IShardPrepare].Schedule(&contents[1])
	} else {
		log.Println("CrossPrepare:: failed")
		com.Response(&config.FailByte)
	}
	return true
}

func (com *CrossPrepareCom) Response(vars ...*[]byte) bool {
	com.lock.Lock()
	defer com.lock.Unlock()
	localId := pyramid.LocalShard.Id()
	contents := message.NewByteContent(crypt.UintToBytes(uint64(localId))).
		AppendByteContent(vars[0])
	com.con.SendMsg(&message.Message{
		Type:       com.Type().ResponseType(),
		Content:    *contents,
		RemoteInfo: pyramid.ShardAddr(com.bShardId),
	})
	return true
}

func (com *CrossPrepareCom) HandleResponse(msg *message.Message) {
	com.lock.Lock()
	defer com.lock.Unlock()
	contents := msg.GetContents()
	remoteShardId := crypt.BytesToUint(contents[0])
	result := contents[1]
	com.acks[int(remoteShardId)] = bytes.Equal(result, config.SuccessByte)
	if !bytes.Equal(result, config.SuccessByte) {
		log.Println("CrossPrepare::error")
	}
	log.Printf("From shard%v, Cross Prepare(%v/%v)", remoteShardId, len(com.acks), len(pyramid.LocalShard.RelatedIShard))
	if len(com.acks) == len(pyramid.LocalShard.RelatedIShard) {
		com.acks = make(map[int]bool)
		Interfaces.Operations[message.CrossCommit].Schedule()
	}
	return
}

func (com *CrossPrepareCom) Reset(con Interfaces.Consensus) Interfaces.CommType {
	com.con = con
	com.acks = make(map[int]bool)
	return com.Type()
}
