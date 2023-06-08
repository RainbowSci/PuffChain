package main

import (
	"fmt"
	"log"
	"math"
	"time"
)

const (
	ROLE_PROPOSER           Role        = 1
	ROLE_VALIDATOR          Role        = 2
	ROLE_PACKER             Role        = 3
	ROLE_SIMPLE             Role        = 4
	STAGE_PRE_PREPARE       Stage       = 1
	STAGE_PREPARE           Stage       = 2
	STAGE_COMMIT            Stage       = 3
	STAGE_REPLY             Stage       = 4
	STAGE_WAIT_PRE_PREPARE  Stage       = 5
	STAGE_PACK              Stage       = 6
	STAGE_WAIT_REPLY        Stage       = 7
	PBFTMSGTYPE_PRE_PREPARE PBFTMsgType = 1
	PBFTMSGTYPE_PREPARE     PBFTMsgType = 2
	PBFTMSGTYPE_COMMIT      PBFTMsgType = 3
	PBFTMSGTYPE_REPLY       PBFTMsgType = 4
)

type (
	Role        int
	Stage       int
	PBFTMsgType int

	// PBFTMsg could be: 1) a control block 2) a prepare vote 3) a commit vote 4) a reply (control block)
	PBFTMsg struct {
		SourceID HashValue

		RoundNum int64

		PBFTMsgType PBFTMsgType

		ControlBlockHash HashValue

		// could be control block or vote
		PBFTMsgContent interface{}
	}

	Vote struct {
		VoteID     HashValue
		VoteResult bool
	}

	// PBFTRound stores the information of a PBFT round FOR THE NODE.
	PBFTRound struct {
		// RoundNum indicates the round number.
		RoundNum int64

		// Role indicates the role of the node in a given round.
		// proposer: proposes a controlBlock in this round.
		// validator: validate in this round.
		// packer: packer a dataBlock in this round.
		// simple: just wait for the consensus controlBlock.
		RoundRole Role

		// Stage indicates the stage this pbft round is in.
		// for the proposer: prePrepare, prepare, commit, reply
		// for the validator: waitPrePrepare, prepare, commit, reply
		// for the packer: pack, waitReply, reply
		// for the simple: waitReply, reply
		RoundStage Stage

		// PrepareVoteCount indicates the prepare vote count of the stage.
		PrepareVoteCount int64

		// CommitVoteCount indicates the commit vote count of the stage.
		CommitVoteCount int64

		// ReplyVoteCount indicates the reply vote count of the stage.
		ReplyVoteCount int64

		// ControlBlockHash indicates the consensus main body control block.
		ControlBlock ControlBlock

		// Timer is used for measuring the time of this round.
		Timer *time.Timer

		// RoundEnd is the channel to indicate the end of a round.
		RoundEnd chan bool
	}
)

func (node *Node) NewPBFTRound(roundNum int64, roundRole Role, roundStage Stage) PBFTRound {
	return PBFTRound{
		RoundNum:         roundNum,
		RoundRole:        roundRole,
		RoundStage:       roundStage,
		PrepareVoteCount: 0,
		CommitVoteCount:  0,
		ReplyVoteCount:   0,
		ControlBlock:     ControlBlock{},
		Timer:            time.NewTimer(ROUND_TIME),
		RoundEnd:         make(chan bool),
	}
}

func (node *Node) PrePrepare() {
	if (node.PBFTRound.RoundRole != ROLE_PROPOSER) || (node.PBFTRound.RoundStage != STAGE_PRE_PREPARE) {
		log.Panic("PANIC 1 in PrePrepare()")
	}

	previousHash := Hash(node.Ledger[len(node.Ledger)-1])
	rawControlBlock := node.NewRawControlBlock(node.PBFTRound.RoundNum, previousHash)

	rawControlBlock.Seed = node.VRFSeed

	// TODO: fill the control block and empty the data block queue, should be a function deliver the POINTER
	lineNum := int(math.Min(300, float64(len(node.DataBlockHashQueue))))
	for i := 0; i < lineNum; i++ {
		rawControlBlock.DataBlockPointers[i] = node.DataBlockHashQueue[i]
	}
	node.DataBlockHashQueue = node.DataBlockHashQueue[lineNum:]
	PrintStructJson(node.DataBlockHashQueue)

	// TODO: Should be signed
	controlBlock := rawControlBlock

	node.PBFTRound.ControlBlock = controlBlock

	controlBlockHash := Hash(controlBlock)

	// Phase 2: generate new PBFT msg

	pbftMsg := PBFTMsg{
		SourceID:         node.NodeID,
		RoundNum:         node.PBFTRound.RoundNum,
		PBFTMsgType:      PBFTMSGTYPE_PRE_PREPARE,
		ControlBlockHash: controlBlockHash,
		PBFTMsgContent:   controlBlock,
	}
	pbftMsgHash := Hash(pbftMsg)

	node.PBFTMsgHashQueue = append(node.PBFTMsgHashQueue, pbftMsgHash)
	node.SafePBFTMsgPool.Lock()
	node.SafePBFTMsgPool.PBFTMsgPool[pbftMsgHash] = pbftMsg
	node.SafePBFTMsgPool.Unlock()

	invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
	node.BroadcastMsg(&invMsg, node.NodeID)

	// Should it be outside this function? Or try to enter Prepare?
	node.PBFTRound.RoundStage = STAGE_PREPARE

	fmt.Println("[Complete PrePrepare]")
}

