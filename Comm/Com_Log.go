package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"log"
	"sync"
)

// not using now
type LogCom struct {
	con           Interfaces.Consensus
	finish        map[string]bool
	handleBlocker sync.Mutex
	cond          *sync.Cond
	threshold     int
}

func (com *LogCom) Request(...*[]byte) bool {
	com.threshold = len(idChain.IDC.NodeMap) // has to receive response from ALL NODES in shard.
	log.Printf("waiting for responses.")
	com.cond.Wait()
	return true
}

func (com *LogCom) HandleRequest(*message.Message) bool {
	return true
}

func (com *LogCom) Response(...*[]byte) bool {
	com.con.SendMsg(&message.Message{
		Type:       Interfaces.Log.ResponseType(),
		Content:    *message.NewStrContent(""),
		RemoteInfo: idChain.IDC.Main().IpAddr,
	})
	return true
}

func (com *LogCom) HandleResponse(msg *message.Message) {
	com.handleBlocker.Lock()
	remoteId := msg.RemoteInfo
	com.finish[remoteId] = true
	log.Printf("from %v (%v/%v)", idChain.IDC.NodeMap[remoteId].IpAddr, len(com.finish), com.threshold)
	if len(com.finish) == com.threshold {
		com.finish = make(map[string]bool) //refresh
		log.Printf("enough")
		com.cond.Signal()
	}
	com.handleBlocker.Unlock()
	return
}

func (com *LogCom) Reset() Interfaces.CommType {
	com = &LogCom{
		finish: make(map[string]bool),
	}
	com.cond = sync.NewCond(&com.handleBlocker)
	com.handleBlocker.Lock()
	return Interfaces.Log
}
