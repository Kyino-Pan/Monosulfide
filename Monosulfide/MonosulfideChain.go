package Monosulfide

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/storage"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

var finishCount = make(map[int]bool)

type MonosulfideChain struct {
	storage      *storage.Storage
	TxPool       *Tx.Pool
	lock         sync.RWMutex
	ownerShard   *Shard
	Blocks       map[crypt.Hash]Block.Block
	TopBlockHash map[int]crypt.Hash
	blockBuffer  map[int]map[crypt.Hash]*Block.FideBlock
	//blockAck     map[int]map[int]crypt.Hash // map[i][j]=k means shard i 已经确认shard j的第k个block
	nonce    uint64
	exp      *Record
	TxEnd    bool
	BlockGs  []Block.Block
	Weight   map[crypt.Hash]int
	Path     map[crypt.Hash][]crypt.Hash
	refs     map[crypt.Hash][]crypt.Hash
	updataT  map[crypt.Hash]time.Time
	Accept   map[int]bool
	_tempAnc map[crypt.Hash]map[crypt.Hash]Block.Block
}

type Record struct {
	IntraTxAmount   int
	CTxAmount       int
	IntraTxDelaySum time.Duration
	CTxDelaySum     time.Duration
	GenAmount       int
}

func (r *Record) Refresh() *Record {
	r.IntraTxAmount = 0
	r.CTxAmount = 0
	r.IntraTxDelaySum = 0
	r.CTxDelaySum = 0
	r.GenAmount = 0
	return r
}

func (ch *MonosulfideChain) Id() int {
	return ch.ownerShard.sid
}

func (ch *MonosulfideChain) GenerateBlock() Block.Block {
	ch.lock.RLock()
	defer ch.lock.RUnlock()
	cnt := 0
	subHashes := make([][]byte, config.FideConf.ShardAmount)
	blockBody := &Block.FideBody{
		SubBlocks: make([]*Block.SubBlock, config.FideConf.ShardAmount),
	}
	txsArray, blockIdx, sum := ch.TxPool.PackageRelayTxs()
	if sum == 0 {
		ch.Save()
		return nil
	}
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		txs := txsArray[i]
		subBlock := &Block.SubBlock{
			CHead: &Block.CBlockHead{
				RemoteBlockHash: ch.TopBlockHash[i],
				TxToot:          Tx.GenTxRoot(txs),
			},
			CBody: &Block.CBlockBody{Txs: txs},
		}
		blockBody.SubBlocks[i] = subBlock
		subHashes[i] = subBlock.Hash().Bytes()
		cnt += len(txs)
	}
	ret := &Block.FideBlock{
		H: &Block.FideHead{
			Nonce:        ch.Nonce() + 1,
			ShardIdx:     blockIdx,
			ParentHash:   ch.TopBlockHash[blockIdx],
			IntraTxRoot:  Tx.GenTxRoot(blockBody.Txs()),
			SubBlockRoot: Tx.GenMPTRoot(subHashes),
			StateRoot:    Tx.GenMPTRoot(subHashes), //todo
			Timestamp:    time.Now(),
		},
		B: blockBody,
	}
	if cnt == 0 && config.ManagerFinished == true {
		// 打包完所有的数据了
		ret.H.StateRoot = []byte("FINISH")
	}
	//log.Printf("%v", ret.Hash())
	return ret
}

func (ch *MonosulfideChain) Anc(h crypt.Hash) map[crypt.Hash]Block.Block {
	ret := make(map[crypt.Hash]Block.Block)
	tempBlock := ch.Blocks[h]
	if tempBlock == nil {
		log.Print(".")
	}
	sid := tempBlock.Head().(*Block.FideHead).ShardIdx
	for tempBlock != ch.BlockGs[sid] {
		ret[tempBlock.Hash()] = tempBlock
		tempBlock = ch.Blocks[tempBlock.Head().(*Block.FideHead).ParentHash]
	}
	ret[ch.BlockGs[sid].Hash()] = ch.BlockGs[sid]
	return ret
}

