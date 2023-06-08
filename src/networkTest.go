package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

type NetworkData struct {
	SourceID  HashValue
	TimeStamp int64
	Content   [1 * 1024 * 1024 / 32]HashValue
}

func (node *Node) StartNetworkTest(sourceNum int) {
	fmt.Println("Start puffchain network test")
	fmt.Println("NodeID is", hex.EncodeToString(node.NodeID[:]))

	indexStr := strings.Split(node.NodeName, "_")
	nodeIndex, err := strconv.Atoi(indexStr[len(indexStr)-1])
	if err != nil {
		log.Panic(err)
	}

	if nodeIndex < sourceNum {

		time.Sleep(time.Second * 3)
		networkData := NetworkData{
			SourceID:  node.NodeID,
			TimeStamp: time.Now().UnixNano(),
			Content:   [1 * 1024 * 1024 / 32]HashValue{},
		}

		node.NetworkDataHashQueue = append(node.NetworkDataHashQueue, Hash(networkData))
		node.SafeNetworkDataPool.Lock()
		node.SafeNetworkDataPool.NetworkDataPool[Hash(networkData)] = networkData
		node.SafeNetworkDataPool.Unlock()

		invMsg := node.NewMsg(MSGTYPE_INV, DATATYPE_NETWORKDATA_HASH, Hash(networkData))

		sendTime := time.Now().UnixNano()
		node.BroadcastMsg(&invMsg, node.NodeID)
		fmt.Println("Send msg at ", sendTime)
	}
}
