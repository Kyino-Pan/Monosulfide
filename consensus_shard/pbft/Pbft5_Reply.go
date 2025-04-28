package pbft

import (
	"blockEmulator/Interfaces"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"crypto/rsa"
	"encoding/json"
	"log"
)

// Reply is disabled currently.
func (con *ConPbft) NewReplyMessage(seq uint64, result bool) *message.Message {
	reply := message.Reply{
		MessageID: seq,
		Result:    result,
	}
	replyByte, err := json.Marshal(reply)
	if err != nil {
		log.Panic()
	}
	contents := message.NewByteContent(&replyByte)
	// broadcast
	return &message.Message{
		Type:       message.CReply,
		Content:    *contents,
		RemoteInfo: con.GetDomain().Main().IpAddr,
	}
}

func (con *ConPbft) parseReplyMessage(msg *message.Message) (uint64, *idChain.Node, bool) {
	contents := msg.GetContents()
	replyBytes := contents[0]
	replyMsg := new(message.Reply)
	err := json.Unmarshal(replyBytes, replyMsg)
	if err != nil {
		log.Panic(err)
	}
	senderNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	return replyMsg.MessageID, senderNode, replyMsg.Result
}

func (con *ConPbft) HandleReply(msg *message.Message) {
	con.Tic()
	if idChain.RunningNode != con.GetDomain().Main() {
		log.Panicf("%v %v", con.GetDomain().Main().IpAddr, idChain.RunningNode.IpAddr)
	}
	seq, senderNode, result := con.parseReplyMessage(msg)
	if con.seq() != seq {
		//log.Printf("Reply seq %v neq curr seq %v\n", seq, con.seq())
		return
	}
	if _, exist := con.cntReplyConfirm[seq]; !exist {
		con.cntReplyConfirm[seq] = make(map[*rsa.PublicKey]bool)
	}
	con.cntReplyConfirm[seq][senderNode.NodeId] = result
	successCnt := uint64(0)
	failedCnt := uint64(0)
	for _, flag := range con.cntReplyConfirm[seq] {
		if flag == true {
			successCnt++
		} else {
			failedCnt++
		}
	}
	threshold := con.GetDomain().Threshold()
	if con.proposalPool[seq].RequestType == message.EpochReset {
		threshold = idChain.IDC.GlobalThreshold()
	}
	if con.isReply[seq] == true {
		log.Printf("Reply %v::Enough\n", seq)
		return
	}
	if con.printFlag {
		log.Printf("Reply %v[%v]:(%v/%v)", seq, senderNode.IpAddr, successCnt, threshold)
	}
	if threshold <= successCnt {
		con.isReply[seq] = true
		log.Printf("----Primary %v Reply round %d executing \n", con.id, con.seq())
		con.Execute(con.proposalPool[seq])
		con.nxtSeq()

		log.Printf("----Primary %v round %v begin------", con.id, con.seq())
		//con.seqLock.Unlock()
		go launch.ClearMsgBuffer()
		if idChain.RunningNode != con.GetDomain().Main() { // may change status after executing
			Interfaces.Communications[Interfaces.MigratePro].Request()
		} else {
			//con.proposeLock.Unlock()
		}
	} else if threshold < failedCnt {
		//con.proposeLock.Unlock()
	} else {
	}
}
