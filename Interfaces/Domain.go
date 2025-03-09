package Interfaces

import (
	"blockEmulator/Block"
	"blockEmulator/idChain"
)

type Domain interface {
	Main() *idChain.Node
	SelectMainNode() *idChain.Node
	Threshold() uint64
	ProcessingBlock() Block.Block
	SetProcessingBlock(block Block.Block)
	GetViewId() uint64
	SetViewId(uint64)
	Addr() string
}
