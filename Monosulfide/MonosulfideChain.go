package Monosulfide

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
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
	nonce uint64
	exp   *Record
	TxEnd bool
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

func (mc *MonosulfideChain) Id() int {
	return mc.ownerShard.sid
}

func (mc *MonosulfideChain) GenerateBlock() Block.Block {
	mc.lock.RLock()
	defer mc.lock.RUnlock()
	cnt := 0
	subHashes := make([][]byte, config.FideConf.ShardAmount)
	blockBody := &Block.FideBody{
		Intra:     nil,
		SubBlocks: make([]*Block.SubBlock, config.FideConf.ShardAmount),
	}
	txsArray := mc.TxPool.PackageRelayTxs()
	amount := 0
	for _, txs := range txsArray {
		amount += len(txs)
	}
	for i := 0; i < config.FideConf.ShardAmount; i++ {
		// package txs to different shards from txPool
		if i == LocalShard.sid {
			innerTx := txsArray[i]
			blockBody.Intra = innerTx
			blockBody.SubBlocks[i] = Block.EmptySubBlock()
			cnt += len(innerTx)
			continue
		}
		txs := txsArray[i]
		subBlock := &Block.SubBlock{
			CHead: &Block.CBlockHead{
				RemoteBlockHash: mc.TopBlockHash[i].Bytes(),
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
			Nonce:        mc.Nonce() + 1,
			ParentHash:   mc.TopBlockHash[mc.Id()],
			IntraTxRoot:  Tx.GenTxRoot(blockBody.Txs()),
			SubBlockRoot: Tx.GenMPTRoot(subHashes),
			StateRoot:    nil, //todo
			Timestamp:    time.Now(),
		},
		B: blockBody,
	}
	if cnt == 0 && config.ManagerFinished == true {
		// 打包完所有的数据了
		ret.H.StateRoot = []byte("FINISH")
	}
	mc.exp.GenAmount += cnt
	//ret.Print()
	return ret
}

func (mc *MonosulfideChain) Append(block *Block.FideBlock) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	var closeProcess = false
	//mc.TxPool.Print()
	isLegal := mc.VerifyBlock(block)
	tBegin := block.Head().Time()
	if isLegal {
		if block.B.SubBlocks[mc.Id()].CHead == nil {
			// 本shard发布的块
			need := true
			if block.H.StateRoot != nil {
				log.Printf("shard%v %v (remaining%v)", mc.Id(), string(block.H.StateRoot),
					config.FideConf.ShardAmount-len(finishCount))
				need = false
				finishCount[mc.Id()] = true
				if len(finishCount) == config.FideConf.ShardAmount {
					if LocalShard.Main() == idChain.RunningNode {
						Interfaces.Communications[Interfaces.SyncFideBlock].Request()
						LocalShard.Chain.Save()
					}
					closeProcess = true
				}
				if config.FideConf.ShardAmount-len(finishCount) == 1 {
					for i := 0; i < config.FideConf.ShardAmount; i++ {
						if exist, _ := finishCount[i]; !exist {
							log.Printf("shard %v unfinished", i)
						}
					}
				}
			}
			// 如果是本shard发布的block，先只commit片内交易
			Txs := block.Body().Txs()
			mc.TxPool.RemoveTxs(Txs)

			// 记录交易延迟
			tIntraCost := time.Since(tBegin)
			mc.exp.IntraTxAmount += len(Txs)
			mc.exp.IntraTxDelaySum += tIntraCost * time.Duration(len(Txs))

			bHash := block.Hash()
			mc.Blocks[bHash] = block
			mc.TopBlockHash[mc.Id()] = bHash
			txToCommit := make([]*Tx.Transaction, 0)
			for sid, b := range block.B.SubBlocks {
				if sid == mc.Id() {
					continue
				}
				txToCommit = append(txToCommit, b.CBody.Txs...)
			}
			mc.LockMoney(txToCommit)
			mc.recordCBlock(mc.Id(), block)
			mc.nonce++
			Interfaces.Communications[Interfaces.SyncFideBlock].Request()
			if need {
				mc.storage.AddBlock(block)
			}
			block.Light()
		} else {
			// 如果是其他shard的block
			// check block-ack
			blockMaker := -1
			for i := 0; i < config.FideConf.ShardAmount; i++ {
				if block.B.SubBlocks[i].CHead == nil {
					blockMaker = i
					if i == mc.Id() {
						log.Panic()
						return
					}
					break
				}
			}
			if blockMaker == -1 {
				log.Panic()
				return
			}
			need := true
			if block.H.StateRoot != nil {
				need = false
				log.Printf("shard%v %v", blockMaker, string(block.H.StateRoot))
				finishCount[blockMaker] = true
				if len(finishCount) == config.FideConf.ShardAmount || config.TpsTest {
					if LocalShard.Main() == idChain.RunningNode {
						LocalShard.Chain.Save()
					}
					closeProcess = true
				}
			}
			mc.TopBlockHash[blockMaker] = block.Hash()
			for id, b := range block.B.SubBlocks {
				if id == blockMaker {
					continue
				}
				//mc.blockAck[blockMaker][id] = *crypt.NewHash(b.CHead.RemoteBlockHash)
				if id == mc.Id() {
					mc.TxPool.RemoveTxs(b.CBody.Txs)
					// 记录交易延迟
					CTxAmount := len(b.CBody.Txs)
					tCTx := time.Since(tBegin)
					mc.exp.CTxAmount += CTxAmount
					mc.exp.CTxDelaySum += tCTx * time.Duration(CTxAmount)
				}
			}
			if need && !config.ClassRelay {
				mc.storage.AddBlock(block.Light())
			}
		}
		if closeProcess {
			config.STOPPER <- true
		}
		//mc.TxPool.Print()
	} else {
		log.Panic()
	}
}

