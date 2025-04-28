package Opts

import (
	"blockEmulator/Block"
	"blockEmulator/Interfaces"
	"blockEmulator/Proposals"
	"blockEmulator/message"
)

// EmptyOpt 实现了一个空操作，用于在区块链系统中作为占位符或空闲操作
type EmptyOpt struct {
	con   Interfaces.Consensus // 共识接口实例
	block Block.Block          // 关联的区块
}

// Reset 初始化空操作并设置其优先级
// 参数:
// - con: 共识接口实例
// 返回值:
// - message.RequestType: 返回操作类型（Empty）
func (op *EmptyOpt) Reset(con Interfaces.Consensus) message.RequestType {
	op.con = con
	con.GetProposalBuffer().SetPriority(message.Empty, Proposals.Now)
	return message.Empty
}

// Schedule 将空操作提案添加到共识系统的提案缓冲区
// 这个方法会触发共识过程为Empty类型的操作
func (op *EmptyOpt) Schedule(...*[]byte) {
	Propose(op.con, message.Empty)
	//Interfaces.Schedule(op.con,Empty, newBlock.EncodeH())
}

// Prepare 在执行操作前的准备阶段
// 对于空操作，总是返回准备就绪（true）
// 参数:
// - []*[]byte: 操作相关的变量数组
// 返回值:
// - bool: 是否准备就绪
func (op *EmptyOpt) Prepare([]*[]byte) bool {
	return true
}

// Verify 验证操作参数是否有效
// 对于空操作，总是返回验证通过（true）
// 参数:
// - [][]byte: 需要验证的参数数组
// 返回值:
// - bool: 验证是否通过
func (op *EmptyOpt) Verify([][]byte) bool {
	return true
}

// Execute 执行操作
// 空操作不需要执行任何实际操作，因此方法体为空
func (op *EmptyOpt) Execute() {
	return
}
