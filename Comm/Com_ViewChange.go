package Comm

import (
	"blockEmulator/Interfaces"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"log"
	"sync"
)

const (
	standBy    = "standBy"
	collecting = "collecting"
)

type ViewChangeCom struct {
	con            Interfaces.Consensus
	count          map[uint64]map[string]bool
	handleBlocker  sync.Mutex
	cond           *sync.Cond
	threshold      int
	state          string
	responseBuffer map[uint64]string // map[viewId] msgId
}

func (com *ViewChangeCom) Type() Interfaces.CommType {
	return Interfaces.ViewChange
}

func (com *ViewChangeCom) Request(...*[]byte) bool {
	com.con.SendMsg(&message.Message{
		Type:       Interfaces.ViewChange.RequestType(),
		Content:    *message.NewByteContent(crypt.UintToBytes(idChain.IDC.GetViewId() + 1)),
		RemoteInfo: com.con.GetDomain().Addr(),
	})
	return true
}

func (com *ViewChangeCom) HandleRequest(msg *message.Message) bool {
	com.handleBlocker.Lock()
	defer com.handleBlocker.Unlock()
	contents := msg.GetContents()
	vid := crypt.BytesToUint(contents[0])
	if vid <= idChain.IDC.GetViewId() {
		return false
	}
	if idChain.IDC.NodeMap[msg.RemoteInfo] != nil {
		if com.count[vid] == nil {
			com.count[vid] = make(map[string]bool)
		}
		com.count[vid][msg.RemoteInfo] = true
		log.Printf("Get view change%v request(%v/%v)", vid, len(com.count[vid]), int(com.con.GetDomain().Threshold()))
		if len(com.count[vid]) >= int(com.con.GetDomain().Threshold()) && com.state == standBy {
			idChain.IDC.Main().Activating = false
			idChain.IDC.Main().Sleeping = false
			idChain.IDC.SelectMainNode()
			log.Printf("New main node is %v", idChain.IDC.Main().IpAddr)
			if idChain.IsIdMainNode() {
				com.con.SendMsg(&message.Message{
					Type:       Interfaces.ViewChange.ResponseType(),
					Content:    *message.NewByteContent(crypt.UintToBytes(vid)),
					RemoteInfo: com.con.GetDomain().Addr(),
				})
				com.change(vid)
				com.con.EnablePropose()
			} else {
				if com.responseBuffer[vid] == idChain.IDC.Main().StrKey() {
					com.change(vid)
					return true // Already received response before.
				}
				com.state = collecting
			}
		}
	}
	return true
}

func (com *ViewChangeCom) Response(...*[]byte) bool {
	return true
}

func (com *ViewChangeCom) HandleResponse(msg *message.Message) {
	contents := msg.GetContents()
	vid := crypt.BytesToUint(contents[0])
	remoteNode := idChain.IDC.NodeMap[msg.RemoteInfo]
	if com.state == standBy {
		com.responseBuffer[vid] = msg.RemoteInfo
	} else if com.state == collecting {
		if remoteNode == idChain.IDC.Main() {
			com.change(vid)
			return
		} else {
			log.Printf("WARNING::CVC2")
		}
	}
	return
}

func (com *ViewChangeCom) change(vid uint64) {
	log.Println("View change end.")
	idChain.IDC.SetView(vid)
	com.con.Tic()
}

func (com *ViewChangeCom) Reset() Interfaces.CommType {
	com.con = Interfaces.Con[config.IdMod]
	com.count = make(map[uint64]map[string]bool)
	com.state = standBy
	com.responseBuffer = make(map[uint64]string)
	return Interfaces.ViewChange
}
