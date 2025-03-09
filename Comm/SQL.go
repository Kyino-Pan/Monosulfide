package Comm

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"bytes"
	"log"
)

var Sql = Interfaces.CommType("Sql")
var SqlComReq = message.MessageType("SqlComReq")
var SqlComResp = message.MessageType("SqlComResp")

type SqlCom struct {
}

func (s SqlCom) Request(vars ...*[]byte) bool {
	if len(vars) != 3 {
		log.Panic()
	}
	addr := string(*vars[0])
	sql := vars[1]
	launch.LaunchIdMsg(&message.Message{
		Type:       SqlComReq,
		Content:    *message.NewByteContent(sql),
		RemoteInfo: addr,
	})
	return true
}

func (s SqlCom) HandleRequest(msg *message.Message) bool {
	contents := msg.GetContents()
	remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	if contents == nil || len(contents) != 1 {
		log.Panic()
	}
	sql := contents[0]

	hash := sql
	bb, err := pyramid.LocalShard.GetBlock(hash)
	var content *message.Content
	if err != nil {
		content = message.NewByteContent(&config.SuccessByte).AppendByteContent(&bb)
	} else {
		content = message.NewByteContent(&config.FailByte).AppendByteContent()
	}
	launch.LaunchPyrMsg(&message.Message{
		Type:       SqlComResp,
		Content:    *content,
		RemoteInfo: remoteNode.IpAddr,
	})
	return true
}

func (s SqlCom) Response(...*[]byte) bool {
	log.Panic()
	return false
}

func (s SqlCom) HandleResponse(msg *message.Message) {
	contents := msg.GetContents()
	remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	if contents == nil || len(contents) != 2 {
		log.Panic()
	}
	result := contents[0]
	if bytes.Equal(result, config.SuccessByte) {
		block := new(Block.StdBlock).Decode(contents[1])
		pyramid.LocalShard.Chain.Append(block, int(remoteNode.ShardID)) //todo 优化掉，只用Append（）
	} else {
		log.Printf("ERROR::SQL FAILED")
	}
}
