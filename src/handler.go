package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
)

func (node *Node) HandleMsg(bMsg []byte, peerIPAddress string) {
	err, msgType, sourceID, timestamp, dataType, msgContentHash, rawMsg := node.ResolveMsg(bMsg)
	if DEBUG {
		fmt.Println("Msg source is", hex.EncodeToString(sourceID[:]))
	}

	if err != nil {
		if DEBUG {
			fmt.Println("PANIC 1 in HandleMsg()", err)
			return
		} else {
			fmt.Println("PANIC 1 in HandleMsg()", err)
			return
			// log.Panic(err)
		}
	}

	// through the first inv msg to fill the peerInfo, this may be initialized in commands
	if node.PeerInfo[sourceID][0] == node.NodeIP {
		node.PeerInfo[sourceID][0] = peerIPAddress
		if DEBUG {
			PrintStructJson(node.PeerInfo)
		}
	}

	msgPeerID := HashValue{}
	if node.PeerInfo[sourceID][0] != node.NodeIP {
		msgPeerID = sourceID
	}

	if IsHashContain(msgContentHash, node.MsgContentHashQueue) {
		if DEBUG {
			fmt.Println("duplicated msg content.")
		}
		return
	} else {
		node.MsgContentHashQueue = append(node.MsgContentHashQueue, msgContentHash)
	}

	if msgType == MSGTYPE_INV {
		msgContent := HashValue{}
		err = json.Unmarshal(rawMsg["MsgContent"], &msgContent)
		if err != nil {
			log.Panic("PANIC 2 in HandleMsg()", err)
		}
		node.HandleInvMsg(msgPeerID, sourceID, timestamp, msgContent, dataType)
	} else if msgType == MSGTYPE_GETDATA {
		msgContent := GetDataContent{}
		err = json.Unmarshal(rawMsg["MsgContent"], &msgContent)
		if err != nil {
			log.Panic("PANIC 3 in HandleMsg()", err)
		}
		node.HandleGetDataMsg(msgPeerID, sourceID, timestamp, msgContent.DataHash, dataType)
	} else if msgType == MSGTYPE_DATA {
		node.HandleDataMsg(msgPeerID, dataType, sourceID, timestamp, rawMsg)
	} else {
		fmt.Println(msgType, sourceID, timestamp, dataType, hex.EncodeToString(msgContentHash[:]), rawMsg)
		log.Panic("PANIC 4 in HandleMsg()", err)
	}
}

func (node *Node) HandleInvMsg(msgPeerID HashValue, sourceID HashValue, timestamp int64, dataHash HashValue,
	dataType DataType) {

	if msgPeerID != sourceID {
		fmt.Println("PANIC 0 in HandleInvMsg()")
	}

	getDataMsg := Msg{}
	if dataType == DATATYPE_DATABLOCK_HASH {
		if DEBUG {
			fmt.Println("Handling data block inv msg from", hex.EncodeToString(msgPeerID[:]), "with hash", hex.EncodeToString(dataHash[:]))
		}

		if IsHashContain(dataHash, node.DataBlockHashQueue) {
			// log.Panic("PANIC 1 in HandleInvMsg()")
			if DEBUG {
				fmt.Println("PANIC 1 in HandleInvMsg()", "Not a big deal?")
			}
		} else {
			getDataMsg = node.NewMsg(MSGTYPE_GETDATA, DATATYPE_DATABLOCK_HASH, GetDataContent{
				DataHash:      dataHash,
				RequireNodeID: node.NodeID,
			})
			// To avoid duplicated inv
			node.DataBlockHashQueue = append(node.DataBlockHashQueue, dataHash)

			node.SendMsg(&getDataMsg, msgPeerID)
			if DEBUG {
				fmt.Println("Require data block data with hash", hex.EncodeToString(dataHash[:]))
				PrintStructJson(node.DataBlockHashQueue)
			}

		}
	} else if dataType == DATATYPE_PBFTMSG_HASH {
		if DEBUG {
			fmt.Println("Handling pbft msg inv msg from", hex.EncodeToString(msgPeerID[:]), "with hash", hex.EncodeToString(dataHash[:]))
		}
		if IsHashContain(dataHash, node.PBFTMsgHashQueue) {
			fmt.Println("(Not) PANIC 2 in HandleInvMsg()")
			return
		} else {
			getDataMsg = node.NewMsg(MSGTYPE_GETDATA, DATATYPE_PBFTMSG_HASH, GetDataContent{
				DataHash:      dataHash,
				RequireNodeID: node.NodeID,
			})
			node.PBFTMsgHashQueue = append(node.PBFTMsgHashQueue, dataHash)
			node.SendMsg(&getDataMsg, msgPeerID)
			if DEBUG {
				fmt.Println("Require pbft msg data with hash", hex.EncodeToString(dataHash[:]))
				PrintStructJson(node.PBFTMsgHashQueue)
			}
		}
	} else if dataType == DATATYPE_NETWORKDATA_HASH {
		if DEBUG {
			fmt.Println("Handling network data inv msg from", hex.EncodeToString(msgPeerID[:]), "with hash", hex.EncodeToString(dataHash[:]))
		}
		if IsHashContain(dataHash, node.NetworkDataHashQueue) {
			// log.Panic("PANIC 3 in HandleInvMsg()")
			fmt.Println("(Not) PANIC 3 in HandleInvMsg()")
			return
		} else {
			getDataMsg = node.NewMsg(MSGTYPE_GETDATA, DATATYPE_NETWORKDATA_HASH, GetDataContent{
				DataHash:      dataHash,
				RequireNodeID: node.NodeID,
			})
			node.NetworkDataHashQueue = append(node.NetworkDataHashQueue, dataHash)
			node.SendMsg(&getDataMsg, msgPeerID)

			if DEBUG {
				fmt.Println("Require network data with hash", hex.EncodeToString(dataHash[:]))
			}
		}
	} else {
		log.Panic("PANIC 4 in HandleInvMsg()")
	}
}