func (ch *MonosulfideChain) Valid(block *Block.FideBlock) bool {
	AncS := make([]map[crypt.Hash]Block.Block, config.FideConf.ShardAmount)
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		AncS[i] = ch.Anc(block.Ref(i))
	}
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		refI := ch.Blocks[block.Ref(i)].(*Block.FideBlock)
		if refI == ch.BlockGs[i] {
			continue
		}
		for j := 0; j < config.FideConf.ShardAmount; j++ {
			if AncS[j][refI.Ref(j)] == nil {
				log.Println("REF ERROR")
				return false
			}
		}
	}
	return true
}

func (ch *MonosulfideChain) Append(block *Block.FideBlock) {
	ch.lock.Lock()
	defer ch.lock.Unlock()

	var closeProcess = false
	//ch.TxPool.Print()
	isLegal := ch.VerifyBlock(block)
	sid := block.H.ShardIdx
	//tBegin := block.Head().Time()
	if isLegal {
		ch.storage.AddBlock(block)
		ch.Blocks[block.Hash()] = block
		ch.Path[block.H.ParentHash] = append(ch.Path[block.H.ParentHash], block.Hash()) // 记录孩子
		for i := 0; i < config.FideConf.ShardAmount; i++ {
			ch.refs[block.Ref(i)] = append(ch.refs[block.Ref(i)], block.Hash()) //记录被引用
		}
		tempBlock := block
		for ch.BlockGs[(sid)] != tempBlock {
			ch.Weight[tempBlock.Hash()] += 1
			ch.updataT[tempBlock.Hash()] = time.Now()
			tempBlock = ch.Blocks[tempBlock.H.ParentHash].(*Block.FideBlock)
		}

		good := true
		for i := 0; i < config.FideConf.ShardAmount; i++ {
			if ch.Anc(ch.TopBlockHash[i])[block.Ref(i)] == nil {
				good = false
			}
		}
		if block.H.ParentHash == ch.TopBlockHash[sid] && good {
			ch.TopBlockHash[sid] = block.Hash()
			ch.TxPool.RemoveTxs(block.B.Txs())
		} else {
			log.Println("Forked")
			ch.FindBestBranches()
		}
		if closeProcess {
			config.STOPPER <- true
		}
		//ch.TxPool.Print()
	} else {
		log.Panic()
	}
}

func (ch *MonosulfideChain) VerifyBlock(block *Block.FideBlock) bool {
	isLegal := ch._verifyRemoteBlock(block)

	if !ch.Valid(block) {
		isLegal = false
	}
	return isLegal
}

func (ch *MonosulfideChain) Nonce() uint64 {
	return ch.nonce
}

func (ch *MonosulfideChain) _verifyRemoteBlock(block *Block.FideBlock) bool {
	// Check sub-block root and hash-chain only.
	hashes := make([][]byte, config.FideConf.ShardAmount)
	for i, b := range block.B.SubBlocks {
		hashes[i] = b.Hash().Bytes()
	}
	if !bytes.Equal(block.H.SubBlockRoot, Tx.GenMPTRoot(hashes)) {
		log.Println("SubBlockRoot error")
		return false
	}
	return true
}

func (ch *MonosulfideChain) LockMoney(commit []*Tx.Transaction) {
	ch.TxPool.RemoveTxs(commit)
	//todo
}

func (ch *MonosulfideChain) recordCBlock(sid int, block *Block.FideBlock) {
	hash := block.Hash()
	ch.blockBuffer[sid][hash] = block
	if block.Nonce() == 0 {
		log.Panic()
	}
}

func (ch *MonosulfideChain) EncodedExp() *[]byte {
	bs, err := json.Marshal(ch.exp)
	if err != nil {
		log.Panic()
	}
	return &bs
}

var saved = false
var lock sync.Mutex

