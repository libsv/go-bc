package spv

import (
	"encoding/hex"

	"github.com/libsv/go-bc"
	"github.com/libsv/go-bt/v2"
)

// AncestryJSON is a spec at https://tsc.bitcoinassociation.net/standards/spv-envelope/ eventually.
type AncestryJSON struct {
	PaymentTx string `json:"paymentTx,omitempty"`
	Depth     uint64 `json:"depth,omitempty"`
	Ancestors []struct {
		RawTx         string            `json:"hex,omitempty"`
		Proof         *bc.MerkleProof   `json:"proof,omitempty"`
		MapiResponses []bc.MapiCallback `json:"mapiResponses,omitempty"`
	} `json:"ancestors,omitempty"`
}

func AncestoryJSONFromBytes(b []byte) (*AncestryJSON, error) {
	ancestry, err := NewAncestryFromBytes(b)
	if err != nil {
		return nil, err
	}
	ancestors := make([]struct{
		RawTx         string
		Proof         *bc.MerkleProof
		MapiResponses []bc.MapiCallback
	})
	for ancestorID, ancestor := range ancestry.Ancestors {
		rawTx := ancestor.Tx.String()
		proof := &bc.MerkleProof{}
		mpb, err := parseBinaryMerkleProof(ancestor.Proof)
		if err != nil {
			return nil, err
		}
		if mpb.flags | targetTypeFlags == 0 {
			"header"
		}
		mp := &bc.MerkleProof{
			Index:      mpb.index,
			TxOrID:     mpb.txOrID,
			Target:     mpb.target,
			Nodes:      mpb.nodes,
			TargetType: target,
			ProofType:  "",
			Composite:  false,
		}
		a := struct{
			RawTx         string
			Proof         *bc.MerkleProof
			MapiResponses []bc.MapiCallback
		}{ 
			RawTx: rawTx, 
			Proof: &bc.MerkleProof{}, 
			MapiResponses: ancestor.MapiResponses
		}
		ancestors = append(ancestors, a)
	}
	j := &AncestryJSON{
		PaymentTx: ancestry.PaymentTx.String(),
		Depth:     0,
		Ancestors: ,
	}
	return j, nil
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
