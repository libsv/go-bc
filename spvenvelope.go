package bc

import (
	"context"
	"errors"
	"time"

	"github.com/libsv/go-bk/envelope"
	"github.com/libsv/go-bt"
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
	MapiResponses []envelope.JSONEnvelope `json:"mapiResponses,omitempty"`
	Inputs        map[string]*SPVEnvelope `json:"inputs"`
}

// MapiResponse is a callback from mApi
type MapiResponse struct {
	CallbackPayload MerkleProof `json:"callbackPayload"`
	APIVersion      string      `json:"apiVersion"`
	Timestamp       time.Time   `json:"timestamp"`
	MinerID         string      `json:"minerId"`
	BlockHash       string      `json:"blockHash"`
	BlockHeight     uint64      `json:"blockHeight"`
	CallbackTxID    string      `json:"callbackTxId"`
	CallbackReason  string      `json:"callbackReason"`
}

// VerifyPayment verifies whether or not the txs supplied via the supplied SPVEnvelope are valid
func (s *SPVClient) VerifyPayment(ctx context.Context, payment *SPVEnvelope) (bool, error) {
	proofs := make(map[string]bool)
	outputValues := make(map[string]uint64)

	tx, err := bt.NewTxFromString(payment.RawTX)
	if err != nil {
		return false, err
	}

	valid, err := s.verifyTxs(ctx, payment, nil, true, proofs)
	if err != nil {
		return false, err
	}
	if !valid {
		return valid, nil
	}

	for _, v := range proofs {
		if !v {
			return false, ErrPaymentNotVerified
		}
	}

	outputValues[tx.GetTxID()] = tx.GetTotalOutputSatoshis()

	return true, nil
}

func (s *SPVClient) verifyTxs(ctx context.Context, payment *SPVEnvelope, childInputs []*bt.Input,
	isRoot bool, proofs map[string]bool) (bool, error) {
	tx, err := bt.NewTxFromString(payment.RawTX)
	if err != nil {
		return false, err
	}
	txID := tx.GetTxID()
	proofs[txID] = false

	if isRoot && payment.Proof != nil {
		return false, ErrRootPaymentConfirmed
	}

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

	// cannot verify the proof or outputs of the root tx
	if isRoot {
		proofs[txID] = true
		return true, nil
	}

	// if at the leafs of tree and transaction is unconfirmed, fail
	if (payment.Inputs == nil || len(payment.Inputs) == 0) && payment.Proof == nil {
		return false, ErrNoConfirmedTransaction
	}

	//childTxInputMap := make(map[string]bool)
	if payment.Proof != nil {
		proofTxID := payment.Proof.TxOrID
		if len(proofTxID) != 64 {
			proofTx, err := bt.NewTxFromString(payment.Proof.TxOrID)
			if err != nil {
				return false, err
			}

			proofTxID = proofTx.GetTxID()
		}

		if proofTxID != payment.TxID {
			return false, ErrTxIDMismatch
		}

		var proofPresent bool
		for _, childInput := range childInputs {
			if childInput.PreviousTxID == proofTxID {
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

		proofs[txID] = valid

		return valid, nil
	}

	var pass bool
	for _, input := range childInputs {
		if input.PreviousTxID != txID {
			continue
		}
		pass = true

		// verify input and output
		output := tx.Outputs[int(input.PreviousTxOutIndex)]
		_ = output
	}

	if !pass {
		return false, ErrTxNotInInputs
	}

	proofs[txID] = true

	return true, nil
}
