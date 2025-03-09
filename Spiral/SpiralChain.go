package Spiral

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

type SpiChain struct {
	storage      *storage.Storage
	TxPool       *Tx.Pool
	lock         sync.RWMutex
	ownerShard   *Shard
	Blocks       map[crypt.Hash]Block.Block
	TopBlockHash map[int]crypt.Hash
	blockBuffer  map[int]map[crypt.Hash]*Block.SpiralBlock
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

func (sc *SpiChain) Id() int {
	return sc.ownerShard.Id
}

func (sc *SpiChain) GenerateBlock() Block.Block {
	sc.lock.RLock()
	defer sc.lock.RUnlock()
	cnt := 0
	subHashes := make([][]byte, config.SpiConf.ShardAmount)
	blockBody := &Block.SpiralBody{
		Intra:     nil,
		SubBlocks: make([]*Block.SubBlock, config.SpiConf.ShardAmount),
	}
	txsArray := sc.TxPool.PackageRelayTxs()
	amount := 0
	for _, txs := range txsArray {
		amount += len(txs)
	}
	for i := 0; i < config.SpiConf.ShardAmount; i++ {
		// package txs to different shards from txPool
		if i == LocalShard.Id {
			innerTx := txsArray[i]
			blockBody.Intra = innerTx
			blockBody.SubBlocks[i] = Block.EmptySubBlock()
			cnt += len(innerTx)
			continue
		}
		txs := txsArray[i]
		subBlock := &Block.SubBlock{
			CHead: &Block.CBlockHead{
				RemoteBlockHash: sc.TopBlockHash[i].Bytes(),
				TxToot:          Tx.GenTxRoot(txs),
			},
			CBody: &Block.CBlockBody{Txs: txs},
		}
		blockBody.SubBlocks[i] = subBlock
		subHashes[i] = subBlock.Hash().Bytes()
		cnt += len(txs)
	}
	ret := &Block.SpiralBlock{
		H: &Block.SpiralHead{
			Nonce:        sc.Nonce() + 1,
			ParentHash:   sc.TopBlockHash[sc.Id()],
			IntraTxRoot:  Tx.GenTxRoot(blockBody.Txs()),
			SubBlockRoot: Tx.GenMPTRoot(subHashes),
			StateRoot:    nil, //todo
			Time:         time.Now(),
		},
		B: blockBody,
	}
	if cnt == 0 && config.ManagerFinished == true {
		// 打包完所有的数据了
		ret.H.StateRoot = []byte("FINISH")
	}
	sc.exp.GenAmount += cnt
	//ret.Print()
	return ret
}

func (sc *SpiChain) Append(block *Block.SpiralBlock) {
	sc.lock.Lock()
	defer sc.lock.Unlock()

	var closeProcess = false
	//sc.TxPool.Print()
	isLegal := sc.VerifyBlock(block)
	tBegin := block.H.Time
	if isLegal {
		if block.B.SubBlocks[sc.Id()].CHead == nil {
			// 本shard发布的块
			need := true
			if block.H.StateRoot != nil {
				log.Printf("shard%v %v (remaining%v)", sc.Id(), string(block.H.StateRoot),
					config.SpiConf.ShardAmount-len(finishCount))
				need = false
				finishCount[sc.Id()] = true
				if len(finishCount) == config.SpiConf.ShardAmount || config.TPS_TEST {
					if LocalShard.Main() == idChain.RunningNode {
						Interfaces.Communications[Interfaces.SyncSpiBlock].Request()
						LocalShard.Chain.Save()
					}
					closeProcess = true
				}
				if config.SpiConf.ShardAmount-len(finishCount) == 1 {
					for i := 0; i < config.SpiConf.ShardAmount; i++ {
						if exist, _ := finishCount[i]; !exist {
							log.Printf("shard %v unfinished", i)
						}
					}
				}
			}
			// 如果是本shard发布的block，先只commit片内交易
			Txs := block.Body().Txs()
			sc.TxPool.RemoveTxs(Txs)

			// 记录交易延迟
			tIntraCost := time.Since(tBegin)
			sc.exp.IntraTxAmount += len(Txs)
			sc.exp.IntraTxDelaySum += tIntraCost * time.Duration(len(Txs))

			bHash := block.Hash()
			sc.Blocks[bHash] = block
			sc.TopBlockHash[sc.Id()] = bHash
			txToCommit := make([]*Tx.Transaction, 0)
			for sid, b := range block.B.SubBlocks {
				if sid == sc.Id() {
					continue
				}
				txToCommit = append(txToCommit, b.CBody.Txs...)
			}
			sc.LockMoney(txToCommit)
			sc.recordCBlock(sc.Id(), block)
			sc.nonce++
			Interfaces.Communications[Interfaces.SyncSpiBlock].Request()
			if need {
				sc.storage.AddBlock(block)
			}
			block.Light()
		} else {
			// 如果是其他shard的block
			// check block-ack
			blockMaker := -1
			for i := 0; i < config.SpiConf.ShardAmount; i++ {
				if block.B.SubBlocks[i].CHead == nil {
					blockMaker = i
					if i == sc.Id() {
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
				if len(finishCount) == config.SpiConf.ShardAmount || config.TPS_TEST {
					if LocalShard.Main() == idChain.RunningNode {
						LocalShard.Chain.Save()
					}
					closeProcess = true
				}
			}
			sc.TopBlockHash[blockMaker] = block.Hash()
			for id, b := range block.B.SubBlocks {
				if id == blockMaker {
					continue
				}
				//sc.blockAck[blockMaker][id] = *crypt.NewHash(b.CHead.RemoteBlockHash)
				if id == sc.Id() {
					sc.TxPool.RemoveTxs(b.CBody.Txs)
					// 记录交易延迟
					CTxAmount := len(b.CBody.Txs)
					tCTx := time.Since(tBegin)
					sc.exp.CTxAmount += CTxAmount
					sc.exp.CTxDelaySum += tCTx * time.Duration(CTxAmount)
				}
			}
			if need && !config.ClassRelay {
				sc.storage.AddBlock(block.Light())
			}
		}
		if closeProcess {
			config.STOPPER <- true
		}
		//sc.TxPool.Print()
	} else {
		log.Panic()
	}
}

func (sc *SpiChain) VerifyBlock(block *Block.SpiralBlock) bool {
	var isLegal bool
	if block.B.SubBlocks[sc.Id()] == nil {
		isLegal = sc._verifyLocalBlock(block)
	} else {
		isLegal = sc._verifyRemoteBlock(block)
	}
	return isLegal
}

func (sc *SpiChain) Nonce() uint64 {
	return sc.nonce
}

func (sc *SpiChain) _verifyLocalBlock(block *Block.SpiralBlock) bool {
	//todo nonce check

	// check pre-block hash
	if block.H.ParentHash != sc.TopBlockHash[sc.Id()] {
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
	subHashes := make([][]byte, config.SpiConf.ShardAmount)
	for i, subB := range block.B.SubBlocks {
		if i == sc.Id() {
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

func (sc *SpiChain) _verifyRemoteBlock(block *Block.SpiralBlock) bool {
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
	hashes := make([][]byte, config.SpiConf.ShardAmount)
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
	if block.H.ParentHash != sc.TopBlockHash[blockMaker] {
		log.Println("Error seq")
	}
	return true
}

func (sc *SpiChain) LockMoney(commit []*Tx.Transaction) {
	sc.TxPool.RemoveTxs(commit)
	//todo
}

func (sc *SpiChain) recordCBlock(sid int, block *Block.SpiralBlock) {
	hash := block.Hash()
	sc.blockBuffer[sid][hash] = block
	if block.Nonce() == 0 {
		log.Panic()
	}
}

//func (sc *SpiChain) ack(sid, ackId int, hash crypt.Hash) {
//	if b := sc.Blocks[hash]; b != nil {
//		//log.Printf("S%v B%v already commited.", sid, b.Nonce())
//		return
//	}
//	for i, subBlock := range sc.blockBuffer[sid][hash].B.SubBlocks {
//		preAck := sc.blockAck[sid][i]
//		currBlock := sc.Blocks[hash]
//		if ackId == sc.Id() {
//			for {
//				if currBlock.Nonce() == preAck {
//					break
//				}
//			}
//
//		}
//		sc.blockAck[sid][i] = preAck
//
//	}
//	sc.blockAck[sid][ackId][hash] += 1
//	if sc.blockAck[sid][hash] == config.SpiConf.Threshold {
//		block := sc.blockBuffer[sid][hash]
//		tBegin := block.H.Time
//		if sid == sc.Id() {
//			// 执行本shard的转出确认
//			for sid, b := range block.B.SubBlocks {
//				if sid == sc.Id() {
//					continue
//				}
//				sc.TxPool.RemoveTxs(b.CBody.Txs)
//			}
//			log.Printf("Local %v sub executed.", block.Nonce())
//		} else {
//			//block.Print()
//			txs := block.B.SubBlocks[sc.Id()].CBody.Txs
//			sc.TxPool.RemoveTxs(txs)
//			// 记录交易延迟
//			CTxAmount := len(txs)
//
//			tCTx := time.Since(tBegin)
//			sc.exp.CTxAmount += CTxAmount
//			sc.exp.CTxDelaySum += tCTx * time.Duration(CTxAmount)
//
//			sc.Blocks[hash] = block
//			log.Printf("S%v B%v sub executed.", sid, block.Nonce())
//		}
//		delete(sc.blockAck[sid], hash)
//		delete(sc.blockBuffer[sid], hash)
//		log.Printf("S%v B%v commited.", sid, block.Nonce())
//	}
//}

func (sc *SpiChain) EncodedExp() *[]byte {
	bs, err := json.Marshal(sc.exp)
	if err != nil {
		log.Panic()
	}
	return &bs
}

var saved = false
var lock sync.Mutex

func (sc *SpiChain) Save() {
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
	writer := storage.NewCsvWriter(0, "Spiral-Result-"+strconv.Itoa(sc.Id())+".csv")
	go writer.Run()
	log.Printf("S%v %v", idChain.RunningNode.ShardID, idChain.RunningNode.IpAddr)
	log.Print(sc.exp)
	numI := sc.exp.IntraTxAmount
	numC := sc.exp.CTxAmount
	sumI := sc.exp.IntraTxDelaySum
	sumC := sc.exp.CTxDelaySum
	writer.Writef(strconv.Itoa(numI))
	writer.Writef(strconv.Itoa(numC))
	writer.Writef(strconv.Itoa(sc.exp.GenAmount))
	writer.Writef(sumI.String())
	writer.Writef(sumC.String())
	writer.Writef("%d", (sumI / time.Duration(numI)).Milliseconds())
	writer.Writef("%d", (sumC / time.Duration(numC+1)).Milliseconds())
	writer.Writef("%d", ((sumC + sumI) / time.Duration(numC+numI)).Milliseconds()) // average TCL
	writer.Writef("%v", float64(numC+numI)/conTime.Seconds())                      // TPS
	writer.Writef("%.2f", config.CommCalc)                                         // KB
	writer.Writef(strconv.Itoa(config.TotalDataSize) + "Txs")
	writer.Writef("S" + strconv.Itoa(config.SpiralShardAmount) + "N" + strconv.Itoa(len(idChain.IDC.NodeMap)))
	Interfaces.Communications[Interfaces.SyncSpiBlock].Request()
	time.Sleep(config.ExitDelay)
	return
}

func NewSpiChain(shard *Shard) *SpiChain {
	ret := &SpiChain{
		storage:      nil,
		lock:         sync.RWMutex{},
		ownerShard:   shard,
		Blocks:       make(map[crypt.Hash]Block.Block),
		TopBlockHash: make(map[int]crypt.Hash),
		blockBuffer:  make(map[int]map[crypt.Hash]*Block.SpiralBlock),
		//blockAck:     make(map[int]map[crypt.Hash]int),
		exp: new(Record).Refresh(),
	}

	baseBlock := &Block.SpiralBlock{
		H: &Block.SpiralHead{
			Nonce:      0,
			ParentHash: idChain.IDC.Chain.TopBlockHash[0],
		},
		B: &Block.SpiralBody{
			Intra:     nil,
			SubBlocks: make([]*Block.SubBlock, 0),
		},
	}
	for i := 0; i < config.SpiConf.ShardAmount; i++ {
		baseBlock.B.SubBlocks = append(baseBlock.B.SubBlocks, Block.EmptySubBlock())
	}
	fmt.Println(baseBlock.Hash().Bytes())
	for i := 0; i < config.SpiConf.ShardAmount; i++ {
		//ret.blockAck[i] = make(map[crypt.Hash]int)
		ret.blockBuffer[i] = make(map[crypt.Hash]*Block.SpiralBlock)
		ret.TopBlockHash[i] = baseBlock.Hash()
		ret.blockBuffer[i][baseBlock.Hash()] = baseBlock
	}
	ret.TxPool = Tx.NewTxPool(shard.Id)
	return ret
}

func (sc *SpiChain) EnableStorage(port string) {
	sc.storage = storage.NewStorage(port, uint64(sc.Id()))

}
