package test

import (
	"blockEmulator/config"
	"blockEmulator/message"
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func NetTester() {
	// 设置服务器监听的端口
	listener, err := net.Listen("tcp", ":19999")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("Listening on :19999")

	for {
		// 等待客户端的连接
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		fmt.Println("Received message:")

		// 创建一个新的goroutine来处理接收到的信息
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	// 使用bufio.NewScanner读取数据
	//scanner := bufio.NewScanner(conn)
	//for scanner.Scan() {
	//	fmt.Println(scanner.Text())
	//}

	//if err := scanner.Err(); err != nil {
	//	fmt.Println("Error reading:", err.Error())
	//}
	clientReader := bufio.NewReader(conn)
	for {
		currMsg, err := clientReader.ReadBytes(config.MsgSplitter)
		cnt := 0
		switch err {
		case nil:
			currMsg = currMsg[:len(currMsg)-1] // remove the splitter '|'
			//fmt.Printf("COUNTER%v:\n%v\n", cnt, currMsg)
			//fmt.Println("--------")
			//fmt.Printf("%s\n", currMsg)
			tp, c := message.SplitMessage(currMsg)
			log.Printf("%v\n%v\n", tp, c)
			cnt++
		case io.EOF:
			return
		}
	}

}