func (node *Node) HandleGetDataMsg(msgPeerID HashValue, sourceID HashValue, timestamp int64, dataHash HashValue,
	dataType DataType) {

	if msgPeerID != sourceID {
		fmt.Println("PANIC 0 in HandleGetDataMsg()")
	}

	dataMsg := Msg{}
	if dataType == DATATYPE_DATABLOCK_HASH {
		if DEBUG {
			fmt.Println("Handling data block get msg from", hex.EncodeToString(msgPeerID[:]), "with hash", hex.EncodeToString(dataHash[:]))
		}
		node.SafeDataBlockPool.Lock()
		dataBlock, isContained := node.SafeDataBlockPool.DataBlockPool[dataHash]
		node.SafeDataBlockPool.Unlock()

		if isContained {
			dataMsg = node.NewMsg(MSGTYPE_DATA, DATATYPE_DATA_BLOCK, dataBlock)
			node.SendMsg(&dataMsg, msgPeerID)
		} else {
			if DEBUG {
				PrintStructJson(node.DataBlockHashQueue)
			}

			node.SafeDataBlockPool.Lock()
			if DEBUG {
				PrintStructJson(node.SafeDataBlockPool.DataBlockPool)
			}
			node.SafeDataBlockPool.Unlock()
			fmt.Println("!!!!!!!FUCKKKKK!!!! No data block data with hash", hex.EncodeToString(dataHash[:]))
		}
	} else if dataType == DATATYPE_PBFTMSG_HASH {
		if DEBUG {
			fmt.Println("Handling pbft msg get msg from ", hex.EncodeToString(msgPeerID[:]), "with hash", hex.EncodeToString(dataHash[:]))
		}
		node.SafePBFTMsgPool.Lock()
		pbftMsg, isContained := node.SafePBFTMsgPool.PBFTMsgPool[dataHash]
		node.SafePBFTMsgPool.Unlock()
		if isContained {
			dataMsg = node.NewMsg(MSGTYPE_DATA, DATATYPE_PBFT_MSG, pbftMsg)
			node.SendMsg(&dataMsg, msgPeerID)
		} else {
			if DEBUG {
				PrintStructJson(node.PBFTMsgHashQueue)
			}

			node.SafePBFTMsgPool.Lock()
			if DEBUG {
				PrintStructJson(node.SafePBFTMsgPool.PBFTMsgPool)
			}

			node.SafePBFTMsgPool.Unlock()
			fmt.Println("!!!!!!!FUCKKKKK!!!! No pbft msg data with hash", hex.EncodeToString(dataHash[:]))
		}

	} else if dataType == DATATYPE_NETWORKDATA_HASH {
		if DEBUG {
			fmt.Println("Handling network data get msg from ", hex.EncodeToString(msgPeerID[:]), "with hash", hex.EncodeToString(dataHash[:]))
		}
		node.SafeNetworkDataPool.Lock()
		networkData, isContained := node.SafeNetworkDataPool.NetworkDataPool[dataHash]
		node.SafeNetworkDataPool.Unlock()

		if isContained {
			dataMsg = node.NewMsg(MSGTYPE_DATA, DATATYPE_NETWORKDATA, networkData)
			node.SendMsg(&dataMsg, msgPeerID)
		} else {
			fmt.Println("!!!!!!!FUCKKKKK!!!!")
		}
	} else {
		log.Panic("PANIC 1 in HandleGetDataMsg()")
	}

}

