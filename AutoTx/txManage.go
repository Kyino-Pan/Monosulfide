package AutoTx

import (
	"blockEmulator/Monosulfide"
	"blockEmulator/Relay"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"blockEmulator/pyramid"
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"os"
	"time"
)

var Manager *TxManager

type TxManager struct {
	totalDataAmount int
	batchDataAmount int
	dataCnt         int
	localPool       *Tx.Pool
}

func NewTxManager(pool *Tx.Pool) *TxManager {
	ret := &TxManager{
		totalDataAmount: config.TotalDataSize,
		batchDataAmount: config.BatchSize,
		dataCnt:         0,
		localPool:       pool,
	}
	return ret
}

// read transactions, the GetNonce of the transactions is - batchDataAmount
func (tm *TxManager) MsgSendingControl() {
	txFile, err := os.Open(config.FileInput)
	if err != nil {
		txFile, err = os.Open(config.DockerInput)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer txFile.Close()
	reader := csv.NewReader(txFile)
	txList := make([]*Tx.Transaction, 0) // save the InternalTxs in this epoch (round)
	log.Printf("MsgControl")
	for {
		data, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panic(err)
		}
		if tx, ok := data2tx(data, uint64(tm.dataCnt)); ok {
			txList = append(txList, tx)
			tm.dataCnt++
		}
		if (len(txList) == tm.batchDataAmount) || (tm.dataCnt == tm.totalDataAmount) {
			tm.sending(txList)
			// reset the variants about tx sending
			txList = make([]*Tx.Transaction, 0)
			if tm.dataCnt == tm.totalDataAmount {
				break
			}
		}
	}
	tm.SendEOF()
}

// transform, data to transaction
// check whether it is a legal InternalTxs message. if so, read InternalTxs and put it into the txList
func data2tx(data []string, nonce uint64) (*Tx.Transaction, bool) {
	if data[6] == "0" && data[7] == "0" && len(data[3]) > 16 && len(data[4]) > 16 && data[3] != data[4] {
		value, ok := new(big.Int).SetString(data[8], 10)
		if !ok {
			log.Panic("new int failed\n")
		}
		sender := data[3][2:]
		recipient := data[4][2:]

		tx := Tx.NewTransaction(sender, recipient, value, nonce, Tx.PryTx)
		return tx, true
	}
	return &Tx.Transaction{}, false
}

func (tm *TxManager) sending(txList []*Tx.Transaction) {
	// the InternalTxs will be sent
	txBuffer := make(map[int][]*Tx.Transaction)

	// 计算每批次应发送的交易数量
	// 基于每秒注入速率和注入间隔计算
	intervalInSeconds := float64(config.TxInjectInterval) / float64(time.Second)
	batchSize := int(float64(config.InjectSpeed) * intervalInSeconds)
	if batchSize < 1 {
		batchSize = 1 // 确保至少发送一个交易
	}

	for txIndex := 0; txIndex <= len(txList); {
		if txIndex == len(txList) {
			break
		}
		tx := txList[txIndex]
		shardIndexes := tx.RelatedShards()
		for _, shard := range shardIndexes {
			tx.Time = time.Now()
			txBuffer[shard] = append(txBuffer[shard], tx)
		}
		// both the sender and receiver's shard will receive the tx.
		txIndex++
		if txIndex%batchSize == 0 || txIndex == len(txList) {
			// send to shard
			tm.SendTxs(txBuffer)
			txBuffer = make(map[int][]*Tx.Transaction)
			time.Sleep(config.TxInjectInterval)
		}
	}
}

func (tm *TxManager) SendTxs(txsInShard map[int][]*Tx.Transaction) {
	for shardId := 0; shardId < (config.ShardAmount); shardId++ {
		txs := Txs2Bytes(txsInShard[shardId])
		content := message.NewByteContent(&txs)
		if config.PyrConf.Enable {
			for _, node := range pyramid.GlobalShards[shardId].NodeMap {
				remoteAddr := node.IpAddr
				launch.LaunchPyrMsg(&message.Message{
					Type:       message.SendTxs,
					Content:    *content,
					RemoteInfo: remoteAddr,
				})
			}
		} else if config.FideConf.Enable {
			for _, node := range Monosulfide.GlobalShards[shardId].NodeMap {
				remoteAddr := node.IpAddr
				launch.LaunchFideMsg(&message.Message{
					Type:       message.SendTxs,
					Content:    *content,
					RemoteInfo: remoteAddr,
				})
			}
		} else if config.RelayConf.Enable {
			for _, node := range Relay.GlobalShards[shardId].NodeMap {
				remoteAddr := node.IpAddr
				launch.LaunchRelayMsg(&message.Message{
					Type:       message.SendTxs,
					Content:    *content,
					RemoteInfo: remoteAddr,
				})
			}
		}
	}
}

func (tm *TxManager) HandleSendTxs(msg *message.Message) {
	cnt := 0
	if config.PyrConf.Enable {
		for pyramid.GlobalShards == nil {
			time.Sleep(100 * time.Millisecond)
			cnt++
			if cnt >= 128 {
				log.Panic("HandleSendTxs::no pyr shard")
			}
		}
	}
	if config.FideConf.Enable {
		// safety check.
		for Monosulfide.GlobalShards == nil {
			time.Sleep(100 * time.Millisecond)
			cnt++
			if cnt >= 32 {
				log.Panic("HandleSendTxs::no Fide shard")
			}
		}
	}
	contents := msg.GetContents()
	txs := ParseTxs(contents[0])
	for _, tx := range *txs {
		tm.localPool.Append(tx)
	}
}

func (tm *TxManager) SendEOF() {
	log.Println("SendEOF")
	if config.PyrConf.Enable || config.FideConf.Enable {
		for _, node := range idChain.IDC.NodeMap {
			launch.LaunchIdMsg(&message.Message{
				Type:       message.TxEOF,
				Content:    nil,
				RemoteInfo: node.IpAddr,
			})
		}
	}
}

func ParseTxs(txsByte []byte) *[]*Tx.Transaction {
	txs := new([]*Tx.Transaction)
	err := json.Unmarshal(txsByte, txs)
	if err != nil {
		log.Panic(err)
		return nil
	}
	return txs
}

func Txs2Bytes(txs []*Tx.Transaction) []byte {
	ret, err := json.Marshal(txs)
	if err != nil {
		log.Panic(err)
	}
	return ret
}
