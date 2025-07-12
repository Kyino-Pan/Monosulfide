package pow

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"golang.org/x/exp/rand"
	"log"
	"math"
	"sync"
	"time"
)

type EasyPoW struct {
	id             int
	proposalBuffer *Proposals.ProposalBuffer
	domain         Interfaces.Domain
	proposeLock    sync.Mutex
	conf           *config.PoWConfig
	expTime        time.Duration
	lastUpdateTime time.Time
	TestCnt        int64
	StartTime      time.Time
	MiningOpt      message.RequestType
}

func NewRelayCon() Interfaces.Consensus {
	ret := NewEasyPoWConsensus(message.RelayTx)
	config.PoWExpTime = config.PoWExpTime * time.Duration(config.MonoxideConf.ShardAmount)
	return ret
}
func NewFideCon() Interfaces.Consensus {
	ret := NewEasyPoWConsensus(message.FideTx)
	return ret
}

func (con *EasyPoW) Propose() {
	// critical section
	con.Disable()
	defer con.Enable()
	log.Println("Propose")
	pro := con.GetProposalBuffer().Pop()
	if pro == nil {
		time.Sleep(250 * time.Millisecond)
		go con.Propose()
		return
	}
	proType, proVars := pro.Get()
	preSuccess := Interfaces.Operations[proType].Prepare(proVars)
	if !preSuccess {
		log.Printf("----Schedule :%v prepare failed----", proType)
		return
	}
	content := message.NewStrContent(string(proType)).AppendByteContent(proVars...)
	con.SendMsg(&message.Message{
		Type:       message.EasyPoWBroadcast,
		Content:    *content,
		RemoteInfo: con.domain.BroadAddr(),
	})
}

func (con *EasyPoW) GetProposalBuffer() *Proposals.ProposalBuffer {
	return con.proposalBuffer
}

func (con *EasyPoW) GetDomain() Interfaces.Domain {
	return con.domain
}

func (con *EasyPoW) SetDomain(domain Interfaces.Domain) {
	con.domain = domain
}

func (con *EasyPoW) Tic() {
	rand.Seed(uint64(time.Now().UnixNano()))
	log.Printf("%v start mining", idChain.RunningNode.Port())
	con.StartTime = time.Now()
	for {
		// 生成一个指数分布的等待时间
		u := rand.Float64()
		wait := -math.Log(1-u) * float64(con.expTime) * float64(len(idChain.IDC.NodeMap))
		time.Sleep(time.Duration(wait))
		Interfaces.Operations[con.MiningOpt].Schedule()
		//Interfaces.Operations[message.FideTx].Schedule()
		//log.Printf("Mined a block, elapsed tests: %v", con.TestCnt)
		con.TestCnt = 0
	}
}

func (con *EasyPoW) Tok() time.Duration {
	log.Printf("SHOULD NOT CALLED")
	return time.Since(con.lastUpdateTime)
}

func (con *EasyPoW) Enable() {
	con.proposeLock.Unlock()
}

func (con *EasyPoW) Disable() {
	con.proposeLock.Lock()
}

func (con *EasyPoW) HandleMsg(m *message.Message) bool {
	contents := m.GetContents()
	switch m.Type {
	case message.EasyPoWBroadcast:
		reqType := message.RequestType(contents[0])
		vars := make([][]byte, 0)
		for i := 1; i < len(contents); i++ {
			vars = append(vars, contents[i])
		}
		if Interfaces.Operations[reqType].Verify(vars) {
			Interfaces.Operations[reqType].Execute()
		}
	default:
		log.Println(message.RelayTx, "EASY_POW_Invalid Message")
		return false
	}
	return true
}

func NewEasyPoWConsensus(mineOpt message.RequestType) *EasyPoW {
	ret := &EasyPoW{
		proposalBuffer: Proposals.NewProposalBuffer(),
		expTime:        config.PoWExpTime,
		lastUpdateTime: time.Now(),
		id:             config.RelayMod,
		MiningOpt:      mineOpt,
	}
	ret.Disable()
	return ret
}

func (con *EasyPoW) Id() int {
	return con.id
}

func (con *EasyPoW) SendMsg(msg *message.Message) {
	launch.SendMsg(con, msg)
}
