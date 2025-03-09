package consensus_shard

import (
	"blockEmulator/AutoTx"
	"blockEmulator/Comm"
	"blockEmulator/Interfaces"
	"blockEmulator/Spiral"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/consensus_shard/pbft"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"blockEmulator/storage"
	"fmt"
	"log"
	"strconv"
	"time"
)

var state = 0

func Selector() {
	timeStamp := time.Now()
	Interfaces.Communications[Comm.Register].Request()
	if idChain.RunningNode == idChain.IDC.Main() {
		state++
		Interfaces.Con[pbft.IdMod].EnablePropose()
	}
	for {
		select {
		case msg := <-launch.BCMsgPool:
			modeStr := (*message.Content)(&msg.Content).CheckSign()
			remoteId := (*message.Content)(&msg.Content).CheckSign()
			msg.RemoteInfo = remoteId
			mode, _ := strconv.Atoi(modeStr)
			Con := Interfaces.Con[mode]
			if msg.Type != Tx.SendTxs {
				storage.CommLogger.Writef("record::%v", msg.Type)
			}
			msgType := msg.Type
			if msgType == Comm.Register.ResponseType() {
				// Uninitialized nodes will run at this brunch
				Interfaces.Communications[Comm.Register].HandleResponse(msg)
				state++ // todo
				// this is a bug. But convenient.
				// Con.Communications[IdInit].Request()
				log.Printf("--->>>Decode time cost:%v", time.Since(timeStamp))
				launch.ClearMsgBuffer()
				continue
			}
			commType := Interfaces.ComTypes[msgType]
			// if Comm obj is deployed, its msgType will be record and be able to be recognized.
			if commType != "" {
				if Interfaces.IsReq(msgType) {
					Interfaces.Communications[commType].HandleRequest(msg)
				} else {
					Interfaces.Communications[commType].HandleResponse(msg)
				}
				continue
			}
			success := false
			if state > 0 {
				if msgType == message.SendTxs {
					AutoTx.Manager.HandleSendTxs(msg)
					success = true
				} else if msgType == message.TxEOF {
					if config.SpiConf.Enable {
						config.ManagerFinished = true
						fmt.Println("_________TX-FINISH_________")
						if idChain.RunningNode == Spiral.LocalShard.Main() {
							Interfaces.Communications[Interfaces.MainBegin].Request()
						}
						success = true
					} else {
						log.Panic("ENABLE BEGIN")
					}
				} else {
					success = Con.HandleMsg(msg)
				}
			}
			if state <= 0 || success == false {
				// record the msg in case that receiving pbftMsg before the successMsg arrive
				log.Printf("Selector::%v,%v to msg buffer", string(msg.Type), msg.RemoteInfo)
				(*message.Content)(&msg.Content).Sign(remoteId).Sign(modeStr)
				launch.MsgBuffer = append(launch.MsgBuffer, msg)
			}
		}
	}
}
