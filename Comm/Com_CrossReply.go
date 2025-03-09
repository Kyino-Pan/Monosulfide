package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"log"
)

type CrossReplyCom struct {
	con    Interfaces.Consensus
	reqCnt map[crypt.Hash]uint64
}

func (com *CrossReplyCom) Type() Interfaces.CommType {
	return Interfaces.CrossReply
}

func (com *CrossReplyCom) Request(vars ...*[]byte) bool {
	com.con.SendMsg(&message.Message{
		Type:       com.Type().RequestType(),
		Content:    *message.NewByteContent(vars[0]),
		RemoteInfo: config.PyramidRelateI,
	})
	return true
}

func (com *CrossReplyCom) HandleRequest(msg *message.Message) bool {
	if com.con.GetDomain().ProcessingBlock() == nil {
		log.Printf("already append.")
		return false
	}
	content := msg.GetContents()
	if len(content) != 1 {
		log.Panic()
	}
	hash := crypt.NewHash(content[0])
	remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	BShardId := remoteNode.ShardID
	threshold := pyramid.GlobalPyrShards[BShardId].Threshold()
	com.reqCnt[*hash]++
	if com.reqCnt[*hash] == threshold {
		Interfaces.Operations[message.AppendCBlock].Propose() // This propose will return AFTER this round of consensus is finished.
		// this propose is not-synced, the PROPOSE will not return until execute is finished
	}
	return true
}

func (com *CrossReplyCom) Response(...*[]byte) bool {
	return true
}

func (com *CrossReplyCom) HandleResponse(*message.Message) {}

func (com *CrossReplyCom) Reset() Interfaces.CommType {
	com.con = Interfaces.Con[config.PyrMod]
	com.reqCnt = make(map[crypt.Hash]uint64)
	return com.Type()
}
