package utils

import (
	"blockEmulator/config"
	"log"
	"strconv"
)

// the default method
func Addr2Shard(addr Address) int {
	last16Addr := addr[len(addr)-8:]
	num, err := strconv.ParseUint(last16Addr, 16, 64)
	if err != nil {
		log.Panic(err)
	}
	return int(num) % config.PyrConf.ShardAmount
}