func (node *Node) HandleDataMsg(msgPeerID HashValue, dataType DataType, sourceID HashValue, timestamp int64,
	rawMsg map[string]json.RawMessage) {

	if msgPeerID != sourceID {
		fmt.Println("PANIC 0 in HandleDataMsg()")
	}

	if dataType == DATATYPE_DATA_BLOCK {
		node.HandleDataBlockMsg(msgPeerID, timestamp, rawMsg)
	} else if dataType == DATATYPE_PBFT_MSG {
		node.HandlePBFTMsg(msgPeerID, timestamp, rawMsg)
	} else if dataType == DATATYPE_NETWORKDATA {
		node.HandleNetworkDataMsg(msgPeerID, timestamp, rawMsg)
	} else {
		log.Panic("PANIC 1 in HandleDataMsg()")
	}
}

func (node *Node) HandleDataBlockMsg(msgPeerID HashValue, timestamp int64,
	rawMsg map[string]json.RawMessage) {
	if DEBUG {
		fmt.Println("Handling data block msg from ", hex.EncodeToString(msgPeerID[:]), "...")
	}
	dataBlock := DataBlock{}
	err := json.Unmarshal(rawMsg["MsgContent"], &dataBlock)
	if err != nil {
		log.Panic("PANIC 1 in HandleDataBlockMsg()", err)
	}
	if DEBUG {
		fmt.Println("[DataBlock Received] Received a data block originated from", hex.EncodeToString(dataBlock.PackerID[:]), "at", timestamp)
	}

	dataBlockHash := Hash(dataBlock)

	node.SafeDataBlockPool.Lock()
	node.SafeDataBlockPool.DataBlockPool[dataBlockHash] = dataBlock
	node.SafeDataBlockPool.Unlock()

	invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_DATABLOCK_HASH, dataBlockHash)

	node.BroadcastMsg(&invMsg, msgPeerID)
}

