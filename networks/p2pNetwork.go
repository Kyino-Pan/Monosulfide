// something about broadcast

package networks

import (
	"blockEmulator/config"
	"blockEmulator/message"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

var connMapLock sync.Mutex
var connectionPool = make(map[string]*TcpConn)

type TcpConn struct {
	conn        net.Conn
	remoteAddr  string
	localAddr   string
	isConnected bool
}

func NewTcpConn(conn net.Conn) *TcpConn {
	tcpConn := TcpConn{
		conn:        conn,
		remoteAddr:  conn.RemoteAddr().String(),
		localAddr:   conn.LocalAddr().String(),
		isConnected: true,
	}
	return &tcpConn
}

func (tc *TcpConn) Read(bf []byte) (int, error) {
	return tc.conn.Read(bf)
}

func Connect(addr string) (*TcpConn, error) {
	connMapLock.Lock()
	defer connMapLock.Unlock()
	tc := new(TcpConn)
	reconnInterval := 100
	if connectionPool[addr] == nil { // no existing connection
		for {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				time.Sleep(time.Duration(reconnInterval) * time.Millisecond)
				reconnInterval *= 2
				if reconnInterval > 200*4 { // waiting too long
					return nil, err
				}
				log.Printf("error occurred while connecting to %v:\n%v", addr, err)
				continue // connect again
			}
			connectionPool[addr] = &TcpConn{
				conn:        conn,
				remoteAddr:  conn.RemoteAddr().(*net.TCPAddr).String(),
				localAddr:   conn.LocalAddr().(*net.TCPAddr).String(),
				isConnected: true,
			}
			break
		}
		fmt.Printf("Conn amount = %v\n", len(connectionPool))
	}
	tc = connectionPool[addr]
	return tc, nil
}

func (tc *TcpConn) GetRemoteAddr() string {
	return tc.remoteAddr
}

func (tc *TcpConn) Close() error {
	connMapLock.Lock()
	defer connMapLock.Unlock()
	if !tc.isConnected {
		return nil
	}
	err := tc.conn.Close()
	if err != nil {
		return err
	}
	tc.isConnected = false
	connectionPool[tc.remoteAddr] = nil
	return nil
}

// todo
func (tc *TcpConn) IsClose() bool {
	return !tc.isConnected
}

func (tc *TcpConn) SendMsg(msgType message.MessageType, content []byte) error {
	msg := EncodeMsg(msgType, content)
	if msgType == message.SendTxs {
		k := msg[4+config.PrefixMsgTypeLen:]
		m := &message.Message{
			Type:       msgType,
			Content:    k,
			RemoteInfo: config.Localhost,
		}
		(*message.Content)(&m.Content).CheckSign()
		remoteId := (*message.Content)(&m.Content).CheckSign()
		m.RemoteInfo = remoteId
		m.GetContents()
	}
	_, err := tc.conn.Write(msg)
	if err != nil {
		return err
	}
	return nil
}

func EncodeMsg(msgType message.MessageType, content []byte) []byte {
	length := uint32(len(content))
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length)
	//fmt.Printf("EncodeH: %v\n", length)

	msg := append([]byte(msgType), make([]byte, config.PrefixMsgTypeLen-len(msgType))...)
	msg = append(header, msg...)
	msg = append(msg, content...)

	return msg
}
