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

	// ErrProofTxMismatch returns if a proof (valid or not) is supplied for a transaction, but this proof
	//is for a transaction other than the one it was bundled with
	ErrProofTxMismatch = errors.New("proof tx id does not match input tx id")

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

	// If a merkle proof is provided, assume we are at the a leaf of the tree.
	// Verify and return the result.
	if payment.Proof != nil {
		return s.verifyLeafTx(ctx, payment, childTxInputs, proofs)
	}

	// If no merkle proof is provided, use the locking and unlocking scripts of this
	// and the child tx to verify legitimacy.
	return s.verifyUnconfirmedTx(txID, tx, childTxInputs, proofs)
}

func (s *SPVClient) verifyLeafTx(ctx context.Context, payment *SPVEnvelope, childTxInputs []*bt.Input,
	proofs map[string]bool) (bool, error) {
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

	// If the tx id of the merkle proof doesn't match any of the tx inputs of the child tx,
	// fail and error
	var proofPresent bool
	for _, cTxInput := range childTxInputs {
		if cTxInput.PreviousTxID == proofTxID {
			proofPresent = true
			break
		}
	}

	if !proofPresent {
		return false, ErrProofTxMismatch
	}

	valid, _, err := s.VerifyMerkleProofJSON(ctx, payment.Proof)
	if err != nil {
		return false, err
	}

	proofs[payment.TxID] = valid

	return valid, nil
}

func (s *SPVClient) verifyUnconfirmedTx(txID string, tx *bt.Tx, childTxInputs []*bt.Input,
	proofs map[string]bool) (bool, error) {
	// If current tx id is not found any tx input of the child tx, fail and error
	var pass bool
	for _, cTxInput := range childTxInputs {
		if cTxInput.PreviousTxID != txID {
			continue
		}
		pass = true

		// TODO: verify child tx input's unlocking script with current tx output's locking script
		output := tx.Outputs[int(cTxInput.PreviousTxOutIndex)]
		_ = output
	}

	if !pass {
		return pass, ErrTxNotInInputs
	}

	proofs[txID] = pass

	return pass, nil
}
