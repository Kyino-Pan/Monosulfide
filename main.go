package main

import (
	"blockEmulator/build"
	"blockEmulator/config"
	"blockEmulator/storage"
	"fmt"
	"github.com/spf13/pflag"
	"log"
	"os"
	"strconv"
)

var (
	shardNum int
	nodeNum  int
	//shardID  int
	//nodeID   int
	modID     int
	packaging bool
	isGen     int
	//accountPubKey string
	seq string
)

func main() {
	pflag.IntVarP(&shardNum, "shardNum", "S", 0, "indicate that how many shards are deployed")
	pflag.IntVarP(&nodeNum, "nodeNum", "N", 0, "indicate how many nodes of each idChain are deployed")
	//pflag.IntVarP(&shardID, "shardID", "s", 0, "id of the idChain to which this node belongs, for example, 0")
	//pflag.IntVarP(&nodeID, "nodeID", "n", 0, "id of this node, for example, 0")
	pflag.IntVarP(&modID, "modID", "m", 4, "choice Committee Method,for example, 0, [CLPA_Broker,CLPA,Broker,Relay,Federal] ")
	pflag.BoolVarP(&packaging, "client", "p", false, "packaging csv files.")
	//pflag.BoolVarP(&config.Debugging, "debug", "d", false, "debug enabled?")
	pflag.IntVarP(&isGen, "gen", "g", 0, "generation bat")
	//pflag.StringVarP(&accountPubKey, "accountPubKey", "k", "", "account public key")
	//pflag.StringVarP(&seq, "num", "p", "", "sequence of this process")
	pflag.Parse()

	log.SetFlags(log.Ltime | log.Lmicroseconds)

	//if isGen {
	//	build.GenerateBatFile(nodeNum, shardNum, modID)
	//	return
	//}

	//if isClient {
	//	build.BuildSupervisor(uint64(nodeNum), uint64(shardNum), uint64(modID))
	//} else {
	//	build.BuildNewPbftNode(uint64(nodeID), uint64(nodeNum), uint64(shardID), uint64(shardNum), uint64(modID))
	//}
	//{
	//	content := message.NewStrContent("\t", "\\\t", "\t\\\ta").Sign("kyino")
	//	log.Printf(content.CheckSign())
	//	parsed := content.ParseContent()
	//	log.Printf("%v", parsed)
	//	return
	//}
	if seq == "10" {
		err := os.Remove("./record")
		if err != nil {
			return
		}
		return
	}
	if shardNum != 0 {
		config.SpiralShardAmount = shardNum
		log.Printf("shardNum = %d", config.SpiralShardAmount)
	}
	config.SpiConf = config.InitSpiralConfig()
	if nodeNum != 0 {

	}
	if packaging {
		storage.MergeCsv()
		return
	}
	if isGen != 0 {
		GenTestSh(isGen, shardNum)
		return
	}
	cnt := 0
	if config.PyrConf.Enable == true {
		cnt++
	}
	if config.SpiConf.Enable == true {
		cnt++
	}
	if config.RelayConf.Enable == true {
		cnt++
	}
	if cnt > 1 {
		log.Panic("Enable multiple sharding consensus. ")
	}
	build.Run(uint64(modID))
}

func GenTestSh(numTasks int, shardNum int) {
	// 创建或打开输出的 shell 脚本文件
	if numTasks < shardNum {
		log.Panic("Not enough sharding tasks")
	}
	file, err := os.Create("test.sh")
	if err != nil {
		fmt.Println("Error creating script file:", err)
		os.Exit(1)
	}
	defer file.Close()

	// 写入 shebang 行
	fmt.Fprintln(file, "#!/bin/zsh")

	// 生成指定数量的 go run 命令
	for i := 1; i <= numTasks; i++ {
		cmd := fmt.Sprintf("go run main.go -S %s > ./output/out%d.txt 2>&1 &", strconv.Itoa(shardNum), i)
		fmt.Fprintln(file, cmd)
	}
	// 写入 wait 命令
	fmt.Fprintln(file, "wait")

	// 设置文件权限以确保脚本可执行
	if err := file.Chmod(0755); err != nil {
		fmt.Println("Error setting script permissions:", err)
		os.Exit(1)
	}

	fmt.Println("Script generated successfully.")
	storage.MergeCsv()
}
