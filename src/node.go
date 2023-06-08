package main

import (
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"github.com/yoseplee/vrf"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Node struct {
	NodeName string

	NodeIP string

	NodeID HashValue

	PeerInfo PeerInfoMap

	MsgContentHashQueue []HashValue

	DataBlockHashQueue []HashValue

	SafeDataBlockPool SafeDataBlockPoolMap

	PBFTMsgHashQueue []HashValue

	SafePBFTMsgPool SafePBFTMsgPoolMap

	NetworkDataHashQueue []HashValue

	SafeNetworkDataPool SafeNetworkDataPoolMap

	// TODO: Could store all the PBFT Rounds
	PBFTRound PBFTRound

	// Could it be a map?
	Ledger []ControlBlock

	VRFSeed    HashValue
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

func NewNode() Node {
	IPAddresses, err := net.InterfaceAddrs()
	IPAddress := ""
	if err != nil {
		log.Panic(err)
	}
	for _, address := range IPAddresses {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				IPAddress = ipNet.IP.String()
			}
		}
	}

	peerInfo := PeerInfoMap{}

	peerAddresses := strings.Split(*PeerAddresses, ";")
	if len(peerAddresses) != *PeerNum {
		log.Panic("Peer number problem!")
	} else {
		for i := range peerAddresses {
			ip := strings.Split(peerAddresses[i], ":")[0]
			peerInfo[Hash(ip)] = []string{IPAddress, peerAddresses[i]}
		}
	}

	publicKey, privateKey := GenerateKeys(nil)

	return Node{
		NodeName:            *NodeName,
		NodeIP:              IPAddress,
		NodeID:              Hash(IPAddress),
		PeerInfo:            peerInfo,
		MsgContentHashQueue: []HashValue{},
		DataBlockHashQueue:  []HashValue{},
		SafeDataBlockPool: SafeDataBlockPoolMap{
			RWMutex:       sync.RWMutex{},
			DataBlockPool: map[HashValue]DataBlock{},
		},
		PBFTMsgHashQueue: []HashValue{},
		SafePBFTMsgPool: SafePBFTMsgPoolMap{
			RWMutex:     sync.RWMutex{},
			PBFTMsgPool: map[HashValue]PBFTMsg{},
		},
		NetworkDataHashQueue: []HashValue{},
		SafeNetworkDataPool: SafeNetworkDataPoolMap{
			RWMutex:         sync.RWMutex{},
			NetworkDataPool: map[HashValue]NetworkData{},
		},
		PBFTRound:  PBFTRound{},
		Ledger:     []ControlBlock{},
		VRFSeed:    HashValue{},
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}

func (node *Node) Start() {
	if DEBUG {
		PrintStructJson(node.PeerInfo)
	}

	initialPortNum := 10000
	for i := 0; i < *PeerNum; i++ {
		portNum := initialPortNum + i
		localListeningAddress := node.NodeIP + ":" + strconv.Itoa(portNum)
		go node.TcpListen(localListeningAddress)
	}

	if *TestMode == "network" {
		node.StartNetworkTest(NETWORK_TEST_NUM)
		for {

		}
	} else if *TestMode == "blockchain" {
		node.LoadGenesisBlock()
		roundNum := int64(1)
		fmt.Println("Start at ", time.Now().UnixNano())
		startTime := time.Now().UnixNano()
		for {
			if roundNum == 2 {
				startTime = time.Now().UnixNano()
			}
			// startTime := time.Now().UnixNano()
			// fmt.Println("Start at ",startTime)
			fmt.Printf("+++++++++++ROUND %d START+++++++++++\n", roundNum)
			node.StartAPBFTRound(roundNum)
			select {
			case <-node.PBFTRound.Timer.C:
				node.Ledger = append(node.Ledger, ControlBlock{})
				fmt.Printf("===========TIMEOUT FOR ROUND %d===========\n\n", roundNum)
			case <-node.PBFTRound.RoundEnd:
				if node.PBFTRound.RoundRole == ROLE_PROPOSER || node.PBFTRound.RoundRole == ROLE_VALIDATOR {
					time.Sleep(time.Second * 20)
				}
				if DEBUG {
					PrintStructJson(node.Ledger[len(node.Ledger)-1])
				}
				fmt.Printf("===========ROUND %d END WITH %d BLOCKS===========\n\n", roundNum, len(node.Ledger))
			}
			// PrintStructJson(node.DataBlockHashQueue)
			// roundTime := time.Now().UnixNano() - startTime
			totalTime := time.Now().UnixNano() - startTime
			if roundNum > 2 {
				throughput := node.CalculateThroughput(roundNum, totalTime)
				fmt.Printf("\n!!!!!!!!!!!!!!!!The throughput is %f txn/s!!!!!!!!!!!!!!!!\n", throughput)
			}

			roundNum += 1
		}
	} else {
		log.Panic("PANIC 1 in Start()")
	}
}

func (node *Node) LoadGenesisBlock() {
	genesisBlock := ControlBlock{
		BlockType:         0,
		RoundNum:          0,
		ProposerID:        HashValue{},
		PreviousBlockHash: HashValue{},
		Seed:              HashValue{},
		SeedProof:         HashValue{},
		Signature:         Signature{},
		DataBlockPointers: [32876]HashValue{},
	}
	node.Ledger = append(node.Ledger, genesisBlock)
	dataBlock := DataBlock{
		BlockType:    0,
		RoundNum:     0,
		PackerID:     HashValue{},
		Signature:    Signature{},
		MerkleRoute:  [2*M - 1]HashValue{},
		Transactions: [M]Transaction{},
	}
	dataBlockHash := Hash(dataBlock)
	node.SafeDataBlockPool.Lock()
	node.SafeDataBlockPool.DataBlockPool[Hash(dataBlockHash)] = dataBlock
	node.SafeDataBlockPool.Unlock()

	node.DataBlockHashQueue = append(node.DataBlockHashQueue, dataBlockHash)
}
func (node *Node) VRFSelection(shift int64) (Role, []byte, []byte, int64) {
	shift = 0
	indexStr := strings.Split(node.NodeName, "_")
	nodeIndex, err := strconv.Atoi(indexStr[len(indexStr)-1])
	if err != nil {
		log.Panic(err)
	}
	index := (int64(nodeIndex) + shift) % int64(TOTALNODE_NUM)

	// TODO: replace later
	if index == 0 {
		return ROLE_PROPOSER, []byte{}, []byte{}, 1
	} else if index < int64(PROPOSER_NUM+VALIDATOR_NUM) {
		return ROLE_VALIDATOR, []byte{}, []byte{}, 1
	} else if index < int64(PROPOSER_NUM+VALIDATOR_NUM+PACKER_NUM) {
		return ROLE_PACKER, []byte{}, []byte{}, 1
	} else {
		return ROLE_SIMPLE, nil, nil, 0
	}
}

func (node *Node) StartAPBFTRound(roundNum int64) {
	// fmt.Println(hex.EncodeToString(node.VRFSeed[:]))
	rand.Seed(int64(binary.BigEndian.Uint64(node.VRFSeed[:])))
	randNum := rand.Intn(TOTALNODE_NUM)
	role, _, _, _ := node.VRFSelection(int64(randNum))
	fmt.Println(role)

	if role == ROLE_PROPOSER {
		time.Sleep(time.Second * 5)
		node.PBFTRound = node.NewPBFTRound(roundNum, ROLE_PROPOSER, STAGE_PRE_PREPARE)

		_, vrfSeedHash, err := vrf.Prove(node.PublicKey, node.PrivateKey, append(node.VRFSeed[:],
			[]byte(strconv.Itoa(int(node.PBFTRound.RoundNum)))...))
		if err != nil {
			log.Panic(err)
		}
		node.VRFSeed = Hash(vrfSeedHash)

		node.PrePrepare()
	} else if role == ROLE_VALIDATOR {
		node.PBFTRound = node.NewPBFTRound(roundNum, ROLE_VALIDATOR, STAGE_WAIT_PRE_PREPARE)
	} else if role == ROLE_PACKER {
		node.PBFTRound = node.NewPBFTRound(roundNum, ROLE_PACKER, STAGE_PACK)
		node.Pack()
	} else if role == ROLE_SIMPLE {
		node.PBFTRound = node.NewPBFTRound(roundNum, ROLE_SIMPLE, STAGE_WAIT_REPLY)
	} else {
		log.Panic("PANIC 1 in StartAPBFTRound()")
	}
}

func (node *Node) CalculateThroughput(roundNum int64, roundTime int64) float64 {
	fmt.Println("Total time: ", (float64(0*(roundNum-3)) + float64(roundTime)/float64(1e9)))
	//tmp := HashValue{}
	txnNumCount := float64(0)
	for i := int64(0); i < roundNum; i++ {
		//for j := 0; j < len(node.Ledger[i].DataBlockPointers); j++ {
		//	if node.Ledger[i].DataBlockPointers[j] != tmp {
		//		txnNumCount += 1
		//	}
		//}
		txnNumCount += 1
		fmt.Println(txnNumCount)
	}

	return txnNumCount * float64(M) / (float64(0*(roundNum-3)) + float64(roundTime)/float64(1e9))
}
