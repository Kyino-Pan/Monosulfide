package main

import (
	"blockEmulator/config"
	"blockEmulator/engine"
	"blockEmulator/storage"
	"fmt"
	"github.com/spf13/pflag"
	"log"
	"os"
	"strconv"
)

var (
	shardNum  int
	nodeNum   int
	packaging bool
	isGen     int
	idMode    int
)

func main() {
	pflag.IntVarP(&shardNum, "shardNum", "S", 0, "indicate that how many shards are deployed")
	pflag.IntVarP(&nodeNum, "nodeNum", "N", 0, "indicate how many nodes of each idChain are deployed")
	pflag.BoolVarP(&packaging, "client", "p", false, "packaging csv files.")
	pflag.IntVarP(&isGen, "gen", "g", 0, "generation bat")
	pflag.IntVarP(&idMode, "idMode", "i", -1, "select consensus protocol using in identity chain, 0 = pow ")
	pflag.Parse()
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	if shardNum != 0 {
		config.ShardAmount = shardNum
		log.Printf("shardNum = %d", config.ShardAmount)
	}
	config.FideConf = config.InitFideConfig()
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
	if config.FideConf.Enable == true {
		cnt++
	}
	if config.RelayConf.Enable == true {
		cnt++
	}
	if cnt > 1 {
		log.Panic("Enable multiple sharding consensus. ")
	}

	engine.Run()
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
