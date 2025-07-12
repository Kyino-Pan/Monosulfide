package consensus_shard

import (
	"blockEmulator/AutoTx"
	"blockEmulator/Interfaces"
	"blockEmulator/Opts"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/consensus_shard/pbft"
	"blockEmulator/consensus_shard/pow"
	"blockEmulator/idChain"
	"blockEmulator/launch"
	"blockEmulator/message"
	"blockEmulator/storage"
	"fmt"
	"log"
	"strconv"
)

var state = 0

func Selector() {
	//timeStamp := time.Now()
	Interfaces.Communications[Interfaces.Register].Request()
	begin := false
	if idChain.RunningNode == idChain.IDC.Main() {
		state++
		Interfaces.Con[pbft.IdMod].Enable()
	}
	for {
		select {
		case msg := <-launch.BCMsgPool:
			modeStr := (*message.Content)(&msg.Content).CheckSign()
			remoteId := (*message.Content)(&msg.Content).CheckSign()

			msg.RemoteInfo = remoteId // replace field with message sender's pubKey

			mode, _ := strconv.Atoi(modeStr)
			Con := Interfaces.Con[mode]
			if msg.Type != Tx.SendTxs {
				storage.CommLogger.Writef("record::%v", msg.Type)
			}

			msgType := msg.Type
			if msgType == Interfaces.Register.ResponseType() {
				// Uninitialized nodes will run at this brunch
				Interfaces.Communications[Interfaces.Register].HandleResponse(msg)
				state++ // todo
				// this is a bug. But convenient.
				// Con.Communications[IdInit].Request()
				//log.Printf("--->>>Decode time cost:%v", time.Since(timeStamp))
				launch.ClearMsgBuffer()
				continue
			}

			commType := Interfaces.ComTypes[msgType]
			// if Comm obj is deployed, its msgType will be record and be able to be recognized here.
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
					if begin == false {
						begin = true
						Opts.CrossBegin()
					}
				} else if msgType == message.TxEOF {
					config.ManagerFinished = true
					fmt.Println("_________TX-FINISH_________")
					Interfaces.LocalShard.GetTxPool().Print()
					success = true
				} else {
					success = Con.HandleMsg(msg)
				}
			}
			if state <= 0 || success == false {
				// record the msg in case that receiving pbftMsg before the successMsg arrive
				log.Printf("Selector%v::%v,%v to msg buffer", state, string(msg.Type), msg.RemoteInfo)
				(*message.Content)(&msg.Content).Sign(remoteId).Sign(modeStr)
				launch.MsgBuffer = append(launch.MsgBuffer, msg)
			}
		}
	}
}

func Init() {
	idChain.Init(launch.Listener.GetListenPort())
	if config.IdConfig.UsingPoW() {
		Interfaces.Con[config.IdMod] = pow.NewIdChainCon()
	} else if config.IdConfig.UsingPbft() {
		Interfaces.Con[config.IdMod] = pbft.NewIdChainCon()
	}
	Interfaces.Con[config.PyrMod] = pbft.NewPyramidCon()
	//Interfaces.Con[config.FideMod] = pbft.NewMonosulfideCon()
	if config.FideConf.Enable {
		Interfaces.Con[config.FideMod] = pow.NewFideCon()
	}
	if config.MonoxideConf.Enable {
		Interfaces.Con[config.RelayMod] = pow.NewRelayCon()
	}
}
