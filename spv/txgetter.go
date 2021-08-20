package spv

import (
	"github.com/libsv/go-bc"
	"github.com/libsv/go-bt/v2"
)

type TXGetter interface {
	Tx(txID string) (*bt.Tx, error)
}

type MerkleProofGetter interface {
	MerkleProof(txID string) (*bc.MerkleProof, error)
}
