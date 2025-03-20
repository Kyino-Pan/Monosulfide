package launch

import (
	"blockEmulator/Monosulfide"
	"blockEmulator/config"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/storage"
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	Listener *NetListener = nil
	// 所有与网络相关的接口被封装在NetListener中
)

type NetListener struct {
	tcpListener net.Listener
	tcpLock     sync.Mutex
}

func NewNetListener(netLn net.Listener) *NetListener {
	return &NetListener{
		tcpListener: netLn,
	}
}

func (nl *NetListener) GetListenPort() string {
	return strings.Split(nl.tcpListener.Addr().String(), ":")[1]
}

func (nl *NetListener) GetLocalAddr() string {
	return nl.tcpListener.Addr().String()
}

func (nl *NetListener) lock() {
	nl.tcpLock.Lock()
}

func (nl *NetListener) unlock() {
	nl.tcpLock.Unlock()
}

func TcpListen() {
	port, _ := strconv.Atoi(config.ListenPort)
	config.InitNetDelay()
	cnt := 0
	for {
		ln, err := net.Listen("tcp", config.Localhost+":"+strconv.Itoa(port))
		if err != nil {
			// 生成一个0到20000之间的随机数
			//randomNumber := rand.IntN(1000) // IntN(n) 生成[0, n)范围内的随机整数
			//port += randomNumber
			port += 10
			cnt++
			if cnt > 1000 {
				log.Panic(err)
			}
			continue
		} else {
			Listener = NewNetListener(ln)
			if cnt == 0 {
				storage.MergeCsv() //merge previous work
			}
			storage.Init(cnt)
			//go storage.CommLogger.Run()
			break
		}
	}
}

func (nl *NetListener) Hearing() {
	storage.CommLogger.Writef("---TCP Listening on: %v", Listener.GetListenPort())
	log.Printf("---TCP Listening on: %v", Listener.GetListenPort())
	go LaunchPad()
	for {
		conn, err := Listener.tcpListener.Accept()
		if err != nil {
			return
		}
		tcpCoon := networks.NewTcpConn(conn)
		go handleRequest(tcpCoon)
	}
}

// deconstruct TCP package and sent info to classifier
func handleRequest(conn *networks.TcpConn) {
	defer func(conn *networks.TcpConn) {
		err := conn.Close()
		if err != nil {
			log.Println(err)
		}
	}(conn)
	senderAddr := conn.GetRemoteAddr()
	for {
		header := make([]byte, 4)
		_, err := io.ReadFull(conn, header)
		if err != nil {
			if err == io.EOF {
				log.Println("handler::client closed")
				if config.FideConf.Enable {
					Monosulfide.LocalShard.Chain.Save()
					config.STOPPER <- true
					//todo
					// this is not safe, just a temporary method.
				}
				return
			}
			log.Panicf("failed to read message length: %v", err)
		}
		length := binary.BigEndian.Uint32(header)
		//fmt.Printf("Decode: %v\n", length)
		// Step 2: 读取消息类型字段（PrefixMsgTypeLen 字节）
		msgTypeBuf := make([]byte, config.PrefixMsgTypeLen)
		_, err = io.ReadFull(conn, msgTypeBuf)
		if err != nil {
			log.Panicf("failed to read message type: %v", err)
		}
		// 去掉消息类型的填充部分
		msgType := message.MessageType(bytes.TrimRight(msgTypeBuf, "\x00"))

		// Step 3: 读取消息内容（长度由前面解析的 length 决定）
		content := make([]byte, length)
		_, err = io.ReadFull(conn, content)
		if err != nil {
			if err == io.EOF {
				log.Printf("handler::client closed")
				if config.FideConf.Enable {
					Monosulfide.LocalShard.Chain.Save()
					config.STOPPER <- true
				}
				return
			}
			log.Panicf("failed to read message content: %v", err)
		}
		go _delaySending(&message.Message{
			Type:       msgType,
			Content:    content,
			RemoteInfo: senderAddr,
		})
	}
}

func _delaySending(m *message.Message) {
	time.Sleep(config.NDelay)
	BCMsgPool <- m
}
