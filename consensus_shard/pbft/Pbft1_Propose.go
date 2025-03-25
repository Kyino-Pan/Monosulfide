package pbft

import (
	"blockEmulator/Interfaces"
	"log"
	"time"
)

func (con *ConPbft) Propose() {
	// critical section
	con.Disable()
	pro := con.GetProposalBuffer().Pop()
	if pro == nil {
		con.Enable()
		time.Sleep(250 * time.Millisecond)
		go con.Propose()
		return
	}
	proType, proVars := pro.Get()
	preSuccess := Interfaces.Operations[proType].PrepareAfterLock(proVars)
	if !preSuccess {
		log.Printf("----Propose(%v) :%v prepare failed----", con.seq(), proType)
		con.Enable()
		return
	}
	request := con.NewPrePrepareMsg(con.seq(), proType, proVars...)
	if con.printFlag {
		log.Printf("----Propose(%v):%v(%v remaining)----", con.seq(), proType, con.GetProposalBuffer().Amount())
	}
	con.SendMsg(request)
}
