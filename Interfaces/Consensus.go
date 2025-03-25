package Interfaces

import (
	"blockEmulator/Proposals"
	"blockEmulator/message"
	"time"
)

type Consensus interface {
	SendMsg(*message.Message)
	Propose()
	GetProposalBuffer() *Proposals.ProposalBuffer
	GetDomain() Domain
	SetDomain(Domain)
	Tic()
	Tok() time.Duration // should return duration since the latest Tic()
	Enable()
	Disable()
	HandleMsg(*message.Message) bool
	Id() int
}

var Con = make([]Consensus, 4)
