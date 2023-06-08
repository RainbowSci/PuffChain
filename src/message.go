package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type (
	MsgType  int
	DataType int

	GetDataContent struct {
		DataHash HashValue

		// RequireNodeID will be same as the SourceID of this msg.
		RequireNodeID HashValue
	}

	MsgHeader struct {
		// [inv, getData, data]
		MsgType int

		Timestamp int64

		SourceID HashValue

		// [hashVale, dataBlock, PBFT msg, network test msg]
		DataType int

		MsgContentHash HashValue
	}

	Msg struct {
		MsgHeader

		MsgContent interface{}
	}
)

const (
	MSGTYPE_INV               = 1
	MSGTYPE_GETDATA           = 2
	MSGTYPE_DATA              = 3
	DATATYPE_DATABLOCK_HASH   = 1
	DATATYPE_PBFTMSG_HASH     = 2
	DATATYPE_NETWORKDATA_HASH = 3
	DATATYPE_DATA_BLOCK       = 4
	DATATYPE_PBFT_MSG         = 5
	DATATYPE_NETWORKDATA      = 6
)

func (node *Node) NewMsg(msgType int, dataType int, msgContent interface{}) Msg {
	return Msg{
		MsgHeader: MsgHeader{
			MsgType:        msgType,
			SourceID:       node.NodeID,
			Timestamp:      0,
			DataType:       dataType,
			MsgContentHash: Hash(msgContent),
		},
		MsgContent: msgContent,
	}
}

func (node *Node) BroadcastMsg(msg *Msg, msgPeerID HashValue) {
	// Msg changed here so pointer as the parameter.
	if msg.MsgType != MSGTYPE_INV {
		log.Panic("PANIC 1 in BroadcastMsg()")
	}

	transformed, isTransformed := (msg.MsgContent).(HashValue)
	if !isTransformed {
		PrintStructJson(msg)
		log.Panic("PANIC 2 in BroadcastMsg()")
	}
	msg.MsgContent = transformed
	msg.Timestamp = time.Now().UnixNano()

	bMessage, err := json.Marshal(*msg)
	if err != nil {
		log.Panic("PANIC 3 in BroadcastMsg()", err)
	}

	for peerID, addresses := range node.PeerInfo {
		if peerID != msgPeerID {
			go TcpDial(bMessage, addresses[1])
		}
	}
}

func (node *Node) SendMsg(msg *Msg, peerID HashValue) {
	msg.Timestamp = time.Now().UnixNano()

	bMessage, err := json.Marshal(*msg)
	if err != nil {
		log.Panic("PANIC 1 in SendMsg()", err)
	}

	go TcpDial(bMessage, node.PeerInfo[peerID][1])
}

func (node *Node) ResolveMsg(bMsg []byte) (error, MsgType, HashValue, int64, DataType, HashValue,
	map[string]json.RawMessage) {
	// 2. Unmarshal it into the raw message
	rawMsg := map[string]json.RawMessage{}
	err := json.Unmarshal(bMsg, &rawMsg)
	if err != nil {
		if DEBUG {
			fmt.Println("ERROR 1 in ResolveMsg()")
		}
		return err, 0, HashValue{}, 0, 0, HashValue{}, nil
	} else {
		var msgType MsgType
		err = json.Unmarshal(rawMsg["MsgType"], &msgType)
		if err != nil {
			log.Panic("PANIC 1 in ResolveMsg()", err)
		}

		sourceID := HashValue{}
		err = json.Unmarshal(rawMsg["SourceID"], &sourceID)
		if err != nil {
			log.Panic("PANIC 2 in ResolveMsg()", err)
		}

		var timestamp int64
		err = json.Unmarshal(rawMsg["Timestamp"], &timestamp)
		if err != nil {
			log.Panic("PANIC 3 in ResolveMsg()", err)
		}

		var dataType DataType
		err = json.Unmarshal(rawMsg["DataType"], &dataType)
		if err != nil {
			log.Panic("PANIC 4 in ResolveMsg()", err)
		}

		msgContentHash := HashValue{}
		err = json.Unmarshal(rawMsg["MsgContentHash"], &msgContentHash)
		if err != nil {
			log.Panic("PANIC 5 in ResolveMsg()", err)
		}

		return err, msgType, sourceID, timestamp, dataType, msgContentHash, rawMsg
	}
}

func (node *Node) ResolvePBFTMsg(bPBFTMsg []byte) (error, HashValue, int64, PBFTMsgType, HashValue,
	map[string]json.RawMessage) {
	rawPBFTMsg := map[string]json.RawMessage{}
	err := json.Unmarshal(bPBFTMsg, &rawPBFTMsg)
	if err != nil {
		if DEBUG {
			fmt.Println("ERROR 1 in ResolvePBFTMsg()")
		}

		return err, HashValue{}, 0, 0, HashValue{}, nil
	} else {
		pbftMsgSourceID := HashValue{}

		err = json.Unmarshal(rawPBFTMsg["SourceID"], &pbftMsgSourceID)
		if err != nil {
			log.Panic("PANIC 0 in ResolvePBFTMsg()", err)
		}

		var roundNum int64
		err = json.Unmarshal(rawPBFTMsg["RoundNum"], &roundNum)
		if err != nil {
			log.Panic("PANIC 1 in ResolvePBFTMsg()", err)
		}

		var pbftMsgType PBFTMsgType
		err = json.Unmarshal(rawPBFTMsg["PBFTMsgType"], &pbftMsgType)
		if err != nil {
			log.Panic("PANIC 2 in ResolvePBFTMsg()", err)
		}

		controlBlockHash := HashValue{}
		err = json.Unmarshal(rawPBFTMsg["ControlBlockHash"], &controlBlockHash)
		if err != nil {
			log.Panic("PANIC 3 in ResolvePBFTMsg()", err)
		}

		return err, pbftMsgSourceID, roundNum, pbftMsgType, controlBlockHash, rawPBFTMsg
	}

}
