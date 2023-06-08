package main

const (
	BLOCKTYPE_CONTROL_BLOCK BlockType = 1
	BLOCKTYPE_DATA_BLOCK    BlockType = 2
)

type (
	BlockType    int
	ControlBlock struct {
		BlockType BlockType

		RoundNum int64

		ProposerID HashValue

		PreviousBlockHash HashValue

		Seed HashValue

		SeedProof HashValue

		Signature Signature

		DataBlockPointers [32876]HashValue
	}

	DataBlock struct {
		BlockType BlockType

		RoundNum int64

		PackerID HashValue

		Signature Signature

		MerkleRoute [2*M - 1]HashValue

		Transactions [M]Transaction
	}
)

func (node *Node) NewRawControlBlock(roundNum int64, previousBlockHash HashValue) ControlBlock {
	return ControlBlock{
		BlockType:         BLOCKTYPE_CONTROL_BLOCK,
		RoundNum:          roundNum,
		ProposerID:        node.NodeID,
		PreviousBlockHash: previousBlockHash,
		Seed:              HashValue{},
		SeedProof:         HashValue{},
		Signature:         Signature{},
		DataBlockPointers: [32876]HashValue{},
	}
}

func (node *Node) NewRawDataBlock(roundNum int64) DataBlock {
	return DataBlock{
		BlockType:    BLOCKTYPE_DATA_BLOCK,
		RoundNum:     roundNum,
		PackerID:     node.NodeID,
		Signature:    Signature{},
		MerkleRoute:  [2*M - 1]HashValue{},
		Transactions: [M]Transaction{},
	}
}
