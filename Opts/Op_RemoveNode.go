package Opts

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"log"
)

type RemoveNodeOpt struct {
	con    Interfaces.Consensus
	nodeId string
}

func (op *RemoveNodeOpt) Reset(con Interfaces.Consensus) message.RequestType {
	con.GetProposalBuffer().SetPriority(message.RemoveNode, Proposals.NormalPriority)
	return message.RemoveNode
}

func (op *RemoveNodeOpt) Propose(vars ...*[]byte) {
	Propose(op.con, message.RemoveNode, vars...)
}
func (op *RemoveNodeOpt) PrepareAfterLock([]*[]byte) bool {
	return true
}

func (op *RemoveNodeOpt) Verify(vars [][]byte) bool {
	// todo
	nodeId := string(vars[0])
	if node := idChain.IDC.NodeMap[nodeId]; node != nil {
		return false
	}
	log.Printf("RemoveNode::no node in node map")
	return false
}

func (op *RemoveNodeOpt) Execute() {
	idChain.IDC.NodeMap[op.nodeId] = nil
	op.nodeId = ""
	return
}
