package spv

import (
	"context"
	"errors"

	"github.com/libsv/go-bc"
	"github.com/libsv/go-bt/v2"
)

// An Client is a struct used to specify interfaces
// used to complete Simple Payment Verification (SPV)
// in conjunction with a Merkle Proof.
//
// The implementation of BlockHeaderChain which is supplied will depend on the client
// you are using, some may return a HeaderJSON response others may return the blockhash.
type Client interface {
	EnvelopeHandler
	MerkleProofVerifier
}

// EnvelopeHandler interfaces the handling (creation and verification) of SPV Envelopes
type EnvelopeHandler interface {
	EnvelopeCreator
	EnvelopeVerifier
}

// EnvelopeCreator interfaces the creation of SPV Envelopes
type EnvelopeCreator interface {
	CreateEnvelope(*bt.Tx) (*Envelope, error)
}

// EnvelopeVerifier interfaces the verification of SPV Envelopes
type EnvelopeVerifier interface {
	VerifyPayment(context.Context, *Envelope) (bool, error)
}

// MerkleProofVerifier interfaces the verification of Merkle Proofs
type MerkleProofVerifier interface {
	VerifyMerkleProof(context.Context, []byte) (bool, bool, error)
	VerifyMerkleProofJSON(context.Context, *bc.MerkleProof) (bool, bool, error)
}

type spvclient struct {
	// BlockHeaderChain will be set when an implementation returning a bc.BlockHeader type is provided.
	bhc bc.BlockHeaderChain
	txg TXGetter
	mpg MerkleProofGetter
}

// NewClient creates a new spv.Client based on the options provided.
// If no BlockHeaderChain implementation is provided, the setup will return an error.
func NewClient(opts ...ClientOpts) (Client, error) {
	cli := &spvclient{}
	for _, opt := range opts {
		opt(cli)
	}
	if cli.bhc == nil {
		return nil, errors.New("at least one blockchain header implementation should be returned")
	}
	return cli, nil
}
