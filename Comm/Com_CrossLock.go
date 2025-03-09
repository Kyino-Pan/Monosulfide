package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"blockEmulator/storage"
	"bytes"
	"log"
	"math/rand/v2"
	"sync"
	"time"
)

type CrossLockCom struct {
	con        Interfaces.Consensus
	comLock    sync.Mutex
	comCond    *sync.Cond
	acks       map[int]bool
	reqLock    sync.Mutex
	mapLock    sync.Mutex
	BMainNode  *idChain.Node
	activating bool
	success    bool
	sleepT     int
}

func (com *CrossLockCom) Type() Interfaces.CommType {
	return Interfaces.CrossLock
}

// this file is a piece of shit
// do NOT try to understand unless there's a bug located here.

// running sequence is like:
// b-request -> i-handleReq -> i-request -> b-handleRequest ->
// b-response -> i-handleResponse -> i-response -> b-handleResponse

func (com *CrossLockCom) Reset() Interfaces.CommType {
	com.activating = false
	com.acks = make(map[int]bool)
	com.comCond = sync.NewCond(&com.comLock)
	com.sleepT = config.SleepMin
	com.con = Interfaces.Con[config.PyrMod]
	return com.Type()
}

func (com *CrossLockCom) debug(info string) {
	return
	if com.BMainNode == nil {
		log.Printf("\t%v::main node is nil", info)
	} else {
		log.Printf("\t%v::main node is %v", info, com.BMainNode.IpAddr)
	}
}

func (com *CrossLockCom) wait() {
	com.comLock.Lock()
	com.comCond.Wait()
	com.comLock.Unlock()
	return
}

func (com *CrossLockCom) awake() {
	com.comLock.Lock()
	com.comCond.Signal()
	com.comLock.Unlock()
}

func (com *CrossLockCom) Request(vars ...*[]byte) bool {
	if pyramid.LocalShard.IsBShard() {
		log.Printf("Lock request")
		com.acks = make(map[int]bool)
		com.activating = true
		com.success = true
		time.Sleep(time.Millisecond * time.Duration(rand.IntN(com.sleepT))) // 抗碰撞
		com.con.SendMsg(&message.Message{
			Type:       com.Type().RequestType(),
			Content:    *message.NewByteContent(),
			RemoteInfo: config.PyramidRelateI,
		})
		com.wait() // will be blocked until every related i-shard is locked
		if com.success {
			com.BlockInner()
			com.Response(&config.SuccessByte)
			com.wait()
			Interfaces.Operations[message.CrossPrePre].Propose()
			storage.StateLogger.Writef("Lock success")
		} else {
			com.Response(&config.FailByte)
			com.sleepT = min(com.sleepT*2, config.SleepMin*128)
			go com.Request(vars...)
		}
		com.activating = false
		return com.success
	} else { // i-shards ->answering b-shards
		com.con.SendMsg(&message.Message{
			Type:       com.Type().RequestType(),
			Content:    *message.NewByteContent(vars[0]), //vars here is success or not
			RemoteInfo: com.BMainNode.IpAddr,
		})
		com.debug("req")
		return true
	}
}
func (com *CrossLockCom) HandleRequest(msg *message.Message) bool {
	com.reqLock.Lock()
	defer com.reqLock.Unlock()
	log.Printf("Lock handle req, %v", idChain.IDC.NodeMap[msg.RemoteInfo].IpAddr)
	if pyramid.LocalShard.IsBShard() {
		if com.activating == false {
			log.Printf("WARNING::CrossLock get illegal response ")
			return false
		}
		contents := msg.GetContents()
		if bytes.Equal(contents[0], config.FailByte) {
			// failed to lock
			log.Printf("Refused by %v", idChain.IDC.NodeMap[msg.RemoteInfo].IpAddr)
			com.success = false
		}
		remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
		remoteShardId := remoteNode.ShardID
		com.acks[int(remoteShardId)] = com.success
		log.Printf("shard %v answered(%v/%v)", remoteShardId, len(com.acks), len(pyramid.LocalShard.RelatedIShard))
		if len(com.acks) == len(pyramid.LocalShard.RelatedIShard) {
			log.Printf("all shards have answered, success = %v", com.success)
			com.acks = make(map[int]bool)
			com.awake()
		}
		return true
	} else { // is i-shard
		com.debug("h-req" + idChain.IDC.NodeMap[msg.RemoteInfo].IpAddr)
		if com.BMainNode == nil && !com.innerIsBlocked() {
			com.BMainNode = idChain.IDC.NodeMap[msg.RemoteInfo]
			log.Printf("Link with %v", com.BMainNode.IpAddr)
			com.BlockInner()
			com.Request(&config.SuccessByte) // telling b-shards to continue
			return true
		} else {
			com.con.SendMsg(&message.Message{
				Type:       com.Type().RequestType(),
				Content:    *message.NewByteContent(&config.FailByte), //vars here is success or not
				RemoteInfo: idChain.IDC.NodeMap[msg.RemoteInfo].IpAddr,
			})
			log.Printf("Refuse to link to %v", idChain.IDC.NodeMap[msg.RemoteInfo].IpAddr)
			return false
		}
	}
}

