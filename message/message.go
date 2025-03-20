package message

import (
	"blockEmulator/config"
	"fmt"
	"log"
	"time"
)

type MessageType string
type RequestType string

const (
	CrossPrePre   RequestType = "CrossPrePre"
	IShardPrepare RequestType = "IShardPrepare"
	SendTxs                   = MessageType(PyrPrefix + "SendTxs")
	AppendCBlock  RequestType = "AppendCBlock"
	CrossCommit   RequestType = "CrossCommit"
	RemoveNode    RequestType = "RemoveNode" // this var is disabled now.
	Empty         RequestType = "Empty"
)

type Message struct {
	Type       MessageType
	Content    []byte
	RemoteInfo string
}

func (msg *Message) GetContents() [][]byte {
	contents := (*Content)(&msg.Content).ParseContent()
	return contents
}

func (t RequestType) String() string {
	return string(t)
}

type ActionType string

const (
	CPrePrepare MessageType = "BC_prePrepare"
	CPrepare    MessageType = "BC_prepare"
	CCommit     MessageType = "BC_commit"
	CReply      MessageType = "BC_reply"

	//CRelay  MessageType = "relay"
	PyrPrefix string = "PY"
	BCPrefix  string = "BC"

	PyrDefault = MessageType(PyrPrefix + "Default")
	//BCDefault  = MessageType(BCPrefix + "Default")
	SyncIdChain MessageType = "SyncShard"
	NodeSilence MessageType = "NodeSilence"
	TxEOF                   = MessageType("TxEOF")
)

type RawMessage struct {
	Content []byte // the content of raw message, txs and blocks (most cases) included
}

type Request struct {
	RequestType RequestType
	Msg         RawMessage // request message
	ReqTime     time.Time  // request time
}

type PrePrepare struct {
	RequestMsg *Request // the request message should be pre-prepared
	Digest     []byte   // the digest of this request, which is the only identifier
	SeqID      uint64
}

type Prepare struct {
	Digest []byte // To identify which request is prepared by this node
	SeqID  uint64
}

type Commit struct {
	Digest []byte // To identify which request is prepared by this node
	SeqID  uint64
}

type Reply struct {
	MessageID uint64
	Result    bool
}

//func MergeMessage(msgType MessageType, content []byte) []byte {
//	b := make([]byte, PrefixMsgTypeLen)
//	for i, v := range []byte(msgType) {
//		b[i] = v
//	}
//	merge := append(b, content...)
//	return merge
//}

func SplitMessage(message []byte) (MessageType, []byte) {
	if len(message) < config.PrefixMsgTypeLen {
		fmt.Println(string(message))
	}
	if len(message) < 30 {
		log.Printf("error msg len:%v", len(message))
	}
	msgTypeBytes := message[:config.PrefixMsgTypeLen]
	msgType_pruned := make([]byte, 0)
	for _, v := range msgTypeBytes {
		if v != byte(0) {
			msgType_pruned = append(msgType_pruned, v)
		}
	}
	msgType := string(msgType_pruned)
	content := message[config.PrefixMsgTypeLen:]
	return MessageType(msgType), content
}

var NewEpoch RequestType

var InternalTx RequestType = "InternalTx"
var FideTx RequestType = "FideTx"
var RelayTx RequestType = "RelayTx"
