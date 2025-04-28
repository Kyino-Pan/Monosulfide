package Interfaces

import (
	"blockEmulator/message"
	"sync"
)

type Communication interface {
	Reset(con Consensus) CommType
	Request(vars ...*[]byte) bool
	HandleRequest(msg *message.Message) bool
	Response(vars ...*[]byte) bool
	HandleResponse(msg *message.Message)
	Type() CommType
}

type CommType string

const (
	CrossPrepare  CommType = "CrossPrepare"
	CrossReply    CommType = "CrossReply"
	CrossLock     CommType = "CrossLock"
	MainBegin     CommType = "MainBegin"
	MigratePro    CommType = "MigratePro" // migrate proposal buffer
	PoWBroadcast  CommType = "PoWBroadcast"
	Ping          CommType = "ping"
	Register      CommType = "Register"
	SyncIBlock    CommType = "SyncIBlock"
	SyncFideBlock CommType = "SyncFideBlock"
	ViewChange    CommType = "ViewChange"
)

func (ct CommType) RequestType() message.MessageType {
	return message.MessageType(string(ct) + "REQ")
}

func (ct CommType) ResponseType() message.MessageType {
	return message.MessageType(string(ct) + "RES")
}

var (
	ComTypes      = make(map[message.MessageType]CommType)
	ComBuffer     = make(map[CommType]map[bool][][]*[]byte) // [type][isRequest][i]=vars
	comBufferLock sync.Mutex
)

func IsReq(mt message.MessageType) bool {
	return string(mt[len(mt)-1]) == "Q"
}

func AppendDelayedCom(comType CommType, isRequest bool, vars ...*[]byte) {
	if ComBuffer[comType] == nil {
		ComBuffer[comType] = make(map[bool][][]*[]byte)
	}
	ComBuffer[comType][isRequest] = append(ComBuffer[comType][isRequest], vars)
}

func ClearComBuffer() {
	comBufferLock.Lock()
	defer comBufferLock.Unlock()
	for comType, mp := range ComBuffer {
		for isRequest, comI := range mp {
			for _, vars := range comI {
				if isRequest {
					Communications[comType].Request(vars...)
				} else {
					Communications[comType].Response(vars...)
				}
			}
		}
	}
	ComBuffer = make(map[CommType]map[bool][][]*[]byte)
}

var Communications = make(map[CommType]Communication)
