package pbft

import (
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"blockEmulator/storage"
	"crypto/rsa"
	"encoding/json"
	"log"
)

func (con *ConPbft) NewCommitMessage(hash []byte, seq uint64) *message.Message {
	pre := message.Commit{
		Digest: hash,
		SeqID:  seq,
	}
	commitByte, err := json.Marshal(pre)
	if err != nil {
		log.Panic()
	}
	contents := message.NewByteContent(&commitByte)
	// broadcast
	return &message.Message{
		Type:       message.CCommit,
		Content:    *contents,
		RemoteInfo: con.legalNodesAddr,
	}
}

func (con *ConPbft) parseCommitMsg(msg *message.Message) (uint64, string) {
	contents := msg.GetContents()
	commitByte := contents[0]
	commitMsg := new(message.Commit)
	err := json.Unmarshal(commitByte, commitMsg)
	if err != nil {
		log.Panic(err)
	}
	return commitMsg.SeqID, string(commitMsg.Digest)
}

func (con *ConPbft) HandleCommit(msg *message.Message) {
	//log.Printf("con addr: %p, pNodeAddr: %p", con, con.pNode)
	seq, _ := con.parseCommitMsg(msg)
	if con.proposalStatus[seq] == Committed {
		return
	}
	// todo
	// seq should be check
	if con.CommitMsgs[seq] == nil {
		con.CommitMsgs[seq] = make(map[string]*message.Message)
	}
	con.CommitMsgs[seq][msg.RemoteInfo] = msg
	//log.Printf("Commit from %v(%v/%v)", idChain.IDC.NodeMap[msg.RemoteInfo].IpAddr, len(con.CommitMsgs[seq]), con.GetDomain().Threshold())
	con.HandleCommitMsgs()
}

func (con *ConPbft) HandleCommitMsgs() {
	con.Tic()
	if len(con.CommitMsgs[con.seq()]) < int(con.GetDomain().Threshold()) {
		// msg amount is not enough
		return
	}
	for _, msg := range con.CommitMsgs[con.seq()] {
		seq, key := con.parseCommitMsg(msg)
		sender := idChain.IDC.NodeMap[msg.RemoteInfo]
		{ // safety check
			_, commitCntExist := con.cntCommitConfirm[key]
			if !commitCntExist {
				con.cntCommitConfirm[key] = make(map[*rsa.PublicKey]bool)
			}
		}
		if sender.IsRunning() {
			con.cntCommitConfirm[key][sender.NodeId] = true
		}
		threshold := con.GetDomain().Threshold()
		cnt := uint64(0)
		for range con.cntCommitConfirm[key] {
			cnt++
		}

		if con.printFlag {
			log.Printf("Commit %v::Sender(%v),(%v/%v)\n", seq, sender.IpAddr, cnt, threshold)
		}
		if cnt >= threshold && con.isReply[seq] == false && con.isCommitBroadcast[key] == true {
			con.isReply[seq] = true
			//replyMsg := con.NewReplyMessage(seq, true) // must be called before Execute, to support mainNode changing
			{
				log.Printf(
					"\t\tRound %d (%v)executing......\n", con.seq(), con.proposalPool[seq].RequestType)
				storage.CommLogger.Writef("Round %d (%v)executing\n", con.seq(), con.proposalPool[seq].RequestType)
				RequestToExecute := con.proposalPool[seq]
				con.Execute(RequestToExecute)
				con.proposalStatus[seq] = Committed
				delete(con.PrePreMsgs, seq)
				delete(con.PrepareMsgs, seq)
				delete(con.CommitMsgs, seq)
				con.nxtSeq()
				go launch.ClearMsgBuffer()
				con.Tic()
				log.Printf("\t S%vRound %d (%v) finish. \n", idChain.RunningNode.ShardID, con.seq(), con.proposalPool[seq].RequestType)
			}
			if idChain.RunningNode != con.GetDomain().Main() {
				if msg := con.PrePreMsgs[con.seq()]; msg != nil {
					// pre-pre msg may arrive before this round execution, and will return false with its seq check.
					con.HandlePrePrepare(msg)
				}
			} else {
				con.Enable()
			}
			//log.Printf("---------Mode %v Round %v Begin-----------", con.id, con.seq())
			storage.CommLogger.Writef("%v Round %v begin", con.id, con.seq())
			return
		}
	}
}
