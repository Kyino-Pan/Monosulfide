package pbft

import (
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"crypto/rsa"
	"encoding/json"
	"log"
	"time"
)

func (con *ConPbft) NewPrepareMessage(hash []byte, seq uint64) *message.Message {
	pre := message.Prepare{
		Digest: hash,
		SeqID:  seq,
	}
	prepareByte, err := json.Marshal(pre)
	if err != nil {
		log.Panic()
	}
	contents := message.NewByteContent(&prepareByte)
	// broadcast
	return &message.Message{
		Type:       message.CPrepare,
		Content:    *contents,
		RemoteInfo: con.legalNodesAddr,
	}
}

func (con *ConPbft) parsePrepareMsg(msg *message.Message) ([]byte, uint64) {
	contents := msg.GetContents()
	prepareByte := contents[0]
	prepareMsg := new(message.Prepare)
	err := json.Unmarshal(prepareByte, prepareMsg)
	if err != nil {
		log.Panic(err)
	}
	hash := prepareMsg.Digest
	seq := prepareMsg.SeqID
	return hash, seq
}

func (con *ConPbft) HandlePrepare(msg *message.Message) {
	remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	if remoteNode == nil {
		log.Printf("%v", msg.RemoteInfo)
		log.Printf("%v", idChain.IDC.NodeMap)
		log.Panic()
	}
	//log.Printf("Prepare")
	hash, seq := con.parsePrepareMsg(msg)
	if con.proposalStatus[seq] >= Prepared {
		return // is already prepared.
	}
	if con.PrepareMsgs[seq] == nil {
		con.PrepareMsgs[seq] = make(map[string]*message.Message)
	}
	if remoteNode.IsRunning() {
		con.PrepareMsgs[seq][msg.RemoteInfo] = msg
	}
	//log.Printf("prepare from %v(%v/%v)", idChain.IDC.NodeMap[msg.RemoteInfo].IpAddr, len(con.PrepareMsgs[seq]), con.GetDomain().Threshold())
	con.HandlePrepareMsgs(hash)
}

func (con *ConPbft) HandlePrepareMsgs(hash []byte) {
	con.Tic()
	req, proposalExist := con.proposalPool[con.seq()]
	if len(con.PrepareMsgs[con.seq()]) < int(con.GetDomain().Threshold()) {
		// msg amount is not enough
		return
	}
	if !proposalExist {
		// msg amount is enough but hasn't received pre-pre
		log.Println("Waiting for pre-pre")
		return
	}
	con.proposalLock.RLock()
	defer con.proposalLock.RUnlock()
	for _, msg := range con.PrepareMsgs[con.seq()] {
		hash, seq := con.parsePrepareMsg(msg)
		key := string(hash)
		sender := idChain.IDC.NodeMap[msg.RemoteInfo]
		if !proposalExist {
			log.Printf("Prepare::Unknown prepare(seq %v), curr seq(%v)\n", seq, con.seq())
			time.Sleep(1 * time.Second)
			launch.BCMsgPool <- msg
			log.Panic()
			return
		}
		// record and count the prepared nodes
		if _, ok := con.cntPrepareConfirm[key]; !ok {
			con.cntPrepareConfirm[key] = make(map[*rsa.PublicKey]bool)
		}
		if sender.IsRunning() {
			con.cntPrepareConfirm[key][sender.NodeId] = true
		}
		cnt := uint64(0)
		for range con.cntPrepareConfirm[key] {
			cnt++
		}
		threshold := con.GetDomain().Threshold()
		if con.printFlag {
			log.Printf("Prepare %v[%v]:(%v/%v)", seq, sender.IpAddr, cnt, threshold)
		}
		if con.isCommitBroadcast[key] {
			log.Printf("Prepare::Already committed")
			log.Printf("WARNING::Shouldn't reach here")
			return
		}
		if req.RequestType == message.NewEpoch && idChain.RunningNode.IsPreparing() {
			threshold = 0
		}
		if cnt >= threshold && con.isCommitBroadcast[key] == false {
			commitMsg := con.NewCommitMessage(hash, seq)
			// generate commit and broadcast
			con.isCommitBroadcast[key] = true
			if idChain.RunningNode.IsRunning() {
				con.SendMsg(commitMsg)
			}
			//log.Printf("Prepare phase end, has reveived %v msgs.", len(con.CommitMsgs[seq]))
			con.proposalStatus[seq] = Prepared
			delete(con.PrepareMsgs, con.seq())
			con.HandleCommitMsgs()
			return
		}
	}
}
