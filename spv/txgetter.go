package spv

import (
	"github.com/libsv/go-bc"
	"github.com/libsv/go-bt/v2"
)

// TXGetter gets a tx from a provided id
type TXGetter interface {
	Tx(txID string) (*bt.Tx, error)
}

// MerkleProofGetter gets a merkle proof for a provided tx id
type MerkleProofGetter interface {
	MerkleProof(txID string) (*bc.MerkleProof, error)
}