func (mc *MonosulfideChain) VerifyBlock(block *Block.FideBlock) bool {
	var isLegal bool
	if block.B.SubBlocks[mc.Id()] == nil {
		isLegal = mc._verifyLocalBlock(block)
	} else {
		isLegal = mc._verifyRemoteBlock(block)
	}
	return isLegal
}

func (mc *MonosulfideChain) Nonce() uint64 {
	return mc.nonce
}

func (mc *MonosulfideChain) _verifyLocalBlock(block *Block.FideBlock) bool {
	//todo nonce check

	// check pre-block hash
	if block.H.ParentHash != mc.TopBlockHash[mc.Id()] {
		log.Println("ParentHash error")
		return false
	}
	txAll := make([]*Tx.Transaction, 0)
	//check intraTxs
	if !bytes.Equal(Tx.GenTxRoot(block.B.Intra), block.H.IntraTxRoot) {
		log.Println("ConfirmedTxRoot error")
		return false
	}
	txAll = append(txAll, block.B.Intra...)

	//check subBlocks
	subHashes := make([][]byte, config.FideConf.ShardAmount)
	for i, subB := range block.B.SubBlocks {
		if i == mc.Id() {
			subHashes[i] = []byte("")
			continue
		}
		subHashes[i] = subB.Hash().Bytes()
		if !bytes.Equal(subB.CHead.TxToot, Tx.GenTxRoot(subB.CBody.Txs)) {
			log.Println("subB.TxToot error")
			return false
		}
		txAll = append(txAll, subB.CBody.Txs...)
	}
	if !bytes.Equal(Tx.GenMPTRoot(subHashes), block.H.SubBlockRoot) {
		log.Println("SubBlockRoot error")
		return false
	}

	//check tx validity
	if !Tx.IsVaildTxSet(txAll) {
		log.Println("Balance not enough")
		return false
	}
	return true
}

func (mc *MonosulfideChain) _verifyRemoteBlock(block *Block.FideBlock) bool {
	// determined the block proposer
	var blockMaker = -1
	for i, b := range block.B.SubBlocks {
		if b.CHead == nil {
			blockMaker = i
			break
		}
	}
	if blockMaker == -1 {
		log.Panic()
		return false
	}
	// Check sub-block root and hash-chain only.
	hashes := make([][]byte, config.FideConf.ShardAmount)
	for i, b := range block.B.SubBlocks {
		if i == blockMaker {
			continue
		}
		hashes[i] = b.Hash().Bytes()
	}
	if !bytes.Equal(block.H.SubBlockRoot, Tx.GenMPTRoot(hashes)) {
		log.Println("SubBlockRoot error")
		return false
	}
	if block.H.ParentHash != mc.TopBlockHash[blockMaker] {
		log.Println("Error seq")
	}
	return true
}

func (mc *MonosulfideChain) LockMoney(commit []*Tx.Transaction) {
	mc.TxPool.RemoveTxs(commit)
	//todo
}

func (mc *MonosulfideChain) recordCBlock(sid int, block *Block.FideBlock) {
	hash := block.Hash()
	mc.blockBuffer[sid][hash] = block
	if block.Nonce() == 0 {
		log.Panic()
	}
}

