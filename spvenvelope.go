package bc

import (
	"context"
	"errors"

	"github.com/libsv/go-bt"
	"github.com/tonicpow/go-minercraft"
)

var (
	// ErrPaymentNotVerified returns if a transaction in the tree provided was missed during verification
	ErrPaymentNotVerified = errors.New("a tx was missed during validation")

	// ErrRootPaymentConfirmed returns if the root payment is already confirmed
	ErrRootPaymentConfirmed = errors.New("root payment must be unconfirmed")

	// ErrNoConfirmedTransaction returns if a path from root to leaf contains no confirmed transcation
	ErrNoConfirmedTransaction = errors.New("not confirmed tx(s) provided")

	// ErrTxIDMismatch returns if they key value pair of a transactions input has a mismatch in txID
	ErrTxIDMismatch = errors.New("input and proof ID mismatch")

	// NotAllInputsSupplied returns if an unconfirmed transaction in envelope contains inputs which are not
	// present in the parent envelope
	ErrNotAllInputsSupplied = errors.New("a tx input missing in parent envelope")

	// ErrTxNotInInputs returns if the tx.Outputs of a transaction supplied in the SPV envelope cannot be
	// matched to any of its child transactions tx.Inputs
	ErrTxNotInInputs = errors.New("could not find tx in child inputs")
)

// SPVEnvelope is a struct which contains all information needed for a transaction to be verified.
//
// spec at https://tsc.bitcoinassociation.net/standards/spv-envelope/
type SPVEnvelope struct {
	TxID          string                  `json:"txid"`
	RawTX         string                  `json:"rawTx,omitempty"`
	Proof         *MerkleProof            `json:"proof,omitempty"`
	MapiResponses []minercraft.Callback   `json:"mapiResponses,omitempty"`
	Inputs        map[string]*SPVEnvelope `json:"inputs"`
}

// VerifyPayment verifies whether or not the txs supplied via the supplied SPVEnvelope are valid
func (s *SPVClient) VerifyPayment(ctx context.Context, payment *SPVEnvelope) (bool, error) {
	proofs := make(map[string]bool)

	valid, err := s.verifyTxs(ctx, payment, nil, true, proofs)
	if err != nil {
		return false, err
	}
	if !valid {
		return valid, nil
	}

	// Check the proofs map for safety, in case any tx was skipped during verification
	for _, v := range proofs {
		if !v {
			return false, ErrPaymentNotVerified
		}
	}

	return true, nil
}

func (s *SPVClient) verifyTxs(ctx context.Context, payment *SPVEnvelope, childTxInputs []*bt.Input,
	isRoot bool, proofs map[string]bool) (bool, error) {
	tx, err := bt.NewTxFromString(payment.RawTX)
	if err != nil {
		return false, err
	}
	txID := tx.GetTxID()
	proofs[txID] = false

	// The root tx is the transaction we're trying to verify, and it should not have a supplied
	// merkle proof.
	if isRoot && payment.Proof != nil {
		return false, ErrRootPaymentConfirmed
	}

	// Recurse to the leaves of the tree and verify upward towards the root. This way, we
	// check any merkle proofs provided first.
	for inputTxID, input := range payment.Inputs {
		if input.TxID == "" {
			input.TxID = inputTxID
		}
		valid, err := s.verifyTxs(ctx, input, tx.GetInputs(), false, proofs)
		if err != nil {
			return false, err
		}
		if !valid {
			return valid, nil
		}
	}

	// Given that the root of the SPVEnvelope is the tx we're trying to prove as legitimate,
	// it will not come with a proof (or any outputs) to verify.
	//
	// As well as this, for this condition to be true, every previous merkle proof or
	// tx verification will have passed. So, we can safely assume success and return true.
	if isRoot {
		proofs[txID] = true
		return true, nil
	}

	// If at the leaves of the tree and transaction is unconfirmed, fail and error.
	if (payment.Inputs == nil || len(payment.Inputs) == 0) && payment.Proof == nil {
		return false, ErrNoConfirmedTransaction
	}

	// Retrieve the index of the linked child tx input to prevent having to search the the inputs
	// slice multiple times later
	inputIdx, err := s.childTxInputIdx(txID, childTxInputs)
	if err != nil {
		return false, err
	}

	// If a merkle proof is provided, assume we are at the a leaf of the tree.
	// Verify and return the result.
	if payment.Proof != nil {
		return s.verifyLeafTx(ctx, payment, proofs)
	}

	// Ensure that every input current present with the current tx is present in the envelope
	// as a parent
	if ok := s.verifyAllTxInputsPresent(payment, tx); !ok {
		return false, ErrNotAllInputsSupplied
	}

	// If no merkle proof is provided, use the locking and unlocking scripts of this
	// and the child tx to verify legitimacy.
	return s.verifyUnconfirmedTx(txID, tx, childTxInputs[inputIdx], proofs)
}

func (S *SPVClient) verifyAllTxInputsPresent(payment *SPVEnvelope, tx *bt.Tx) bool {
	// If an unconfirmed tx has an input which is not present in the spv envelope, we
	// should fail and error, as we cannot prove the legitimacy of those inputs.
	for _, txInput := range tx.Inputs {
		if _, ok := payment.Inputs[txInput.PreviousTxID]; !ok {
			return false
		}
	}

	return true
}

func (s *SPVClient) childTxInputIdx(txID string, childTxInputs []*bt.Input) (int, error) {
	// Search through the child tx's inputs, and when a tx id match is found, return its index.
	for i, cTxInput := range childTxInputs {
		if cTxInput.PreviousTxID == txID {
			return i, nil
		}
	}

	// Otherwise, fail and error
	return 0, ErrTxNotInInputs
}

func (s *SPVClient) verifyLeafTx(ctx context.Context, payment *SPVEnvelope, proofs map[string]bool) (bool, error) {
	proofTxID := payment.Proof.TxOrID
	if len(proofTxID) != 64 {
		proofTx, err := bt.NewTxFromString(payment.Proof.TxOrID)
		if err != nil {
			return false, err
		}

		proofTxID = proofTx.GetTxID()
	}

	// If the tx id of the merkle proof doesn't match the tx id provided in the SPVEnvelope,
	// fail and error
	if proofTxID != payment.TxID {
		return false, ErrTxIDMismatch
	}

	valid, _, err := s.VerifyMerkleProofJSON(ctx, payment.Proof)
	if err != nil {
		return false, err
	}

	proofs[payment.TxID] = valid

	return valid, nil
}

func (s *SPVClient) verifyUnconfirmedTx(txID string, tx *bt.Tx, childTxInput *bt.Input,
	proofs map[string]bool) (bool, error) {
	// TODO: verify child tx input's unlocking script with current tx output's locking script
	output := tx.Outputs[int(childTxInput.PreviousTxOutIndex)]
	_ = output

	proofs[txID] = true

	return true, nil
}
