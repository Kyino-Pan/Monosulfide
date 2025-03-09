package pbft

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/message"
	"log"
	"time"
)

func (con *ConPbft) Propose(reqType message.RequestType, vars ...*[]byte) {
	//con.seqLock.Lock()
	con.GetProposalBuffer().Push(&Proposals.Proposal{
		ReqType: reqType,
		Vars:    vars,
	})
	//log.Printf("%v add to proBuffer, remaining %v", reqType, con.GetProposalBuffer().Amount())
	go con.innerPropose()
}

var cnt = 0

func (con *ConPbft) innerPropose() {
	// critical section
	con.DisablePropose()
	cnt++
	//round := cnt
	//log.Printf("%vget propose lock, current threshold is %v", round, con.GetProposalBuffer().GetThreshold())
	Interfaces.ClearComBuffer()
	pro := con.GetProposalBuffer().Pop()
	if pro == nil {
		//log.Printf("%vrelease propose lock", round)
		con.EnablePropose()
		time.Sleep(250 * time.Millisecond)
		go con.innerPropose()
		return
	}
	proType, proVars := pro.Get()
	con.ProposalTimeCtr = time.Now() // record time
	preSuccess := Interfaces.Operations[proType].PrepareAfterLock(proVars)
	if !preSuccess {
		log.Printf("----Propose(%v) :%v prepare failed----", con.seq(), proType)
		con.EnablePropose()

		return
	}
	request := con.NewPrePrepareMsg(con.seq(), proType, proVars...)
	if con.printFlag {
		log.Printf("----Propose(%v):%v(%v remaining)----", con.seq(), proType, con.GetProposalBuffer().Amount())
	}
	//con.isReply[con.seq()] = false
	con.SendMsg(request)
}