func (ch *MonosulfideChain) Save() {
	lock.Lock()
	defer lock.Unlock()
	if idChain.RunningNode != LocalShard.mainNode {
		return
	}
	if saved {
		return
	}
	saved = true
	conTime := time.Since(config.TxBegin)
	writer := storage.NewCsvWriter(0, "Fide-Result-"+strconv.Itoa(ch.Id())+".csv")
	go writer.Run()
	log.Printf("S%v %v", idChain.RunningNode.ShardID, idChain.RunningNode.IpAddr)
	//log.Print(ch.exp)
	numI := ch.exp.IntraTxAmount
	numC := ch.exp.CTxAmount
	sumI := time.Duration(0)
	sumC := time.Duration(0)
	cnt := 0
	bCnt := 0
	smallCnt := 0
	space := uint64(0)
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		ancs := ch.Anc(ch.TopBlockHash[i])
		for _, anc := range ancs {
			space += uint64(len(anc.Encode()))
			bCnt++
			if anc == ch.BlockGs[i] {
				continue
			}
			sub := anc.Body().(*Block.FideBody).SubBlocks[i]
			ITx := 0
			if sub != nil && sub.CBody != nil {
				ITx = len(sub.CBody.Txs)
			}
			cnt += len(anc.Body().Txs())
			if float64(cnt) <= (config.MaxBlockSize * 0.1) {
				smallCnt++
			}
			CTx := len(anc.Body().Txs()) - ITx
			numI += ITx
			numC += CTx
			for j, subB := range anc.Body().(*Block.FideBody).SubBlocks {
				if j == i {
					if subB.CBody != nil {
						for _, tx := range subB.CBody.Txs {
							sumI += anc.Head().Time().Sub(tx.Time)
						}
					}
				} else {
					if subB.CBody != nil {
						for _, tx := range subB.CBody.Txs {
							sumC += anc.Head().Time().Sub(tx.Time)
						}
					}
				}
			}
		}
	}
	avgTCL := (sumC + sumI).Milliseconds() / int64(numC+numI)
	TPS := float64(numC+numI) / conTime.Seconds()
	if idChain.RunningNode.Port() == config.MainPort {
		log.Printf("MONOSULFIDE REPORT")
		log.Printf("S%v, InjectSpeed = %v, DataSize = %vm PoWExpT = %v",
			config.FideConf.ShardAmount, config.InjectSpeed, config.TotalDataSize, config.PoWExpTime)
		log.Printf("Tol: %v txs\nAverage TCL: %v", cnt, avgTCL)
		log.Printf("IntraTCL = %v", (sumI / time.Duration(numI)).Milliseconds())
		log.Printf("CrossTCL = %v", (sumC / time.Duration(numC+1)).Milliseconds())
		log.Printf("TPS: %v ", TPS)
		log.Printf("Pivot: (%v/%v)", bCnt, len(ch.Blocks))
		log.Printf("Space : %v", space)
		log.Printf("Small: %v", smallCnt)
		log.Printf("Time : %v", time.Now().Sub(config.TxBegin))
	}
	writer.Writef(strconv.Itoa(numI))
	writer.Writef(strconv.Itoa(numC))
	writer.Writef(strconv.Itoa(ch.exp.GenAmount))
	writer.Writef(sumI.String())
	writer.Writef(sumC.String())
	writer.Writef("%d", (sumI / time.Duration(numI)).Milliseconds())
	writer.Writef("%d", (sumC / time.Duration(numC+1)).Milliseconds())
	writer.Writef("%v", avgTCL)            // average TCL
	writer.Writef("%v", TPS)               // TPS
	writer.Writef("%.2f", config.CommCalc) // KB
	writer.Writef(strconv.Itoa(config.TotalDataSize) + "Txs")
	writer.Writef("S" + strconv.Itoa(config.FideConf.ShardAmount) + "N" + strconv.Itoa(len(idChain.IDC.NodeMap)))
	//Interfaces.Communications[Interfaces.SyncFideBlock].Request()
	time.Sleep(config.ExitDelay)
	config.STOPPER <- true
	return
}

func NewFideChain(shard *Shard) *MonosulfideChain {
	ret := &MonosulfideChain{
		storage:      nil,
		lock:         sync.RWMutex{},
		ownerShard:   shard,
		Blocks:       make(map[crypt.Hash]Block.Block),
		TopBlockHash: make(map[int]crypt.Hash),
		blockBuffer:  make(map[int]map[crypt.Hash]*Block.FideBlock),
		//blockAck:     make(map[int]map[crypt.Hash]int),
		BlockGs: make([]Block.Block, config.ShardAmount),
		Weight:  make(map[crypt.Hash]int),
		Path:    make(map[crypt.Hash][]crypt.Hash),
		refs:    make(map[crypt.Hash][]crypt.Hash),
		Accept:  make(map[int]bool),
		updataT: make(map[crypt.Hash]time.Time),
		exp:     new(Record).Refresh(),
	}
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		ret.Accept[i] = true
	}
	baseBlock := &Block.FideBlock{
		H: &Block.FideHead{
			Nonce:      0,
			ParentHash: idChain.IDC.Chain.TopBlockHash[0],
		},
		B: &Block.FideBody{
			SubBlocks: make([]*Block.SubBlock, 0),
		},
	}
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		baseBlock.B.SubBlocks = append(baseBlock.B.SubBlocks, Block.EmptySubBlock())
	}
	fmt.Println(baseBlock.Hash().Bytes())
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		//ret.blockAck[i] = make(map[crypt.Hash]int)
		ret.blockBuffer[i] = make(map[crypt.Hash]*Block.FideBlock)
		ret.TopBlockHash[i] = baseBlock.Hash()
		ret.blockBuffer[i][baseBlock.Hash()] = baseBlock
		ret.BlockGs[i] = baseBlock
		ret.Blocks[baseBlock.Hash()] = baseBlock
	}
	ret.TxPool = Tx.NewTxPool(shard.sid)
	ret.EnableStorage(idChain.RunningNode.Port())
	return ret
}

