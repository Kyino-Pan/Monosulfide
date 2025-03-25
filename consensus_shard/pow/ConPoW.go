package pow

import (
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

	difficulty     *big.Int
	ExpTime        time.Duration
	lastUpdateTime time.Time
}

func NewPoWConsensus() *NakamotoPoW {
	ret := &NakamotoPoW{
		proposalBuffer: Proposals.NewProposalBuffer(),
		difficulty:     config.IdConfig.Difficulty, // 是同一个对象
		ExpTime:        1 * time.Minute,
		lastUpdateTime: time.Now(),
	}
	ret.Disable()
	return ret
}

func NewIdChainCon() *NakamotoPoW {
	ret := NewPoWConsensus()
	ret.id = config.IdMod
	ret.domain = idChain.IDC
	return ret
}

func (con *NakamotoPoW) UpdateDifficulty(bTime time.Time) {
	// 计算上次更新以来经过的时间
	elapsed := bTime.Sub(con.lastUpdateTime)

	// 将 elapsed 和 ExpTime 转为 big.Int（单位均为纳秒）
	elapsedInt := big.NewInt(int64(elapsed))
	expInt := big.NewInt(int64(con.ExpTime))

	// 计算新的 difficulty（目标值）： newDifficulty = oldDifficulty * elapsed / ExpTime
	newDifficulty := new(big.Int).Mul(con.difficulty, elapsedInt)
	newDifficulty.Div(newDifficulty, expInt)
	log.Printf("difficulty %v update to %v", con.difficulty.String(), newDifficulty.String())
	con.difficulty = newDifficulty
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
		time.Sleep(250 * time.Millisecond)
		go con.Propose()
		return
	}
	proType, proVars := pro.Get()
	preSuccess := Interfaces.Operations[proType].PrepareAfterLock(proVars)
	if !preSuccess {
		log.Printf("----Propose(%v) : prepare failed----", proType)
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
