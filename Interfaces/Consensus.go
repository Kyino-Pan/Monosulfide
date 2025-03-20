package Interfaces

import (
	"blockEmulator/Proposals"
	"blockEmulator/message"
	"time"
)

type Consensus interface {
	SendMsg(*message.Message)
	Propose(message.RequestType, ...*[]byte)
	GetProposalBuffer() *Proposals.ProposalBuffer
	GetDomain() Domain
	SetDomain(Domain)
	Tic()
	Tok() time.Duration // should return duration since the latest Tic()
	EnablePropose()
	DisablePropose()
	HandleMsg(*message.Message) bool
	Id() int
}

var Con = make([]Consensus, 4)
