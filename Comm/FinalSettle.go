package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/Monosulfide"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"blockEmulator/storage"
	"encoding/json"
	"log"
	"strconv"
)

// not using now
type FinishCom struct {
	con         Interfaces.Consensus
	finishCount int
	writer      *storage.CsvWriter
}

func (com *FinishCom) Request(...*[]byte) bool {
	if config.FideConf.Enable {
		chain := Monosulfide.LocalShard.Chain
		content :=
			message.NewByteContent(chain.EncodedExp()).
				AppendStrContent(strconv.Itoa(Monosulfide.LocalShard.Id))
		com.con.SendMsg(&message.Message{
			Type:       Interfaces.Finish.RequestType(),
			Content:    *content,
			RemoteInfo: config.MonitorAddr,
		})
		return true
	}
	return false
}

func (com *FinishCom) HandleRequest(msg *message.Message) bool {
	if config.FideConf.Enable {
		exp := new(Monosulfide.Record)
		contents := msg.GetContents()
		err := json.Unmarshal(contents[0], exp)
		if err != nil {
			log.Println(err.Error())
		}
		//sid, err := strconv.Atoi(string(contents[1]))
		if err != nil {
			log.Println(err.Error())
		}
		com.finishCount += 1
	}
	return true
}

func (com *FinishCom) Response(...*[]byte) bool {
	com.con.SendMsg(&message.Message{
		Type:       Interfaces.Finish.ResponseType(),
		Content:    *message.NewStrContent(""),
		RemoteInfo: idChain.IDC.Main().IpAddr,
	})
	return true
}

func (com *FinishCom) HandleResponse(msg *message.Message) {
}

func (com *FinishCom) Reset() Interfaces.CommType {
	com.con = Interfaces.Con[config.IdMod]
	com.finishCount = 0
	return Interfaces.Finish
}
