package launch

import (
	"blockEmulator/Spiral"
	"blockEmulator/config"
	"blockEmulator/idChain"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/pyramid"
	"blockEmulator/storage"
	"log"
	"strconv"
)

var (
	MsgBuffer  = make([]*message.Message, 0)
	LaunchPool = make(chan *message.Message, 1024)
	BCMsgPool  = make(chan *message.Message, 2048) // 区块链消息池
)

type ConInfo struct {
	ConId   int
	OptType message.RequestType
	Vars    []*[]byte
}

func LaunchPyrMsg(m *message.Message) {
	SignAndLaunch(m, config.PyrMod)
}

func LaunchIdMsg(m *message.Message) {
	SignAndLaunch(m, config.IdMod)
}

func LaunchSpiMsg(m *message.Message) {
	SignAndLaunch(m, config.SpiMod)
}

func SignAndLaunch(m *message.Message, id int) {
	(*message.Content)(&m.Content).Sign(idChain.RunningNode.StrKey()).Sign(strconv.Itoa(id))
	LaunchPool <- m
}

func TestConn(addr string) bool {
	connected := _launch_(&message.Message{
		Type:       "Ping",
		Content:    *message.NewStrContent().Sign("").Sign(""),
		RemoteInfo: addr,
	})
	return connected
}

func LaunchPad() {
	for {
		msg := <-LaunchPool
		_launchPad(msg)
	}
}

func _launchPad(msg *message.Message) {
	switch addr := msg.RemoteInfo; {
	case addr == config.IdRunningAddr:
		log.Printf("LaunchPad::%v to %v,(%v/%v).\n", msg.Type, msg.RemoteInfo, len(LaunchPool), cap(LaunchPool))
		//log.Printf("%v,%v", idChain.IDC.NodeMap, msg)
		for _, node := range idChain.IDC.NodeMap {
			if node.IsRunning() {
				MSG := &message.Message{
					Type:       msg.Type,
					Content:    msg.Content,
					RemoteInfo: node.IpAddr,
				}
				_launch_(MSG)
			}
		}
	//case strings.Contains(config.ShardPrefix):
	case addr == config.IdLegalAddr:
		log.Printf("LaunchPad::%v to %v,(%v/%v).\n", msg.Type, msg.RemoteInfo, len(LaunchPool), cap(LaunchPool))
		for _, node := range idChain.IDC.NodeMap {
			if node.IsLegal() == true {
				MSG := &message.Message{
					Type:       msg.Type,
					Content:    msg.Content,
					RemoteInfo: node.IpAddr,
				}
				_launch_(MSG)
			}
		}
	case addr == config.PyrRunningAddr:
		log.Printf("LaunchPad::%v to %v,(%v/%v).\n", msg.Type, msg.RemoteInfo, len(LaunchPool), cap(LaunchPool))
		for _, node := range pyramid.LocalShard.NodeMap {
			if node.IsRunning() {
				MSG := &message.Message{
					Type:       msg.Type,
					Content:    msg.Content,
					RemoteInfo: node.IpAddr,
				}
				_launch_(MSG)
			}
		}
	case addr == config.PyramidMainAddr:
		msg.RemoteInfo = pyramid.LocalShard.Main().IpAddr
		log.Printf("LaunchPad::%v to %v,(%v/%v).\n", msg.Type, msg.RemoteInfo, len(LaunchPool), cap(LaunchPool))
		go _launch_(msg)
	case addr == config.PyramidRelateI:
		log.Printf("LaunchPad::%v to %v,(%v/%v).\n", msg.Type, msg.RemoteInfo, len(LaunchPool), cap(LaunchPool))
		for _, index := range pyramid.LocalShard.RelatedIShard {
			if pyramid.GlobalPyrShards[index].Main() == nil {
				continue
			}
			shardAddr := pyramid.GlobalPyrShards[index].Main().IpAddr
			MSG := &message.Message{
				Type:       msg.Type,
				Content:    msg.Content,
				RemoteInfo: shardAddr,
			}
			_launch_(MSG)
		}
	case addr == config.SpiRunningAddr:
		for _, node := range Spiral.LocalShard.NodeMap {
			if node.IsLegal() == true {
				MSG := &message.Message{
					Type:       msg.Type,
					Content:    msg.Content,
					RemoteInfo: node.IpAddr,
				}
				_launch_(MSG)
			}
		}
	default:
		//log.Printf("LaunchPad::%v to %v,(%v/%v).\n", msg.Type, msg.RemoteInfo, len(LaunchPool), cap(LaunchPool))
		_launch_(msg)
	}
}

func _launch_(msg *message.Message) bool {
	tc, err := networks.Connect(msg.RemoteInfo)
	if err != nil {
		log.Println(err)
		log.Printf("Msg(%v) abandoned.", msg.Type)
		return false
	}
	if msg.Type != message.SendTxs {
		storage.CommLogger.Writef("Sent::%v to %v", msg.Type, msg.RemoteInfo)
		//log.Printf("Sent::%v to %v", msg.Type, msg.RemoteInfo)
	}
	err = tc.SendMsg(msg.Type, msg.Content)
	if err != nil {
		return true
	}
	return true
}

func ClearMsgBuffer() {
	for _, msg := range MsgBuffer {
		BCMsgPool <- msg
	}
	MsgBuffer = make([]*message.Message, 0)
}