func (node *Node) HandlePBFTMsg(msgPeerID HashValue, timestamp int64,
	rawMsg map[string]json.RawMessage) {
	if DEBUG {
		fmt.Println("Handling pbft msg from peer", hex.EncodeToString(msgPeerID[:]), "...")
	}
	bPBFTMsg := rawMsg["MsgContent"]
	err, pbftMsgSourceID, roundNum, pbftMsgType, controlBlockHash, rawPBFTMsg := node.ResolvePBFTMsg(bPBFTMsg)
	if err != nil {
		log.Panic("PANIC 1 in HandlePBFTMsg()", err)
	}

	//if roundNum != node.PBFTRound.RoundNum {
	//	if DEBUG {
	//		fmt.Println("Not this round PBFT msg!")
	//	}
	//	return
	//}

	// PBFTMsgHashQueue has already recorded this hash at inv stage

	if pbftMsgType == PBFTMSGTYPE_PRE_PREPARE {
		// handle pre-prepare
		controlBlock := ControlBlock{}
		err = json.Unmarshal(rawPBFTMsg["PBFTMsgContent"], &controlBlock)
		if err != nil {
			log.Panic("PANIC 2 in HandlePBFTMsg()", err)
		}

		pbftMsg := PBFTMsg{
			SourceID:         pbftMsgSourceID,
			RoundNum:         roundNum,
			PBFTMsgType:      pbftMsgType,
			ControlBlockHash: controlBlockHash,
			PBFTMsgContent:   controlBlock,
		}
		pbftMsgHash := Hash(pbftMsg)
		node.SafePBFTMsgPool.Lock()
		node.SafePBFTMsgPool.PBFTMsgPool[pbftMsgHash] = pbftMsg
		node.SafePBFTMsgPool.Unlock()

		// whether process it or directly forward it
		if (roundNum != node.PBFTRound.RoundNum) || (node.PBFTRound.RoundStage != STAGE_WAIT_PRE_PREPARE) || (node.
			PBFTRound.RoundRole != ROLE_VALIDATOR) {
			// TODO: should or not?
			node.VRFSeed = controlBlock.Seed
			node.PBFTRound.ControlBlock = controlBlock

			// TODO: should this invMSg be added into the msg hash queue? NO?
			invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
			node.BroadcastMsg(&invMsg, msgPeerID)
		} else {
			node.HandlePrePrepare(controlBlock)
			invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
			node.BroadcastMsg(&invMsg, msgPeerID)
		}

	} else if pbftMsgType == PBFTMSGTYPE_PREPARE {
		// handle prepare
		vote := Vote{}
		err = json.Unmarshal(rawPBFTMsg["PBFTMsgContent"], &vote)
		if err != nil {
			log.Panic("PANIC 3 in HandlePBFTMsg()", err)
		}

		pbftMsg := PBFTMsg{
			SourceID:         pbftMsgSourceID,
			RoundNum:         roundNum,
			PBFTMsgType:      pbftMsgType,
			ControlBlockHash: controlBlockHash,
			PBFTMsgContent:   vote,
		}
		pbftMsgHash := Hash(pbftMsg)
		node.SafePBFTMsgPool.Lock()
		node.SafePBFTMsgPool.PBFTMsgPool[pbftMsgHash] = pbftMsg
		node.SafePBFTMsgPool.Unlock()

		if (roundNum != node.PBFTRound.RoundNum) || ((node.PBFTRound.RoundRole != ROLE_VALIDATOR) && (node.PBFTRound.
			RoundRole != ROLE_PROPOSER)) {
			invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
			node.BroadcastMsg(&invMsg, msgPeerID)
		} else {
			node.HandlePrepare(vote)
			invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
			node.BroadcastMsg(&invMsg, msgPeerID)
		}

	} else if pbftMsgType == PBFTMSGTYPE_COMMIT {
		// handle commit
		vote := Vote{}
		err = json.Unmarshal(rawPBFTMsg["PBFTMsgContent"], &vote)
		if err != nil {
			log.Panic("PANIC 4 in HandlePBFTMsg()", err)
		}

		pbftMsg := PBFTMsg{
			SourceID:         pbftMsgSourceID,
			RoundNum:         roundNum,
			PBFTMsgType:      pbftMsgType,
			ControlBlockHash: controlBlockHash,
			PBFTMsgContent:   vote,
		}
		pbftMsgHash := Hash(pbftMsg)
		node.SafePBFTMsgPool.Lock()
		node.SafePBFTMsgPool.PBFTMsgPool[pbftMsgHash] = pbftMsg
		node.SafePBFTMsgPool.Unlock()

		if (roundNum != node.PBFTRound.RoundNum) || ((node.PBFTRound.RoundRole != ROLE_VALIDATOR) && (node.PBFTRound.
			RoundRole != ROLE_PROPOSER)) {
			invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
			node.BroadcastMsg(&invMsg, msgPeerID)
		} else {
			node.HandelCommit(vote)
			invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
			node.BroadcastMsg(&invMsg, msgPeerID)
		}

	} else if pbftMsgType == PBFTMSGTYPE_REPLY {
		// handle reply
		controlBlock := ControlBlock{}
		err = json.Unmarshal(rawPBFTMsg["PBFTMsgContent"], &controlBlock)
		if err != nil {
			log.Panic("PANIC 5 in HandlePBFTMsg()", err)
		}

		pbftMsg := PBFTMsg{
			SourceID:         pbftMsgSourceID,
			RoundNum:         roundNum,
			PBFTMsgType:      pbftMsgType,
			ControlBlockHash: controlBlockHash,
			PBFTMsgContent:   controlBlock,
		}
		pbftMsgHash := Hash(pbftMsg)
		node.SafePBFTMsgPool.Lock()
		node.SafePBFTMsgPool.PBFTMsgPool[pbftMsgHash] = pbftMsg
		node.SafePBFTMsgPool.Unlock()

		if (roundNum != node.PBFTRound.RoundNum) || ((node.PBFTRound.RoundRole != ROLE_PACKER) && (node.PBFTRound.
			RoundRole != ROLE_SIMPLE) && (node.PBFTRound.
			RoundStage != STAGE_WAIT_REPLY)) {

			invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
			node.BroadcastMsg(&invMsg, msgPeerID)
		} else {
			node.HandleReply(controlBlock)
			invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_PBFTMSG_HASH, pbftMsgHash)
			node.BroadcastMsg(&invMsg, msgPeerID)
		}

	} else {
		log.Panic("PANIC 6 in HandlePBFTMsg()")
	}
}
func (node *Node) HandlePrePrepare(controlBlock ControlBlock) {
	fmt.Println("{Handle PrePrepare}")

	//TODO: Verify the control block
	node.VRFSeed = controlBlock.Seed
	node.PBFTRound.ControlBlock = controlBlock
	node.PBFTRound.RoundStage = STAGE_PREPARE
	node.Prepare()

}

