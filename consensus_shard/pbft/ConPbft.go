package pbft

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"crypto/rsa"
	"encoding/json"
	"log"
	"sync"
	"time"
)

const (
	IdMod   = config.IdMod
	PyrMod  = config.PyrMod
	FideMod = config.FideMod
)

var (
	Unprepared = 1
	Prepared   = 2
	Committed  = 3
)

type ConPbft struct {
	// pointer to PyramidNode data
	proposeLock       sync.Mutex                // used by main node
	proposalBuffer    *Proposals.ProposalBuffer // used by main node
	PrePreMsgs        map[uint64]*message.Message
	PrepareMsgs       map[uint64]map[string]*message.Message
	cntPrepareConfirm map[string]map[*rsa.PublicKey]bool // count the prepare confirm message, [messageHash][Node]bool
	CommitMsgs        map[uint64]map[string]*message.Message
	cntCommitConfirm  map[string]map[*rsa.PublicKey]bool // count the commit confirm message, [messageHash][Node]bool
	isCommitBroadcast map[string]bool
	//prepareLock       sync.Mutex
	//commitLock        sync.Mutex
	//replyLock         sync.Mutex                         // used by main node
	cntReplyConfirm map[uint64]map[*rsa.PublicKey]bool // used by main node
	idChainLock     sync.Mutex
	proposalPool    map[uint64]*message.Request // RequestHash to Request
	proposalStatus  map[uint64]int
	proposalLock    sync.RWMutex
	isReply         map[uint64]bool
	isFinished      map[uint64]bool // used by main node
	tempChainInfo   *message.Content
	legalNodesAddr  string
	printFlag       bool
	ProposalTimeCtr time.Time
	domain          Interfaces.Domain
	id              int // is used to verify the pbft belongs to which party.
	sequence        uint64
	manLock         sync.Mutex
	manCond         *sync.Cond
	Zilean          map[string]time.Time
}

func (con *ConPbft) Id() int {
	return con.id
}

func (con *ConPbft) HandleMsg(msg *message.Message) bool {
	switch msg.Type {
	case message.CPrePrepare:
		con.HandlePrePrepare(msg)
	case message.CPrepare:
		con.HandlePrepare(msg)
	case message.CCommit:
		con.HandleCommit(msg)
	case message.CReply:
		con.HandleReply(msg)
	case message.NodeSilence:
		go con.HandleNodeSilence(msg)
	default:
		return false
	}
	return true
}

func (con *ConPbft) EnablePropose() {
	con.proposeLock.Unlock()
}

func (con *ConPbft) DisablePropose() {
	con.proposeLock.Lock()
}

func (con *ConPbft) GetDomain() Interfaces.Domain {
	return con.domain
}

func (con *ConPbft) SetDomain(d Interfaces.Domain) {
	con.domain = d
}

func NewPbftConsensus() *ConPbft {
	ret := &ConPbft{
		proposalPool:      make(map[uint64]*message.Request),
		PrePreMsgs:        make(map[uint64]*message.Message),
		PrepareMsgs:       make(map[uint64]map[string]*message.Message),
		CommitMsgs:        make(map[uint64]map[string]*message.Message),
		cntPrepareConfirm: make(map[string]map[*rsa.PublicKey]bool),
		cntCommitConfirm:  make(map[string]map[*rsa.PublicKey]bool),
		//cntReplyConfirm:   make(map[uint64]map[*rsa.PublicKey]bool),
		isCommitBroadcast: make(map[string]bool),
		isReply:           make(map[uint64]bool),
		proposalBuffer:    Proposals.NewProposalBuffer(),
		proposalStatus:    make(map[uint64]int),
		Zilean:            make(map[string]time.Time),
	}
	ret.proposeLock.Lock()
	return ret
}

func NewIdChainCon() *ConPbft {
	ret := NewPbftConsensus()
	ret.legalNodesAddr = config.IdLegalAddr
	// register Operations here

	//ret.Communications[HeartBeat] = ret.NewHeartBeatCom()
	ret.domain = idChain.IDC
	ret.printFlag = false
	ret.id = config.IdMod
	return ret
}

