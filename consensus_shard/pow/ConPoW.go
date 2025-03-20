package pow

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/launch"
	"blockEmulator/message"
	"crypto/sha256"
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
	initDiffStr := "00000000FFFF0000000000000000000000000000000000000000000000000000"
	initDifficulty, ok := new(big.Int).SetString(initDiffStr, 16)
	if !ok {
		panic("解析初始难度失败")
	}
	ret := &NakamotoPoW{
		proposalBuffer: Proposals.NewProposalBuffer(),
		difficulty:     initDifficulty,
		ExpTime:        32 * time.Minute,
		lastUpdateTime: time.Now(),
	}
	return ret
}

func (con *NakamotoPoW) _updateDifficulty(bTime time.Time) {
	// 计算上次更新以来经过的时间
	elapsed := bTime.Sub(con.lastUpdateTime)

	// 将 elapsed 和 ExpTime 转为 big.Int（单位均为纳秒）
	elapsedInt := big.NewInt(int64(elapsed))
	expInt := big.NewInt(int64(con.ExpTime))

	// 计算新的 difficulty（目标值）： newDifficulty = oldDifficulty * elapsed / ExpTime
	newDifficulty := new(big.Int).Mul(con.difficulty, elapsedInt)
	newDifficulty.Div(newDifficulty, expInt)

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
	con.DisablePropose()
	pro := con.GetProposalBuffer().Pop()
	if pro == nil {
		con.EnablePropose()
		time.Sleep(250 * time.Millisecond)
		go con.Propose()
		return
	}
	proType, proVars := pro.Get()
	preSuccess := Interfaces.Operations[proType].PrepareAfterLock(proVars)
	if !preSuccess {
		log.Printf("----Propose(%v) : prepare failed----", proType)
		con.EnablePropose()
		return
	}
	content := message.NewStrContent(proType.String()).AppendByteContent(proVars...).Bytes()
	Interfaces.Communications[Interfaces.PoWBroadcast].Request(content)
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
	con.proposeLock.Lock()
}

func doubleSHA256(blockHeaderBytes []byte) [32]byte {
	firstHash := sha256.Sum256(blockHeaderBytes)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash
}

func isValidBlock(blockHeaderBytes []byte, target *big.Int) bool {
	hash := doubleSHA256(blockHeaderBytes)
	// 将哈希值转换为大整数（注意：比特币中的哈希通常以小端格式存储，转换时要留意字节顺序）
	hashInt := new(big.Int).SetBytes(hash[:])
	// 比较计算结果和目标值
	return hashInt.Cmp(target) < 0
}

var pow Interfaces.Consensus = &NakamotoPoW{}
