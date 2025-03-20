package Interfaces

import (
	"blockEmulator/Proposals"
	"blockEmulator/message"
)

type Operation interface {
	Reset(con Consensus) message.RequestType
	// Propose When Propose is called, the proposal is sent to the buffer instead of proposing immediately.
	// So if vars may change after other proposal executed, you should init them in PrepareAfterLock
	// Furthermore, if you want the Propose runs immediately, priority should set to purpose num
	// and the Propose should be called during previous Execute WITHOUT go
	Propose(vars ...*[]byte)
	// PrepareAfterLock The proposal may not be executed at once,
	// so some sensitive parameter(such as block) should be packaged after the proposal is selected.
	PrepareAfterLock(vars []*[]byte) bool
	// Verify is called to verify whether the parameters are valid.
	Verify(vars [][]byte) bool
	Execute()
}

var Operations = make(map[message.RequestType]Operation)

func TransVars(vars []*[]byte) [][]byte {
	temp := make([][]byte, 0)
	for _, v := range vars {
		temp = append(temp, *v)
	}
	return temp
}

func Propose(con Consensus, reqType message.RequestType, vars ...*[]byte) {
	con.GetProposalBuffer().Push(&Proposals.Proposal{
		ReqType: reqType,
		Vars:    vars,
	})
	//log.Printf("%v add to proBuffer, remaining %v", reqType, con.GetProposalBuffer().Amount())
	go con.Propose()
}
