package pow

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/launch"
	"blockEmulator/message"
	"log"
	"sync"
	"time"
)

type NakamotoPoW struct {
	id             int
	proposalBuffer *Proposals.ProposalBuffer
	domain         Interfaces.Domain
	proposeLock    sync.Mutex

	difficulty     uint64
	TPer64B        time.Duration // time per 64Block
	ExpTime        time.Duration
	lastUpdateTime time.Time
}

func (con *NakamotoPoW) _updateDifficulty(bTime time.Time) {
	multi := float64(bTime.Sub(con.lastUpdateTime)) / float64(con.ExpTime)
	con.difficulty = uint64(float64(con.difficulty) * multi)
}

func (con *NakamotoPoW) HandleMsg(message *message.Message) bool {
	//TODO implement me
	panic("implement me")
}

func (con *NakamotoPoW) Id() int {
	return con.id
}

func (con *NakamotoPoW) SendMsg(message *message.Message) {
	launch.SendMsg(con, message)
}

func (con *NakamotoPoW) Propose(requestType message.RequestType, vars ...*[]byte) {
	if len(vars) == 0 {
		log.Panic("Vars should contain 1 at least vars to calc PoW")
	} else {
		con.GetProposalBuffer().Push(&Proposals.Proposal{
			ReqType: requestType,
			Vars:    vars,
		})
		go con.innerPropose()
	}
}

func (con *NakamotoPoW) innerPropose() {
	con.DisablePropose()
	Interfaces.ClearComBuffer()
	pro := con.GetProposalBuffer().Pop()
	if pro == nil {
		//log.Printf("%vrelease propose lock", round)
		con.EnablePropose()
		time.Sleep(250 * time.Millisecond)
		go con.innerPropose()
		return
	}

}

func (con *NakamotoPoW) GetProposalBuffer() *Proposals.ProposalBuffer {
	return con.proposalBuffer
}

func (con *NakamotoPoW) GetDomain() Interfaces.Domain {
	return con.domain
}

func (con *NakamotoPoW) SetDomain(domain Interfaces.Domain) {
	con.domain = domain
}

func (con *NakamotoPoW) Tic() {
	panic("PoW does not require Tic & Tok")
}

func (con *NakamotoPoW) Tok() time.Duration {
	panic("PoW does not require Tic & Tok")
}

func (con *NakamotoPoW) EnablePropose() {
	con.proposeLock.Unlock()
}

func (con *NakamotoPoW) DisablePropose() {
	//TODO implement me
	panic("implement me")
}
