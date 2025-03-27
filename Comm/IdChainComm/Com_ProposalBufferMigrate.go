package IdChainComm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Opts"
	"blockEmulator/Proposals"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"log"
)

type MigrateProCom struct {
	con Interfaces.Consensus
}

func (com *MigrateProCom) Type() Interfaces.CommType {
	return Interfaces.MigratePro
}

func (com *MigrateProCom) Reset(con Interfaces.Consensus) Interfaces.CommType {
	com.con = con
	return com.Type()
}

func (com *MigrateProCom) Request(...*[]byte) bool {
	// is called when current node has just turned to normalNode from mainNode
	// will send unproposed proposal to the new mainNode
	// and replace them with dummyProposal
	if idChain.IsIdMainNode() {
		log.Println("WARNING::Calling migrate while being a main node")
		return false
	}
	// currently is used by identity chain only.
	amount := com.con.GetProposalBuffer().Amount()
	log.Printf("Migrating proposalbuffer(%v).\n", amount)
	com.con.SendMsg(com._generateMsg(uint64(amount)))
	return true
}

func (com *MigrateProCom) _generateMsg(num uint64) *message.Message {
	amount := crypt.UintToBytes(num)
	content := message.NewByteContent(amount)
	for {
		pro := com.con.GetProposalBuffer().Pop()
		if pro == nil {
			break
		}
		reqType, vars := pro.Get()
		content.AppendStrContent(reqType.String()).
			AppendByteContent(crypt.UintToBytes(uint64(len(vars)))).
			AppendByteContent(vars...)
	}
	return &message.Message{
		Type:       Interfaces.MigratePro.RequestType(),
		Content:    *content,
		RemoteInfo: com.con.GetDomain().Main().IpAddr,
	}
}

func (com *MigrateProCom) _parseMsg(msg *message.Message) []*Proposals.Proposal {
	contents := msg.GetContents()
	amount := crypt.BytesToUint(contents[0])
	cnt := 1
	proposals := make([]*Proposals.Proposal, 0)
	for i := uint64(0); i < amount; i++ {
		p := new(Proposals.Proposal)
		p.ReqType = message.RequestType(contents[cnt])
		p.Vars = make([]*[]byte, 0)
		cnt++
		paramAmount := crypt.BytesToUint(contents[cnt])
		cnt++
		for paramAmount != 0 {
			paramAmount--
			p.Vars = append(p.Vars, &contents[cnt])
			cnt++
		}
		proposals = append(proposals, p)
	}
	return proposals
}

func (com *MigrateProCom) HandleRequest(msg *message.Message) bool {
	if idChain.IsIdMainNode() {
		pros := com._parseMsg(msg)
		log.Printf("Recieve migrateMsg(%v)", len(pros))
		for _, p := range pros {
			Opts.Propose(com.con, p.ReqType, p.Vars...)
		}
		return true
	} else {
		msg.RemoteInfo = idChain.IDC.Main().IpAddr
		log.Printf("Redirect to the main node(%v)", msg.RemoteInfo)
		com.con.SendMsg(msg)
		return false
	}
}

func (com *MigrateProCom) Response(vars ...*[]byte) bool {
	panic("")
}

func (com *MigrateProCom) HandleResponse(msg *message.Message) {
	panic("")
}
