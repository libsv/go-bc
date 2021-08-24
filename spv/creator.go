package spv

import (
	"context"
	"fmt"

	"github.com/libsv/go-bc"
	"github.com/libsv/go-bt/v2"
	"github.com/pkg/errors"
)

// A Creator is an interface used to build the spv.Envelope data type for
// Simple Payment Verification (SPV).
//
// The implementation of an spv.TxStore and spv.MerkleProofStore which is supplied will depend
// on the client you are using.
type Creator interface {
	CreateEnvelope(context.Context, *bt.Tx) (*Envelope, error)
}

// TxStore interfaces the a tx store
type TxStore interface {
	Tx(ctx context.Context, txID string) (*bt.Tx, error)
}

// MerkleProofStore interfaces a Merkle Proof store
type MerkleProofStore interface {
	MerkleProof(ctx context.Context, txID string) (*bc.MerkleProof, error)
}

type creator struct {
	txc TxStore
	mpc MerkleProofStore
}

// NewCreator creates a new spv.Creator with the provided spv.TxStore and tx.MerkleProofStore.
// If either implementation is not provided, the setup will return an error.
func NewCreator(txc TxStore, mpc MerkleProofStore) (Creator, error) {
	if txc == nil {
		return nil, errors.New("an spv.TxStore implementation is required")
	}
	if mpc == nil {
		return nil, errors.New("an spv.MerkleProofStore implementation is required")
	}

	return &creator{txc: txc, mpc: mpc}, nil
}

// CreateEnvelope builds and returns an spv.Envelope for the provided tx.
func (c *creator) CreateEnvelope(ctx context.Context, tx *bt.Tx) (*Envelope, error) {
	if len(tx.Inputs) == 0 {
		return nil, ErrNoTxInputs
	}

	envelope := &Envelope{
		TxID:    tx.TxID(),
		RawTx:   tx.String(),
		Parents: make(map[string]*Envelope),
	}

	for _, input := range tx.Inputs {
		pTxID := input.PreviousTxIDStr()

		// If we already have added the tx to the parent envelope, there's no point in
		// redoing the same work
		if _, ok := envelope.Parents[pTxID]; ok {
			continue
		}

		// Check the store for a Merkle Proof for the current input.
		mp, err := c.mpc.MerkleProof(ctx, pTxID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get merkle proof for tx %s", pTxID)
		}
		if mp != nil {
			// If a Merkle Proof exists, build and return an spv.Envelope
			envelope.Parents[pTxID] = &Envelope{
				TxID:  pTxID,
				Proof: mp,
			}

			// Skip getting the tx data as we have everything we need for verifying the current tx.
			continue
		}

		// If no merkle proof was found for the input, build a *bt.Tx from its TxID and recursively
		// call this function building envelopes for inputs without proofs, until a parent with a
		// Merkle Proof is found.
		pTx, err := c.txc.Tx(ctx, pTxID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get tx %s", pTxID)
		}
		if pTx == nil {
			return nil, fmt.Errorf("could not find tx %s", pTxID)
		}

		pEnvelope, err := c.CreateEnvelope(ctx, pTx)
		if err != nil {
			return nil, err
		}

		envelope.Parents[pTxID] = pEnvelope
	}

	return envelope, nil
}