func (node *Node) HandlePrepare(vote Vote) {
	fmt.Println("{Handle Prepare}")

	// TODO: we skip this consensus body checking for there is a strong synchronization check.
	if vote.VoteResult {
		node.PBFTRound.PrepareVoteCount += 1
	}
	node.TryEnterCommit()
}

func (node *Node) HandelCommit(vote Vote) {
	fmt.Println("{Handle Commit}")

	// TODO: Verification.
	if vote.VoteResult {
		node.PBFTRound.CommitVoteCount += 1
	}
	node.TryEnterReply()
}

func (node *Node) HandleReply(controlBlock ControlBlock) {
	fmt.Println("{Handle Reply}")

	if Hash(controlBlock) == Hash(node.PBFTRound.ControlBlock) {
		node.PBFTRound.ReplyVoteCount += 1
	}

	node.TryEndRound()
}

func (node *Node) HandleNetworkDataMsg(msgPeerID HashValue, timestamp int64,
	rawMsg map[string]json.RawMessage) {
	fmt.Println("Handling network data msg from peer", hex.EncodeToString(msgPeerID[:]), "...")

	networkData := NetworkData{}
	err := json.Unmarshal(rawMsg["MsgContent"], &networkData)
	if err != nil {
		log.Panic(err)
	}

	fmt.Println("Received a networkData originated from", hex.EncodeToString(networkData.SourceID[:]), "at", timestamp)

	networkDataHash := Hash(networkData)

	node.SafeNetworkDataPool.Lock()
	node.SafeNetworkDataPool.NetworkDataPool[networkDataHash] = networkData
	node.SafeNetworkDataPool.Unlock()

	invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_NETWORKDATA_HASH, networkDataHash)

	// TODO: Check the performance?
	node.BroadcastMsg(&invMsg, msgPeerID)
}

func (node *Node) TryEnterCommit() {
	if node.PBFTRound.PrepareVoteCount >= 2*FNum-1 {
		if node.PBFTRound.RoundStage == STAGE_PREPARE {
			fmt.Println("[Complete Prepare]")
			node.PBFTRound.RoundStage = STAGE_COMMIT
			node.Commit()
		} else {
			fmt.Println("[Wait Prepare]: Enough votes but not convert the state")
			return
		}
	} else {
		fmt.Println("[Wait Prepare]: There are", node.PBFTRound.PrepareVoteCount, "votes now for Prepare stage.")
		return
	}
}

func (node *Node) TryEnterReply() {
	if node.PBFTRound.CommitVoteCount >= 2*FNum {
		if node.PBFTRound.RoundStage == STAGE_COMMIT {
			node.Ledger = append(node.Ledger, node.PBFTRound.ControlBlock)
			fmt.Println("[Complete Commit]")
			node.PBFTRound.RoundStage = STAGE_REPLY
			node.Reply()
		} else {
			fmt.Println("[Wait Commit]: Enough votes but not convert the state")
			return
		}
	} else {
		fmt.Println("[Wait Commit]: There is", node.PBFTRound.CommitVoteCount, "votes now for Commit stage.")
		return
	}
}

func (node *Node) TryEndRound() {
	controlBlock := node.PBFTRound.ControlBlock

	if node.PBFTRound.ReplyVoteCount >= FNum+1 {
		if node.PBFTRound.RoundStage == STAGE_WAIT_REPLY {
			node.PBFTRound.RoundStage = STAGE_REPLY
			node.Ledger = append(node.Ledger, controlBlock)

			for i := 0; i < len(controlBlock.DataBlockPointers); i++ {
				tmp := HashValue{}
				if controlBlock.DataBlockPointers[i] != tmp {
					node.SearchAndDeleteDataBlock(controlBlock.DataBlockPointers[i])
				}
			}

			node.VRFSeed = controlBlock.Seed

			node.PBFTRound.RoundEnd <- true
			fmt.Println("[Round Completed]: This round has completed!")
		} else {
			fmt.Println("[Wait Reply]: Enough votes but not convert the state")
		}
	} else {
		fmt.Println("[Wait Reply]: There is", node.PBFTRound.ReplyVoteCount, "votes now for Reply stage.")
	}
}
