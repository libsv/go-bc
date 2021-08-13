package bc

import (
	"context"
	"errors"

	"github.com/libsv/go-bt"
)

var (
	// ErrPaymentNotVerified returns if a transaction in the tree provided was missed during verification
	ErrPaymentNotVerified = errors.New("a tx was missed during validation")

	// ErrTipTxConfirmed returns if the tip transaction is already confirmed
	ErrTipTxConfirmed = errors.New("tip transaction must be unconfirmed")

	// ErrNoConfirmedTransaction returns if a path from tip to beginning/anchor contains no confirmed transcation
	ErrNoConfirmedTransaction = errors.New("not confirmed/anchored tx(s) provided")

	// ErrTxIDMismatch returns if they key value pair of a transactions input has a mismatch in txID
	ErrTxIDMismatch = errors.New("input and proof ID mismatch")

	// ErrNotAllInputsSupplied returns if an unconfirmed transaction in envelope contains inputs which are not
	// present in the parent envelope
	ErrNotAllInputsSupplied = errors.New("a tx input missing in parent envelope")

	// ErrNoTxInputsToVerify returns if a transaction has no inputs
	ErrNoTxInputsToVerify = errors.New("a tx has no inputs to verify")

	// ErrNilInitialPayment returns if a transaction has no inputs
	ErrNilInitialPayment = errors.New("initial payment cannot be nil")

	// ErrInputRefsOutOfBoundsOutput returns if a transaction has no inputs
	ErrInputRefsOutOfBoundsOutput = errors.New("tx input index into output is out of bounds")
)

// SPVEnvelope is a struct which contains all information needed for a transaction to be verified.
//
// spec at https://tsc.bitcoinassociation.net/standards/spv-envelope/
type SPVEnvelope struct {
	TxID          string                  `json:"txid,omitempty"`
	RawTX         string                  `json:"rawTx,omitempty"`
	Proof         *MerkleProof            `json:"proof,omitempty"`
	MapiResponses []MapiCallback          `json:"mapiResponses,omitempty"`
	Parents       map[string]*SPVEnvelope `json:"parents,omitempty"`
}

// VerifyPayment verifies whether or not the txs supplied via the supplied SPVEnvelope are valid
func (s *SPVClient) VerifyPayment(ctx context.Context, initialPayment *SPVEnvelope) (bool, error) {
	if initialPayment == nil {
		return false, ErrNilInitialPayment
	}

	// The tip tx is the transaction we're trying to verify, and it should not have a supplied
	// Merkle Proof.
	if initialPayment.IsAnchored() {
		return false, ErrTipTxConfirmed
	}

	valid, err := s.verifyTxs(ctx, initialPayment)
	if err != nil {
		return false, err
	}

	return valid, nil
}

func (s *SPVClient) verifyTxs(ctx context.Context, payment *SPVEnvelope) (bool, error) {
	tx, err := bt.NewTxFromString(payment.RawTX)
	if err != nil {
		return false, err
	}

	// If at the beginning or middle of the tx chain and tx is unconfirmed, fail and error.
	if !payment.IsAnchored() && (payment.Parents == nil || len(payment.Parents) == 0) {
		return false, ErrNoConfirmedTransaction
	}

	// Recurse back to the anchor transactions of the transaction chain and verify forward towards
	// the tip transaction. This way, we check that the first transactions in the chain are anchored
	// to the blockchain through a valid Merkle Proof.
	for parentTxID, parent := range payment.Parents {
		if parent.TxID == "" {
			parent.TxID = parentTxID
		}

		valid, err := s.verifyTxs(ctx, parent)
		if err != nil {
			return false, err
		}
		if !valid {
			return false, nil
		}
	}

	// If a Merkle Proof is provided, assume we are at the anchor/beginning of the tx chain.
	// Verify and return the result.
	if payment.IsAnchored() {
		return s.verifyTxAnchor(ctx, payment)
	}

	// We must verify the tx or else we can not know if any of it's child txs are valid.
	return s.verifyUnconfirmedTx(tx, payment)
}

func (s *SPVClient) verifyTxAnchor(ctx context.Context, payment *SPVEnvelope) (bool, error) {
	proofTxID := payment.Proof.TxOrID
	if len(proofTxID) != 64 {
		proofTx, err := bt.NewTxFromString(payment.Proof.TxOrID)
		if err != nil {
			return false, err
		}

		proofTxID = proofTx.GetTxID()
	}

	// If the txid of the Merkle Proof doesn't match the txid provided in the SPVEnvelope,
	// fail and error
	if proofTxID != payment.TxID {
		return false, ErrTxIDMismatch
	}

	valid, _, err := s.VerifyMerkleProofJSON(ctx, payment.Proof)
	if err != nil {
		return false, err
	}

	return valid, nil
}

func (s *SPVClient) verifyUnconfirmedTx(tx *bt.Tx, payment *SPVEnvelope) (bool, error) {
	// If no tx inputs have been provided, fail and error
	if len(tx.Inputs) == 0 {
		return false, ErrNoTxInputsToVerify
	}

	for _, input := range tx.Inputs {
		parent, ok := payment.Parents[input.PreviousTxID]
		if !ok {
			return false, ErrNotAllInputsSupplied
		}

		parentTx, err := bt.NewTxFromString(parent.RawTX)
		if err != nil {
			return false, err
		}

		// If the input is indexing an output that is out of bounds, fail and error
		if int(input.PreviousTxOutIndex) > len(parentTx.Outputs)-1 {
			return false, ErrInputRefsOutOfBoundsOutput
		}

		output := parentTx.Outputs[int(input.PreviousTxOutIndex)]

		// TODO: verify script using input and previous output
		_ = output
	}

	return true, nil
}

// IsAnchored returns true if the envelope is the anchor tx.
func (s *SPVEnvelope) IsAnchored() bool {
	return s.Proof != nil
}
