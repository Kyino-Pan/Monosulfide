package Interfaces

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/config"
	"blockEmulator/crypt"
	"blockEmulator/idChain"
)

var LocalShard Domain
var GlobalShards []Domain

type Domain interface {
	Reset(port int, i int)
	Id() int
	SetMain(node *idChain.Node)
	Main() *idChain.Node
	SelectMain() *idChain.Node

	Threshold() uint64
	ProcessingBlock() Block.Block
	SetProcessingBlock(block Block.Block)

	GetMap() map[string]*idChain.Node
	SetMap(mp map[string]*idChain.Node)

	BroadAddr() string
	GetTxPool() *Tx.Pool

	Append(block Block.Block)
}

func SelectMain(d Domain) *idChain.Node {
	randNum := idChain.IDC.GetRand()
	newIdMainNodeID := crypt.PubKey2Str(idChain.SelectRandomKey(d.GetMap(), randNum))
	if config.EnableSpy == true && d.Id() == config.SpyAtShard && config.SpyIsMainNode {
		for _, node := range d.GetMap() {
			if node.Port() == config.ListenPort {
				d.SetMain(node)
				break
			}
		}
	} else {
		d.SetMain(d.GetMap()[newIdMainNodeID])
	}
	return d.Main()
}