func (com *CrossLockCom) Response(vars ...*[]byte) bool {
	com.reqLock.Lock()
	defer com.reqLock.Unlock()
	if pyramid.LocalShard.IsBShard() {
		if bytes.Equal(*vars[0], config.FailByte) {
			com.con.SendMsg(&message.Message{
				Type:       com.Type().ResponseType(),
				Content:    *message.NewByteContent(&config.FailByte),
				RemoteInfo: config.PyramidRelateI,
			})
		} else if bytes.Equal(*vars[0], config.SuccessByte) {
			com.con.SendMsg(&message.Message{
				Type:       com.Type().ResponseType(),
				Content:    *message.NewByteContent(&config.SuccessByte),
				RemoteInfo: config.PyramidRelateI,
			})
		} else {
			com.debug("resp")
			log.Panic()
		}
	} else {
		localShard := pyramid.LocalShard.Chain
		localHash := localShard.TopBlockHash[localShard.Id()].Bytes()
		com.con.SendMsg(&message.Message{
			Type:       com.Type().ResponseType(),
			Content:    *message.NewByteContent(&localHash),
			RemoteInfo: com.BMainNode.IpAddr,
		})
		com.BMainNode = nil
	}
	return true
}

func (com *CrossLockCom) HandleResponse(msg *message.Message) {
	com.reqLock.Lock()
	defer com.reqLock.Unlock()
	contents := msg.GetContents()
	remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	sender := idChain.IDC.NodeMap[msg.RemoteInfo]
	if remoteNode == idChain.RunningNode {
		if !bytes.Equal(contents[0], config.FailByte) {
			log.Panic()
		} else {
			com.UnblockInner()
			com.BMainNode = nil
			return
		}
	}
	if pyramid.LocalShard.IsBShard() {
		remoteSID := sender.ShardID
		log.Printf("get lock resp from shard %v", remoteSID)
		remoteTopHash := contents[0]
		localChain := pyramid.LocalShard.Chain
		localChain.TopBlockHash[int(remoteSID)] = *crypt.NewHash(remoteTopHash)
		localChain.TopHashToBeConfirmed[int(remoteSID)] = true
		com.acks[int(idChain.IDC.NodeMap[msg.RemoteInfo].ShardID)] = true
		if len(com.acks) == len(pyramid.LocalShard.RelatedIShard) {
			log.Printf("all shards have locked")
			com.acks = make(map[int]bool)
			com.awake()
		}
		return
	} else {
		com.debug("h-resp" + idChain.IDC.NodeMap[msg.RemoteInfo].IpAddr)
		if (com.innerIsBlocked() && com.BMainNode != sender) || com.BMainNode == nil {
			return // ignore other b-shards response.
		}
		if bytes.Equal(contents[0], config.SuccessByte) {
			log.Printf("Get %v's ack", sender.IpAddr)
			Interfaces.AppendDelayedCom(Interfaces.CrossLock, false)
		} else if bytes.Equal(contents[0], config.FailByte) {
			log.Printf("Stopping link to %v", com.BMainNode.IpAddr)
			com.UnblockInner()
			com.BMainNode = nil
		} else {
			log.Panic()
		}
	}
}

var innerBlocked = false

func (com *CrossLockCom) BlockInner() {
	com.con.GetProposalBuffer().SetThreshold(Proposals.CrossPriority)
	log.Printf("Inner txs blocked.")
	innerBlocked = true
}

func (com *CrossLockCom) innerIsBlocked() bool {
	return innerBlocked
}

func (com *CrossLockCom) UnblockInner() {
	com.con.GetProposalBuffer().SetThreshold(Proposals.LowestPriority)
	log.Printf("Inner blocking end.")
	innerBlocked = false
}
