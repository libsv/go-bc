package spv

import (
	"context"
	"fmt"

	"github.com/libsv/go-bk/crypto"
	"github.com/libsv/go-bt/v2"
	"github.com/libsv/go-bt/v2/bscript"
	"github.com/pkg/errors"

	"github.com/libsv/go-bc"
)

const (
	flagTx    = byte(1)
	flagProof = byte(2)
	flagMapi  = byte(3)
)

// Ancestry is a payment and its ancestors.
type Ancestry struct {
	PaymentTx *bt.Tx
	Ancestors map[[32]byte]*Ancestor
}

// Ancestor is an internal struct for validating transactions with their ancestors.
type Ancestor struct {
	Tx            *bt.Tx
	Proof         []byte
	MapiResponses []*bc.MapiCallback
}

// binaryChunk is a clear way to pass around chunks while keeping their type.
type binaryChunk struct {
	ContentType byte
	Data        []byte
}

type extendedInput struct {
	input *bt.Input
	vin   int
}

// NewAncestryFromBytes creates a new struct from the bytes of a txContext.
func NewAncestryFromBytes(b []byte) *Ancestry {
	offset := uint64(1)
	total := uint64(len(b))

	l, size := bt.DecodeVarInt(b[offset:])
	offset += uint64(size)
	paymentTx, err := bt.NewTxFromBytes(b[offset : offset+l])
	if err != nil {
		panic(err)
	}
	ancestry := &Ancestry{
		PaymentTx: paymentTx,
		Ancestors: make(map[[32]byte]*Ancestor),
	}
	offset += l

	var TxID [32]byte

	for total > offset {
		chunk := parseChunk(b, &offset)
		switch chunk.ContentType {
		case flagTx:
			hash := crypto.Sha256d(chunk.Data)
			copy(TxID[:], bt.ReverseBytes(hash)) // fixed size array from slice.
			tx, err := bt.NewTxFromBytes(chunk.Data)
			if err != nil {
				panic(err)
			}
			ancestry.Ancestors[TxID] = &Ancestor{
				Tx: tx,
			}
		case flagProof:
			ancestry.Ancestors[TxID].Proof = chunk.Data
		case flagMapi:
			callBacks, err := parseMapiCallbacks(chunk.Data)
			if err != nil {
				panic(err)
			}
			ancestry.Ancestors[TxID].MapiResponses = callBacks
		default:
			continue
		}
	}
	return ancestry
}

func parseChunk(b []byte, offset *uint64) binaryChunk {
	typeOfNextData := b[*offset]
	*offset++
	l, size := bt.DecodeVarInt(b[*offset:])
	*offset += uint64(size)
	chunk := binaryChunk{
		ContentType: typeOfNextData,
		Data:        b[*offset : *offset+l],
	}
	*offset += l
	return chunk
}

func parseMapiCallbacks(b []byte) ([]*bc.MapiCallback, error) {
	if len(b) == 0 {
		return nil, errors.New("There are no callback bytes")
	}
	var internalOffset uint64
	allBinary := uint64(len(b))
	numOfMapiResponses := b[internalOffset]
	if numOfMapiResponses == 0 && len(b) == 1 {
		return nil, errors.New("There are no callbacks")
	}
	internalOffset++

	// split up the binary into flakes where each one is to be processed concurrently.
	var responses = [][]byte{}
	for ok := true; ok; ok = allBinary > internalOffset {
		l, size := bt.DecodeVarInt(b[internalOffset:])
		internalOffset += uint64(size)
		response := b[internalOffset : internalOffset+l]
		internalOffset += l
		responses = append(responses, response)
	}

	mapiResponses := make([]*bc.MapiCallback, 0)
	for _, response := range responses {
		mapiResponse, err := bc.NewMapiCallbackFromBytes(response)
		if err != nil {
			fmt.Println(err)
			return nil, errors.New("couldn't parse the callback bytes")
		}
		mapiResponses = append(mapiResponses, mapiResponse)
	}
	return mapiResponses, nil
}

