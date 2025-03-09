package Proposals

import "blockEmulator/message"

type Proposal struct {
	ReqType message.RequestType
	Vars    []*[]byte
}

func (pro *Proposal) Get() (message.RequestType, []*[]byte) {
	return pro.ReqType, pro.Vars
}
