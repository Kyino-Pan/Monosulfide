package Proposals

import (
	"blockEmulator/message"
	"sync"
)

var (
	LowestPriority = 1024
	NormalPriority = 128
	Emergency      = 64
	CrossPriority  = 32
	//CrossPreparePriority = CrossPriority - 2
	CrossCommitPriority = CrossPriority - 1 // will only let cross commit be proposed.
)

// Operation with a smaller priority will be proposed earlier

type ProposalBuffer struct {
	pros           []*Proposal
	priority       map[message.RequestType]int // the smaller, the more prior
	lock           sync.RWMutex
	priorThreshold int
}

func (pq *ProposalBuffer) SetPriority(reqType message.RequestType, pri int) {
	pq.priority[reqType] = pri
}

func NewProposalBuffer() *ProposalBuffer {
	var pros []*Proposal
	pros = make([]*Proposal, 0)
	ret := &ProposalBuffer{
		pros:           pros,
		priority:       make(map[message.RequestType]int),
		priorThreshold: LowestPriority,
	}
	ret.pros = append(ret.pros, nil)
	return ret
}

func (pq *ProposalBuffer) SetThreshold(threshold int) {
	pq.lock.Lock()
	defer pq.lock.Unlock()
	pq.priorThreshold = threshold
}

func (pq *ProposalBuffer) get(index int) *Proposal {
	return pq.pros[index]
}

func (pq *ProposalBuffer) Amount() int {
	pq.lock.RLock()
	defer pq.lock.RUnlock()
	return len(pq.pros) - 1
}

func (pq *ProposalBuffer) len() int {
	return len(pq.pros)
}

func (pq *ProposalBuffer) less(i, j int) bool {
	// 优先级高的元素在堆中位置更靠前
	return pq.priority[pq.get(i).ReqType] < pq.priority[pq.get(j).ReqType]
}

func (pq *ProposalBuffer) swap(i, j int) {
	pq.pros[i], pq.pros[j] = pq.pros[j], pq.pros[i]
}

func (pq *ProposalBuffer) Push(x *Proposal) {
	pq.lock.Lock()
	defer pq.lock.Unlock()
	pq.pros = append(pq.pros, x)
	currIndex := pq.len() - 1
	for {
		if currIndex == 1 {
			break
		}
		father := currIndex / 2
		if pq.less(currIndex, father) {
			pq.swap(currIndex, father)
			currIndex = father
		} else {
			break
		}
	}
}

func (pq *ProposalBuffer) Pop() *Proposal {
	pq.lock.Lock()
	defer pq.lock.Unlock()
	if len(pq.pros) == 1 {
		//log.Println("No proposal")
		return nil
	}
	index := 1
	ret := pq.get(1)
	if pq.priority[ret.ReqType] > pq.priorThreshold {
		// block the proposals
		return nil
	}
	root := pq.get(len(pq.pros) - 1)
	pq.pros[1] = root
	pq.pros = pq.pros[:len(pq.pros)-1]
	for {
		l, r := index*2, index*2+1
		temp := l
		if r >= pq.len() {
			if l >= pq.len() {
				break
			}
			// else temp = l
		} else if pq.less(r, l) {
			temp = r
		}
		if pq.less(temp, index) {
			pq.swap(temp, index)
			index = temp
		} else {
			break
		}
	}
	return ret
}

func (pq *ProposalBuffer) GetThreshold() int {
	return pq.priorThreshold
}
