package spv

import (
	"encoding/hex"

	"github.com/libsv/go-bc"
	"github.com/libsv/go-bt/v2"
)

// AncestryJSON is a spec at https://tsc.bitcoinassociation.net/standards/spv-envelope/ eventually.
type AncestryJSON struct {
	PaymentTx string `json:"paymentTx,omitempty"`
	Ancestors []struct {
		RawTx         string            `json:"hex,omitempty"`
		Proof         *bc.MerkleProof   `json:"proof,omitempty"`
		MapiResponses []bc.MapiCallback `json:"mapiResponses,omitempty"`
	} `json:"ancestors,omitempty"`
}

// Bytes takes an AncestryJSON and returns the serialised bytes.
func (j *AncestryJSON) Bytes() ([]byte, error) {
	binaryTxContext := make([]byte, 0)

	// Binary format version 1.
	binaryTxContext = append(binaryTxContext, 1)

	// first tx is the payment.
	paymentTx, err := hex.DecodeString(j.PaymentTx)
	if err != nil {
		return nil, err
	}
	length := bt.VarInt(uint64(len(paymentTx)))
	binaryTxContext = append(binaryTxContext, length...)
	binaryTxContext = append(binaryTxContext, paymentTx...)

	// follow with the list of ancestors, including their proof or mapi responses if present.
	for _, ancestor := range j.Ancestors {
		rawTx, err := hex.DecodeString(ancestor.RawTx)
		if err != nil {
			return nil, err
		}
		length := bt.VarInt(uint64(len(rawTx)))
		binaryTxContext = append(binaryTxContext, flagTx)
		binaryTxContext = append(binaryTxContext, length...)
		binaryTxContext = append(binaryTxContext, rawTx...)
		if ancestor.Proof != nil {
			rawProof, err := ancestor.Proof.Bytes()
			if err != nil {
				return nil, err
			}
			length := bt.VarInt(uint64(len(rawProof)))
			binaryTxContext = append(binaryTxContext, flagProof)
			binaryTxContext = append(binaryTxContext, length...)
			binaryTxContext = append(binaryTxContext, rawProof...)
		}
		if ancestor.MapiResponses != nil && len(ancestor.MapiResponses) > 0 {
			binaryTxContext = append(binaryTxContext, flagMapi)
			numOfMapiResponses := bt.VarInt(uint64(len(ancestor.MapiResponses)))
			binaryTxContext = append(binaryTxContext, numOfMapiResponses...)
			for _, mapiResponse := range ancestor.MapiResponses {
				mapiR, err := mapiResponse.Bytes()
				if err != nil {
					return nil, err
				}
				dataLength := bt.VarInt(uint64(len(mapiR)))
				binaryTxContext = append(binaryTxContext, dataLength...)
				binaryTxContext = append(binaryTxContext, mapiR...)
			}
		}
	}

	return binaryTxContext, nil
}
