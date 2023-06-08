package main

import (
	"flag"
	"math"
	"time"
)

var TestMode = flag.String("testMode", "blockchain", "Indicate the test mode")

var NodeName = flag.String("name", "node_0", "The node name.")
var PeerNum = flag.Int("peerNum", 4, "The number of its peer nodes.")
var PeerAddresses = flag.String("peerAddresses", "127.0.0.1:10001;127.0.0.1:10002;127.0.0.1:10003;127.0.0.1:10004", "The TCP listening addresses of its peer nodes for this node")

const (
	DEBUG            bool          = false
	NETWORK_TEST_NUM int           = 3
	ROUND_TIME       time.Duration = 600 * time.Second
	TOTALNODE_NUM    int           = 100
	PROPOSER_NUM     int           = 1
	VALIDATOR_NUM    int           = 9
	PACKER_NUM       int           = 0
	COMMITTEE_NUM    int           = PROPOSER_NUM + VALIDATOR_NUM
	M                int           = 1820 * 4
)

var FNum = int64(math.Floor((float64(COMMITTEE_NUM) - 1) / 3))

func main() {
	flag.Parse()

	node := NewNode()

	node.Start()
}
