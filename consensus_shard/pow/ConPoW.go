package pow

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"log"
	"math/big"
	"sync"
	"time"
)

type NakamotoPoW struct {
	id             int
	proposalBuffer *Proposals.ProposalBuffer
	domain         Interfaces.Domain
	proposeLock    sync.Mutex
	conf           *config.PoWConfig
	expTime        time.Duration
	lastUpdateTime time.Time
}

func NewPoWConsensus() *NakamotoPoW {
	ret := &NakamotoPoW{
		proposalBuffer: Proposals.NewProposalBuffer(),
		expTime:        1 * time.Minute,
		lastUpdateTime: time.Now(),
	}
	ret.Disable()
	return ret
}

func (con *NakamotoPoW) difficulty() *big.Int {
	return con.conf.Difficulty
}

func NewIdChainCon() *NakamotoPoW {
	ret := NewPoWConsensus()
	ret.conf = config.IdConfig.PowConf
	ret.id = config.IdMod
	ret.domain = idChain.IDC
	return ret
}

func (con *NakamotoPoW) UpdateDifficulty(head Block.Head) {
	// 计算上次更新以来经过的时间
	if head.GetNonce() == 0 || head.GetNonce()%con.conf.UpdatePeriod != 0 {

	}
	bTime := head.Time()
	elapsed := bTime.Sub(con.lastUpdateTime)

	// 将 elapsed 和 expTime 转为 big.Int（单位均为纳秒）
	elapsedInt := big.NewInt(int64(elapsed))
	expInt := big.NewInt(int64(con.expTime))

	// 计算新的 difficulty（目标值）： newDifficulty = oldDifficulty * elapsed / expTime
	newDifficulty := new(big.Int).Mul(con.difficulty(), elapsedInt)
	newDifficulty.Div(newDifficulty, expInt)
	log.Printf("difficulty %v update to %v", con.difficulty().String(), newDifficulty.String())
	con.conf.Difficulty = newDifficulty
	con.lastUpdateTime = bTime
	return
}

func (con *NakamotoPoW) HandleMsg(msg *message.Message) bool {
	switch msg.Type {
	// currently PoW broadcast is implemented by Comm.
	default:
		return false
	}
	//return true
}

func (con *NakamotoPoW) Id() int {
	return con.id
}

func (con *NakamotoPoW) SendMsg(message *message.Message) {
	launch.SendMsg(con, message)
}

func (con *NakamotoPoW) Propose() {
	// critical section
	con.Disable()
	pro := con.GetProposalBuffer().Pop()
	if pro == nil {
		con.Enable()
		time.Sleep(256 * time.Millisecond)
		go con.Propose()
		return
	}
	proType, proVars := pro.Get()
	preSuccess := Interfaces.Operations[proType].Prepare(proVars)
	if !preSuccess {
		log.Printf("----Schedule(%v) : prepare failed----", proType)
		con.Enable()
		return
	}
	content := message.NewStrContent(proType.String()).AppendByteContent(proVars...).Bytes()
	Interfaces.Communications[Interfaces.PoWBroadcast].Request(content)
	con.Enable()
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

func (con *NakamotoPoW) Enable() {
	con.proposeLock.Unlock()
}

func (con *NakamotoPoW) Disable() {
	con.proposeLock.Lock()
}
