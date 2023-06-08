package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

type (
	HashValue   [32]byte
	Signature   [256]byte
	Transaction [512]byte

	// PeerInfoMap is a map struct with for peer information,
	// e.g: {"nodeID":[localAddress,remoteAddress]}.
	PeerInfoMap map[HashValue][]string

	SafeDataBlockPoolMap struct {
		sync.RWMutex
		DataBlockPool map[HashValue]DataBlock
	}

	SafePBFTMsgPoolMap struct {
		sync.RWMutex
		PBFTMsgPool map[HashValue]PBFTMsg
	}

	SafeNetworkDataPoolMap struct {
		sync.RWMutex
		NetworkDataPool map[HashValue]NetworkData
	}
)

func IsHashContain(val HashValue, array []HashValue) bool {
	for _, item := range array {
		if val == item {
			return true
		}
	}
	return false
}

func Hash(content interface{}) HashValue {
	switch content.(type) {
	default:
		bContent, err := json.Marshal(content)
		if err != nil {
			log.Panic(err)
		}
		return sha256.Sum256(bContent)
	}
}

func (node *Node) SearchAndDeleteDataBlock(dataBlockHash HashValue) {
	for i := 0; i < len(node.DataBlockHashQueue); i++ {
		if dataBlockHash == node.DataBlockHashQueue[i] {
			node.SafeDataBlockPool.Lock()
			delete(node.SafeDataBlockPool.DataBlockPool, dataBlockHash)
			node.SafeDataBlockPool.Unlock()
			if i == 0 {
				node.DataBlockHashQueue = node.DataBlockHashQueue[1:]
			} else {
				node.DataBlockHashQueue = append(node.DataBlockHashQueue[:i-1], node.DataBlockHashQueue[i+1:]...)
			}
		}
	}
}

func PrintStructJson(content interface{}) {
	bContent, err := json.Marshal(content)
	if err != nil {
		fmt.Println(err)
	}
	var out bytes.Buffer
	err = json.Indent(&out, bContent, "", "\t")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("content=%v\n", out.String())
}

// MarshalText implements the interface for json encoding.
func (nodeID HashValue) MarshalText() (text []byte, err error) {
	return []byte(hex.EncodeToString(nodeID[:])), nil
}

// UnmarshalText implements the interface for json encoding.
func (nodeID *HashValue) UnmarshalText(text []byte) error {
	id, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}
	copy(nodeID[:], id[:32])
	return nil
}

// MarshalText implements the interface for json encoding.
func (signature Signature) MarshalText() (text []byte, err error) {
	return []byte(hex.EncodeToString(signature[:])), nil
}

// MarshalText implements the interface for json encoding.
func (transaction Transaction) MarshalText() (text []byte, err error) {
	return []byte(hex.EncodeToString(transaction[:])), nil
}

// UnmarshalText implements the interface for json encoding.
func (signature *Signature) UnmarshalText(text []byte) error {
	id, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}
	copy(signature[:], id[:256])
	return nil
}

// UnmarshalText implements the interface for json encoding.
func (transaction *Transaction) UnmarshalText(text []byte) error {
	id, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}
	copy(transaction[:], id[:250])
	return nil
}