//func (sc *MonosulfideChain) ack(sid, ackId int, hash crypt.Hash) {
//	if b := sc.Blocks[hash]; b != nil {
//		//log.Printf("S%v B%v already commited.", sid, b.GetNonce())
//		return
//	}
//	for i, subBlock := range sc.blockBuffer[sid][hash].B.SubBlocks {
//		preAck := sc.blockAck[sid][i]
//		currBlock := sc.Blocks[hash]
//		if ackId == sc.sid() {
//			for {
//				if currBlock.GetNonce() == preAck {
//					break
//				}
//			}
//
//		}
//		sc.blockAck[sid][i] = preAck
//
//	}
//	sc.blockAck[sid][ackId][hash] += 1
//	if sc.blockAck[sid][hash] == config.FideConf.Threshold {
//		block := sc.blockBuffer[sid][hash]
//		tBegin := block.H.Timestamp
//		if sid == sc.sid() {
//			// 执行本shard的转出确认
//			for sid, b := range block.B.SubBlocks {
//				if sid == sc.sid() {
//					continue
//				}
//				sc.TxPool.RemoveTxs(b.CBody.Txs)
//			}
//			log.Printf("Local %v sub executed.", block.GetNonce())
//		} else {
//			//block.Print()
//			txs := block.B.SubBlocks[sc.sid()].CBody.Txs
//			sc.TxPool.RemoveTxs(txs)
//			// 记录交易延迟
//			CTxAmount := len(txs)
//
//			tCTx := time.Since(tBegin)
//			sc.exp.CTxAmount += CTxAmount
//			sc.exp.CTxDelaySum += tCTx * time.Duration(CTxAmount)
//
//			sc.Blocks[hash] = block
//			log.Printf("S%v B%v sub executed.", sid, block.GetNonce())
//		}
//		delete(sc.blockAck[sid], hash)
//		delete(sc.blockBuffer[sid], hash)
//		log.Printf("S%v B%v commited.", sid, block.GetNonce())
//	}
//}

func (mc *MonosulfideChain) EncodedExp() *[]byte {
	bs, err := json.Marshal(mc.exp)
	if err != nil {
		log.Panic()
	}
	return &bs
}

var saved = false
var lock sync.Mutex

func (mc *MonosulfideChain) Save() {
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
	writer := storage.NewCsvWriter(0, "Fide-Result-"+strconv.Itoa(mc.Id())+".csv")
	go writer.Run()
	log.Printf("S%v %v", idChain.RunningNode.ShardID, idChain.RunningNode.IpAddr)
	log.Print(mc.exp)
	numI := mc.exp.IntraTxAmount
	numC := mc.exp.CTxAmount
	sumI := mc.exp.IntraTxDelaySum
	sumC := mc.exp.CTxDelaySum
	writer.Writef(strconv.Itoa(numI))
	writer.Writef(strconv.Itoa(numC))
	writer.Writef(strconv.Itoa(mc.exp.GenAmount))
	writer.Writef(sumI.String())
	writer.Writef(sumC.String())
	writer.Writef("%d", (sumI / time.Duration(numI)).Milliseconds())
	writer.Writef("%d", (sumC / time.Duration(numC+1)).Milliseconds())
	writer.Writef("%d", ((sumC + sumI) / time.Duration(numC+numI)).Milliseconds()) // average TCL
	writer.Writef("%v", float64(numC+numI)/conTime.Seconds())                      // TPS
	writer.Writef("%.2f", config.CommCalc)                                         // KB
	writer.Writef(strconv.Itoa(config.TotalDataSize) + "Txs")
	writer.Writef("S" + strconv.Itoa(config.FideConf.ShardAmount) + "N" + strconv.Itoa(len(idChain.IDC.NodeMap)))
	Interfaces.Communications[Interfaces.SyncFideBlock].Request()
	time.Sleep(config.ExitDelay)
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
		exp: new(Record).Refresh(),
	}

	baseBlock := &Block.FideBlock{
		H: &Block.FideHead{
			Nonce:      0,
			ParentHash: idChain.IDC.Chain.TopBlockHash[0],
		},
		B: &Block.FideBody{
			Intra:     nil,
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
	}
	ret.TxPool = Tx.NewTxPool(shard.sid)
	ret.EnableStorage(idChain.RunningNode.Port())
	return ret
}

func (mc *MonosulfideChain) EnableStorage(port string) {
	mc.storage = storage.NewStorage(port, uint64(mc.Id()))
}