func (node *Node) Prepare() {
	if (node.PBFTRound.RoundRole != ROLE_VALIDATOR) || (node.PBFTRound.RoundStage != STAGE_PREPARE) {
		log.Panic("PANIC 1 in Prepare()")
	}

	// Control block of this round is stored in the PBFTRound
	// TODO: Need verification.

	controlBlockHash := Hash(node.PBFTRound.ControlBlock)
	pbftMsg := PBFTMsg{
		SourceID:         node.NodeID,
		RoundNum:         node.PBFTRound.RoundNum,
		PBFTMsgType:      PBFTMSGTYPE_PREPARE,
		ControlBlockHash: controlBlockHash,
		PBFTMsgContent: Vote{
			VoteID:     node.NodeID,
			VoteResult: true,
		},
	}
	pbftMsgHash := Hash(pbftMsg)

	node.PBFTMsgHashQueue = append(node.PBFTMsgHashQueue, pbftMsgHash)
	node.SafePBFTMsgPool.Lock()
	node.SafePBFTMsgPool.PBFTMsgPool[pbftMsgHash] = pbftMsg
	node.SafePBFTMsgPool.Unlock()

	invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
	node.BroadcastMsg(&invMsg, node.NodeID)

	// Should it be outside this function?
	node.TryEnterCommit()
}

func (node *Node) Commit() {
	if ((node.PBFTRound.RoundRole != ROLE_PROPOSER) && (node.PBFTRound.RoundRole != ROLE_VALIDATOR)) || (node.PBFTRound.
		RoundStage != STAGE_COMMIT) {
		log.Panic("PANIC 1 in Commit()")
	}

	controlBlockHash := Hash(node.PBFTRound.ControlBlock)
	pbftMsg := PBFTMsg{
		SourceID:         node.NodeID,
		RoundNum:         node.PBFTRound.RoundNum,
		PBFTMsgType:      PBFTMSGTYPE_COMMIT,
		ControlBlockHash: controlBlockHash,
		PBFTMsgContent: Vote{
			VoteID:     node.NodeID,
			VoteResult: true,
		},
	}
	pbftMsgHash := Hash(pbftMsg)

	node.PBFTMsgHashQueue = append(node.PBFTMsgHashQueue, pbftMsgHash)
	node.SafePBFTMsgPool.Lock()
	node.SafePBFTMsgPool.PBFTMsgPool[pbftMsgHash] = pbftMsg
	node.SafePBFTMsgPool.Unlock()

	invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
	node.BroadcastMsg(&invMsg, node.NodeID)

	node.TryEnterReply()
}

func (node *Node) Reply() {
	if ((node.PBFTRound.RoundRole != ROLE_PROPOSER) && (node.PBFTRound.RoundRole != ROLE_VALIDATOR)) || (node.PBFTRound.
		RoundStage != STAGE_REPLY) {
		log.Panic("PANIC 1 in Reply()")
	}

	controlBlockHash := Hash(node.PBFTRound.ControlBlock)
	pbftMsg := PBFTMsg{
		SourceID:         node.NodeID,
		RoundNum:         node.PBFTRound.RoundNum,
		PBFTMsgType:      PBFTMSGTYPE_REPLY,
		ControlBlockHash: controlBlockHash,
		PBFTMsgContent:   node.PBFTRound.ControlBlock,
	}
	pbftMsgHash := Hash(pbftMsg)

	node.PBFTMsgHashQueue = append(node.PBFTMsgHashQueue, pbftMsgHash)
	node.SafePBFTMsgPool.Lock()
	node.SafePBFTMsgPool.PBFTMsgPool[pbftMsgHash] = pbftMsg
	node.SafePBFTMsgPool.Unlock()

	invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
	node.BroadcastMsg(&invMsg, node.NodeID)

	fmt.Println("[Round Completed]: This round has completed!")
	node.PBFTRound.RoundEnd <- true
}

func (node *Node) Pack() {
	if (node.PBFTRound.RoundRole != ROLE_PACKER) || (node.PBFTRound.RoundStage != STAGE_PACK) {
		log.Panic("PANIC 1 in Pack()")
	}

	rawDataBlock := node.NewRawDataBlock(node.PBFTRound.RoundNum)
	// TODO: Sign the block and verification
	dataBlock := rawDataBlock

	dataBlockHash := Hash(dataBlock)

	node.DataBlockHashQueue = append(node.DataBlockHashQueue, dataBlockHash)

	node.SafeDataBlockPool.Lock()
	node.SafeDataBlockPool.DataBlockPool[dataBlockHash] = dataBlock
	node.SafeDataBlockPool.Unlock()

	invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_DATABLOCK_HASH, dataBlockHash)
	node.BroadcastMsg(&invMsg, node.NodeID)

	node.PBFTRound.RoundStage = STAGE_WAIT_REPLY

	fmt.Println("[Complete Pack]")
}