func (ch *MonosulfideChain) EnableStorage(port string) {
	ch.storage = storage.NewStorage(port, uint64(ch.Id()))
}

func (ch *MonosulfideChain) FindBestBranches() {
	log.Printf("FindBestBranches")
	Bs := map[crypt.Hash]Block.Block{}
	refCnt := make(map[crypt.Hash]int)
	for k, b := range ch.Blocks {
		Bs[k] = b
		refCnt[k] = config.FideConf.ShardAmount
	}
	top := make([]crypt.Hash, config.FideConf.ShardAmount)
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		top[i] = ch.BlockGs[i].Hash()
	}
	seq := make([]crypt.Hash, 0)
	LockMap := make(map[crypt.Hash]bool)
	Removeable := make(map[crypt.Hash]Block.Block)
	for _, b := range ch.BlockGs {
		Removeable[b.Hash()] = b
	}
	for len(Removeable) != 0 {
		removed := false
		for _, b := range Removeable {
			if LockMap[b.Hash()] == false {
				removed = true
				delete(Bs, b.Hash())
				decs := ch.Path[b.Hash()]
				if len(decs) > 1 {
					for _, dec := range decs {
						LockMap[dec] = true
					}
				}
				for _, c := range ch.refs[b.Hash()] {
					refCnt[c]--
					if refCnt[c] == 0 {
						Removeable[c] = Bs[c]
					}
				}
				top[b.Head().(*Block.FideHead).ShardIdx] = b.Hash()
				seq = append(seq, b.Hash())
				delete(Removeable, b.Hash())
			}
		}
		if !removed {
			best := crypt.Hash{}
			maxW := 0
			lastT := time.Now()
			for _, b := range Removeable {
				w := ch.Weight[b.Hash()]
				if w > maxW {
					maxW = w
					best = b.Hash()
					lastT = ch.updataT[b.Hash()]
				} else if w == maxW {
					if ch.updataT[b.Hash()].Before(lastT) {
						best = b.Hash()
						lastT = ch.updataT[b.Hash()]
					}
				}
			}
			sid := ch.Blocks[best].Head().(*Block.FideHead).ShardIdx
			for _, b := range Removeable {
				if b.Head().(*Block.FideHead).ShardIdx == sid && b.Hash() != best {
					delete(Removeable, b.Hash())
					delete(LockMap, b.Hash())
				}
			}
			delete(LockMap, best)
		}
	}
	newCnt, oldCnt := 0, 0
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		if ch.TopBlockHash[i] != top[i] {
			log.Printf("%v, %v", ch.TopBlockHash[i], ch.Blocks[ch.TopBlockHash[i]] == nil)
			oldAnc := ch.Anc(ch.TopBlockHash[i])
			newAnc := ch.Anc(top[i])
			for k, oldB := range oldAnc {
				if newAnc[k] == nil {
					for _, tx := range oldB.Body().Txs() {
						ch.TxPool.Append(tx)
					}
					oldCnt++
				}
			}
			for k, newB := range newAnc {
				if oldAnc[k] == nil {
					ch.TxPool.RemoveTxs(newB.Body().Txs())
					newCnt++
				}
			}
		}
		log.Printf("Rollback %v blocks, new path got %v blocks", oldCnt, newCnt)
		ch.TopBlockHash[i] = top[i]
	}
}
