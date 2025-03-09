// definition of node and idChain

package idChain

import (
	"blockEmulator/Block"
	"blockEmulator/Tx"
	"blockEmulator/crypt"
	"crypto/rsa"
	"fmt"
	"log"
	"strings"
)

type Node struct {
	NodeId     *rsa.PublicKey
	ShardID    uint64
	IpAddr     string
	Activating bool
	Sleeping   bool
	Silence    uint64
}

var RunningNode *Node
var PriKey *rsa.PrivateKey = nil

func (n *Node) PrintNode() {
	v := []interface{}{
		n.NodeId,
		n.ShardID,
		n.IpAddr,
		n.Activating,
		n.Sleeping,
	}
	fmt.Printf("%v\n", v)
}

func (n *Node) IsRunning() bool {
	return !n.Sleeping && n.Activating
} // running

func (n *Node) IsWaiting() bool {
	return n.Sleeping && !n.Activating
} // registered nodes

func (n *Node) IsPreparing() bool {
	return n.Sleeping && n.Activating
} // during newEpoch nodes

func (n *Node) IsLegal() bool {
	return n.Sleeping || n.Activating
} // registered and running nodes

func (n *Node) State() uint {
	if n.IsRunning() {
		return 0
	}
	return 1
}

func (n *Node) Port() string {
	return strings.Split(n.IpAddr, ":")[1]
}

func (n *Node) StrKey() string {
	return string(crypt.EncodePublicKey(n.NodeId))
}

func IsIdMainNode() bool {
	if IDC == nil {
		return false
	}
	IDC.Lock()
	defer IDC.Unlock()
	if RunningNode == nil || IDC == nil {
		log.Println("Pbft::WARNING::Unsafe calling IsIdMainNode")
		return false
	}
	return IDC.Main() == RunningNode
}

func InitNode(isMain bool) {
	if isMain {
		b := new(Block.StdBlock).Decode(nil)
		ob, _ := b.(*Block.StdBlock)
		ob.B.Transactions = append(make([]*Tx.Transaction, 0),
			Tx.GenerateRegisterTx(
				RunningNode.StrKey(),
				RunningNode.IpAddr,
				IDC.Nonce()))
		IDC.Append(ob)
		RunningNode = IDC.NodeMap[RunningNode.StrKey()]
	} else {
		RunningNode.Sleeping = true
	}
	IDC.NodeMap[RunningNode.StrKey()] = RunningNode
	// add the origin block of the idChain.
	IDC.ChainInfo()
	return
}
