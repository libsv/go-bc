package spv

import (
	"encoding/hex"
	"fmt"

	"github.com/libsv/go-bt/v2"
	"github.com/pkg/errors"

	"github.com/libsv/go-bc"
)

const (
	flagTx    = 1
	flagProof = 2
	flagMapi  = 3
)

// Envelope is a struct which contains all information needed for a transaction to be verified.
//
// spec at https://tsc.bitcoinassociation.net/standards/spv-envelope/
type Envelope struct {
	TxID          string               `json:"txid,omitempty"`
	RawTx         string               `json:"rawTx,omitempty"`
	Proof         *bc.MerkleProof      `json:"proof,omitempty"`
	MapiResponses []bc.MapiCallback    `json:"mapiResponses,omitempty"`
	Parents       map[string]*Envelope `json:"parents,omitempty"`
}

// IsAnchored returns true if the envelope is the anchor tx.
func (e *Envelope) IsAnchored() bool {
	return e.Proof != nil
}

// HasParents returns true if this envelope has immediate parents.
func (e *Envelope) HasParents() bool {
	return e.Parents != nil && len(e.Parents) > 0
}

// ParentTx will return a parent if found and convert the rawTx to a bt.TX, otherwise a ErrNotAllInputsSupplied error is returned.
func (e *Envelope) ParentTx(txID string) (*bt.Tx, error) {
	env, ok := e.Parents[txID]
	if !ok {
		return nil, errors.Wrapf(ErrNotAllInputsSupplied, "expected parent tx %s is missing", txID)
	}
	return bt.NewTxFromString(env.RawTx)
}

// Bytes takes an spvEnvelope struct and returns a pointer to the serialised bytes
func (e *Envelope) Bytes() *[]byte {
	flake := make([]byte, 0)

	// Binary format version 1
	flake = append(flake, 1)

	// currentTx
	currentTx, err := hex.DecodeString(e.RawTx)
	if err != nil {
		fmt.Print(err)
	}
	dataLength := bt.VarInt(uint64(len(currentTx)))

	// what is the next data going to be?
	flake = append(flake, flagTx)        // it's going to be a tx
	flake = append(flake, dataLength...) // it's going to be this long
	flake = append(flake, currentTx...)  // the tx data

	err = serialiseInputs(e.Parents, &flake)
	if err != nil {
		fmt.Println(err)
	}

	return &flake
}

// serialiseInputs is a recursive input serialiser for spv Envelopes
func serialiseInputs(parents map[string]*Envelope, flake *[]byte) error {
	for txid, input := range parents {
		fmt.Printf("%+v\n", txid)

		currentTx, err := hex.DecodeString(input.RawTx)
		if err != nil {
			fmt.Print(err)
		}
		dataLength := bt.VarInt(uint64(len(currentTx)))
		*flake = append(*flake, flagTx)        // first data will always be a rawTx
		*flake = append(*flake, dataLength...) // of this length
		*flake = append(*flake, currentTx...)  // the data
		if input.MapiResponses != nil && len(input.MapiResponses) > 0 {
			fmt.Print("implement mapi response serialisation") // TODO mapi response serialisation
		}
		if input.Proof != nil {
			proof, err := input.Proof.ToBytes()
			if err != nil {
				return errors.Wrap(err, "Failed to serialise this input's proof struct")
			}
			proofLength := bt.VarInt(uint64(len(proof)))
			*flake = append(*flake, flagProof)      // it's going to be a proof
			*flake = append(*flake, proofLength...) // of this length
			*flake = append(*flake, proof...)       // the data
		}
		if input.HasParents() {
			return serialiseInputs(input.Parents, flake)
		}
	}
	return nil
}

// NewEnvelopeFromBytes will encode an spv envelope byte slice
// into the Envelope structure.
//
func NewEnvelopeFromBytes(b []byte) (*Envelope, error) {
	var envelope Envelope
	var offset uint64 = 0

	// the first byte is the version number
	version := b[offset]
	if version != 1 {
		return nil, errors.New("We can only handle version 1 of the SPV Envelope Binary format")
	}
	offset++
	parseChunksRecursively(b, &offset, &envelope)
	return &envelope, nil
}

// ParseChunksRecursively will identify the next chunk of data's type and length,
// and pull out the stream into the appropriate struct
func parseChunksRecursively(b []byte, offset *uint64, eCurrent *Envelope) {
	typeOfNextData := b[*offset]
	*offset++
	l, size := bt.DecodeVarInt(b[*offset:])
	*offset += uint64(size)
	switch typeOfNextData {
	case flagTx:
		tx, err := bt.NewTxFromBytes(b[*offset : *offset+l])
		if err != nil {
			fmt.Println(err)
		}
		txid := tx.TxID()
		inputs := map[string]*Envelope{}
		for _, input := range tx.Inputs {
			inputs[hex.EncodeToString(input.PreviousTxID())] = &Envelope{}
		}
		eCurrent.TxID = txid
		eCurrent.RawTx = tx.String()
		howManyInputs := len(tx.Inputs)
		fmt.Println("txid", txid, " has ", howManyInputs, " inputs")
		fmt.Println("tx1", tx)
		*offset += l
		if uint64(len(b)) > *offset && b[*offset] != flagTx {
			parseChunksRecursively(b, offset, eCurrent)
		} else {
			eCurrent.Parents = inputs
		}
		for _, input := range inputs {
			if uint64(len(b)) > *offset {
				parseChunksRecursively(b, offset, input)
			}
		}
	case flagProof:
		binaryProof, err := parseBinaryMerkleProof(b[*offset : *offset+l])
		fmt.Println(hex.EncodeToString(b[*offset : *offset+l]))
		if err != nil {
			fmt.Println(err)
		}
		proof := bc.MerkleProof{
			Index:      binaryProof.index,
			TxOrID:     binaryProof.txOrID,
			Target:     binaryProof.target,
			Nodes:      binaryProof.nodes,
			TargetType: flagType(binaryProof.flags),
			// ignoring proofType and compositeType for this version
		}
		eCurrent.Proof = &proof
		*offset += l
	default:
		fmt.Printf("Unknown data type: %v, used for: %v", typeOfNextData, b[*offset:*offset+l])
		*offset += l
	}
	if uint64(len(b)) > *offset {
		parseChunksRecursively(b, offset, eCurrent)
	}
}

func flagType(flags byte) string {
	switch flags & targetTypeFlags {
	// if bits 1 and 2 of flags are NOT set, target should contain a block hash (32 bytes)
	// if bit 2 of flags is set, target should contain a merkle root (32 bytes)
	case 0, 4:
		return "blockhash"
	// if bit 1 of flags is set, target should contain a block header (80 bytes)
	case 2:
		return "header"
	default:
		return ""
	}
}