// VerifyAncestryBinary will verify a slice of bytes which is a binary spv envelope.
func VerifyAncestryBinary(binaryData []byte, mpv MerkleProofVerifier, opts ...VerifyOpt) (bool, error) {
	o := &verifyOptions{
		proofs: true,
		script: true,
		fees:   false,
	}
	for _, opt := range opts {
		opt(o)
	}
	if binaryData[0] != 1 { // the first byte is the version number.
		return false, errors.New("We can only handle version 1 of the SPV Envelope Binary format")
	}
	ancestry := NewAncestryFromBytes(binaryData)
	err := VerifyAncestors(ancestry, mpv, o)
	if err != nil {
		return false, err
	}
	return true, nil
}

// VerifyAncestors will run through the map of Ancestors and check each input of each transaction to verify it.
// Only if there is no Proof attached.
func VerifyAncestors(ancestry *Ancestry, mpv MerkleProofVerifier, opts *verifyOptions) error {
	leaves := ancestry.Ancestors
	var paymentTxID [32]byte
	copy(paymentTxID[:], ancestry.PaymentTx.TxIDBytes())
	paymentLeaf := &Ancestor{
		Tx: ancestry.PaymentTx,
	}
	leaves[paymentTxID] = paymentLeaf
	for ancestorID, ancestor := range leaves {
		// if we have a proof, check it.
		if ancestor.Proof != nil && opts.proofs {
			// check proof.
			validProof, _, err := mpv.VerifyMerkleProof(context.Background(), ancestor.Proof)
			if err != nil || !validProof {
				return ErrInvalidProof
			}
		}
		inputsToCheck := make(map[[32]byte]*extendedInput)
		if opts.script || opts.fees {
			for idx, input := range ancestor.Tx.Inputs {
				var inputID [32]byte
				copy(inputID[:], input.PreviousTxID())
				inputsToCheck[inputID] = &extendedInput{
					input: input,
					vin:   idx,
				}
			}
		}
		if opts.script {
			// otherwise check the inputs.
			for inputID, extendedInput := range inputsToCheck {
				input := extendedInput.input
				// check if we have that ancestor, if not validation fail.
				if ancestry.Ancestors[inputID] == nil {
					if ancestor.Proof == nil && opts.proofs {
						return ErrProofOrInputMissing
					}
					continue
				}
				lockingScript := ancestry.Ancestors[inputID].Tx.Outputs[input.PreviousTxOutIndex].LockingScript
				unlockingScript := input.UnlockingScript
				if !verifyInputOutputPair(ancestor.Tx, lockingScript, unlockingScript) {
					fmt.Println("verifyInputOutputPair failed for: ", ancestorID, " - input: ", inputID)
					return ErrPaymentNotVerified
				}
			}
		}
		if opts.fees {
			if opts.feeQuote == nil {
				return ErrNoFeeQuoteSupplied
			}
			// no need to check fees for transactions we have proofs for
			if ancestor.Proof == nil {
				// add satoshi amounts to all inputs which correspond to outputs we have
				for inputID, extendedInput := range inputsToCheck {
					if ancestry.Ancestors[inputID] == nil {
						return ErrCannotCalculateFeePaid
					}
					sats := ancestry.Ancestors[inputID].Tx.Outputs[extendedInput.input.PreviousTxOutIndex].Satoshis
					ancestor.Tx.Inputs[extendedInput.vin].PreviousTxSatoshis = sats
				}
				// check the fees
				ok, err := ancestor.Tx.IsFeePaidEnough(opts.feeQuote)
				if err != nil || !ok {
					return ErrFeePaidNotEnough
				}
			}
		}
	}
	return nil
}

func verifyInputOutputPair(tx *bt.Tx, lock *bscript.Script, unlock *bscript.Script) bool {
	// TODO script interpreter.
	return true
}