func NewPyramidCon() *ConPbft {
	ret := NewPbftConsensus()
	ret.legalNodesAddr = config.PyrRunningAddr
	// register Operations here
	ret.printFlag = false
	ret.id = config.PyrMod
	return ret
}

func NewFideBehavior() Interfaces.Consensus {
	ret := NewPbftConsensus()
	ret.legalNodesAddr = config.FideRunningAddr
	ret.printFlag = false
	ret.id = config.FideMod
	return ret
}

func (con *ConPbft) seq() uint64 {
	return con.sequence
}

func (con *ConPbft) setSeq(vSeq uint64) {
	log.Println("Set called")
	con.sequence = vSeq
	return
}

func (con *ConPbft) nxtSeq() {
	con.sequence += 1
}

func (con *ConPbft) SyncResponse(msg *message.Message) {
	contents := (*message.Content)(&msg.Content).ParseContent()
	nodeInfo := new(idChain.Node)
	err := json.Unmarshal(contents[0], nodeInfo)
	if err != nil {
		return
	}
	chainInfo, err := json.Marshal(idChain.IDC)
	content := message.NewByteContent(&chainInfo)
	if err != nil {
		return
	}
	con.SendMsg(&message.Message{
		Type:       message.SyncIdChain,
		Content:    *content,
		RemoteInfo: nodeInfo.IpAddr,
	})
}

func (con *ConPbft) Execute(req *message.Request) {
	reqType := req.RequestType
	Interfaces.Operations[reqType].Execute()
}

func (con *ConPbft) HandleNodeSilence(msg *message.Message) {
	contents := msg.GetContents()
	nodeId := string(contents[0])
	if node := idChain.IDC.NodeMap[nodeId]; node != nil {
		node.Silence += 1
		print(node.Silence)
		if node.Silence >= idChain.IDC.Threshold() {
			Interfaces.Operations[message.RemoveNode].Propose(&contents[0])
		}
	}
}

func (con *ConPbft) SendMsg(m *message.Message) {
	launch.SendMsg(con, m)
}

func (con *ConPbft) PrepareTempChainInfo() {
	blocks := idChain.IDC.Chain.GetBlocks()
	chainInfo := message.NewByteContent(crypt.UintToBytes(uint64(len(blocks))))
	for i := len(blocks) - 1; i >= 0; i-- {
		tempBlock := blocks[i]
		byteBlock := tempBlock.Encode()
		chainInfo.AppendByteContent(&byteBlock)
	}
	con.tempChainInfo = chainInfo
}

func (con *ConPbft) EnableViewChange(i int) {
	interval := time.Duration(i) * time.Millisecond
	ticker := time.NewTicker(interval / 10) // 设置一个较小的时间间隔，以便定时检查状态
	defer ticker.Stop()
	for range ticker.C {
		t := con.Tok()
		if t >= interval {
			Interfaces.Communications[Interfaces.ViewChange].Request()
			con.Tic()
		} else if t >= interval/2 && con.GetDomain().Main() == idChain.RunningNode {
			Interfaces.Propose(con, message.Empty)
			con.Tic()
		} else if t >= interval/2 && con.GetDomain().Main().IpAddr == launch.Listener.GetLocalAddr() {
			panic("")
		}
	}
}

func (con *ConPbft) Tic() {
	con.manLock.Lock()
	defer con.manLock.Unlock()
	con.Zilean[config.ViewTrigger] = time.Now()
}
func (con *ConPbft) Tok() time.Duration {
	con.manLock.Lock()
	defer con.manLock.Unlock()
	return time.Since(con.Zilean[config.ViewTrigger])
}

func (con *ConPbft) GetProposalBuffer() *Proposals.ProposalBuffer {
	return con.proposalBuffer
}

func Init() {
	idChain.Init(launch.Listener.GetListenPort())
	Interfaces.Con[IdMod] = NewIdChainCon()
	Interfaces.Con[PyrMod] = NewPyramidCon()
	Interfaces.Con[FideMod] = NewFideBehavior()
}
