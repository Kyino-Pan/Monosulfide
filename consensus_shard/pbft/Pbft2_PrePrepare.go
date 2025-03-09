package pbft

import (
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"blockEmulator/storage"
	"encoding/json"
	"log"
	"time"
)

func (con *ConPbft) NewPrePrepareMsg(seq uint64,
	planType message.RequestType, vars ...*[]byte) *message.Message {
	//if !idChain.IsIdMainNode() {
	//	log.Panic("PrePre::WARNING::Can only be called by main node")
	//	return nil
	//}
	raws := message.NewByteContent(vars...)

	req := &message.Request{
		RequestType: planType,
		Msg:         message.RawMessage{Content: *raws},
		ReqTime:     time.Now(),
	}
	hash := crypt.GetDigest(req)
	prePrepareMsg := message.PrePrepare{
		RequestMsg: req,
		Digest:     hash,
		SeqID:      seq,
	}
	prePrepareJson, err := json.Marshal(prePrepareMsg)
	if err != nil {
		log.Panic()
	}
	content := message.NewByteContent(&prePrepareJson)
	msg := &message.Message{
		Type:       message.CPrePrepare,
		Content:    *content,
		RemoteInfo: con.legalNodesAddr,
	}
	con.proposalPool[seq] = req
	return msg
}

func (con *ConPbft) parsePrePrepareMsg(msg *message.Message) (uint64, *message.Request, []byte, *idChain.Node) {
	//fmt.Println("received the PrePrepare ...")
	// decode the message
	prePrepareMsg := new(message.PrePrepare)
	contents := msg.GetContents()
	err := json.Unmarshal(contents[0], prePrepareMsg)
	if err != nil {
		log.Panic(err)
	}
	req := prePrepareMsg.RequestMsg
	remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	return prePrepareMsg.SeqID, req, prePrepareMsg.Digest, remoteNode
}

func (con *ConPbft) HandlePrePrepare(msg *message.Message) {
	//log.Printf("PrePre")
	con.Tic()
	seq, req, hash, remoteNode := con.parsePrePrepareMsg(msg)
	if con.PrePreMsgs[seq] == nil {
		con.PrePreMsgs[seq] = msg
	} else if idChain.IDC.NodeMap[msg.RemoteInfo] != con.domain.Main() {
		log.Printf("%v, %v, %v, %v", req.RequestType, remoteNode.IpAddr, seq, con.seq())
		log.Panic("main node is malicious")
	}
	if req.RequestType == message.NewEpoch && idChain.RunningNode.IsWaiting() {
		con.setSeq(seq)
		idChain.RunningNode.Activating = true //pNode: waiting -> preparing
	}
	tempSeq := con.seq()
	if req.RequestType == message.NewEpoch && idChain.RunningNode.IsWaiting() {
		con.setSeq(seq)
		idChain.RunningNode.Activating = true //pNode: waiting -> preparing
	}
	result := con.verify(seq, req, hash, remoteNode)
	if result == false {
		if req.RequestType == message.NewEpoch && idChain.RunningNode.IsWaiting() {
			con.setSeq(tempSeq)
			idChain.RunningNode.Activating = false
			//roll back if propose is wrong.
		}
		return
	}
	//log.Printf("PrePre %v[%v]%v", seq, remoteNode.IpAddr, req.RequestType)
	storage.CommLogger.Writef("PrePre %v[%v]%v", seq, remoteNode.IpAddr, req.RequestType)

	vars := (*message.Content)(&req.Msg.Content).ParseContent()
	con.proposalPool[seq] = req
	con.proposalStatus[seq] = Unprepared
	con.isReply[seq] = false

	flag := Interfaces.Operations[req.RequestType].Verify(vars)
	if !flag {
		con.proposalPool[seq] = nil
		log.Printf("Verify false")
		con.SendMsg(con.NewReplyMessage(seq, false))
		return
	}
	prepareMessage := con.NewPrepareMessage(hash, seq)
	if flag {
		if idChain.RunningNode.IsRunning() {
			con.SendMsg(prepareMessage)
			// curr node itself is ready
		} else {
			prepareMessage.RemoteInfo = idChain.RunningNode.StrKey()
			con.HandlePrepare(prepareMessage)
		}
	}
}

func (con *ConPbft) verify(seq uint64, req *message.Request, hash []byte, remoteNode *idChain.Node) bool {
	if remoteNode != con.GetDomain().Main() {
		log.Panicf("Proposer should be %v, got %v", con.GetDomain().Main().IpAddr, remoteNode.IpAddr)
		return false
	}
	if idChain.RunningNode.IsWaiting() && req.RequestType != message.NewEpoch {
		return false
	}
	if string(hash) != string(crypt.GetDigest(req)) {
		log.Printf("PrePre::the digest is not consistent, so refuse to prepare. \n")
		return false
	}
	if con.seq() < seq {
		if _, exist := con.proposalPool[seq]; !exist {
			if con.seq() == seq-1 {
				log.Printf("PrePre::too early")
				time.Sleep(time.Duration(config.ViewChangeTime) / 8)
				launch.MsgBuffer = append(launch.MsgBuffer)
				return false
			} else {
				// too late
				log.Panic("pre-pre::should ask for previous proposals.")
				return false
			}
		}
		log.Printf("PrePre::the Sequence id is inconsistent(%v neq %v)\n", con.seq(), seq)
	} else if con.seq() > seq {
		log.Printf("?")
		return false
	}
	return true
}
