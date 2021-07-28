package bc

import (
	"context"
	"errors"
	"time"

	"github.com/libsv/go-bk/envelope"
	"github.com/libsv/go-bt"
)

// SPVEnvelope is a struct which contains all information needed
// for a transaction to be verified.
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

// VerifyPayment verifieds whether or not the supplied SPVEnvelope is valid
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
			return false, errors.New("payment was not verified")
		}
	}

	outputValues[tx.GetTxID()] = tx.GetTotalOutputSatoshis()

	return true, nil
}

func (s *SPVClient) verifyTxs(ctx context.Context, payment *SPVEnvelope, parentInputs []*bt.Input,
	isRoot bool, proofs map[string]bool) (bool, error) {
	tx, err := bt.NewTxFromString(payment.RawTX)
	if err != nil {
		return false, err
	}
	txID := tx.GetTxID()
	proofs[txID] = false

	if isRoot && payment.Proof != nil {
		return false, errors.New("root payment must be unconfirmed")
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
		return false, errors.New("no confirmed transaction provided")
	}

	parentInputsMap := make(map[string]bool)
	for _, parentInput := range parentInputs {
		parentInputsMap[parentInput.PreviousTxID] = true
	}

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
			return false, errors.New("input and proof id mismatch")
		}

		if _, ok := parentInputsMap[proofTxID]; !ok {
			return false, errors.New("proof for different tx supplied")
		}

		valid, _, err := s.VerifyMerkleProofJSON(ctx, payment.Proof)
		if err != nil {
			return false, err
		}

		proofs[txID] = valid

		return valid, nil
	}

	var pass bool
	for _, input := range parentInputs {
		if input.PreviousTxID != txID {
			continue
		}
		pass = true

		// verify input and output
		output := tx.Outputs[int(input.PreviousTxOutIndex)]
		_ = output
	}

	if !pass {
		return false, errors.New("could not find any inputs in tx")
	}

	proofs[txID] = true

	return true, nil
}
