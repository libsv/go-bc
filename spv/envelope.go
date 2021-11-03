package spv

import (
	"encoding/hex"
	"fmt"
	"sync"

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

// ParentTX will return a parent if found and convert the rawTx to a bt.TX, otherwise a ErrNotAllInputsSupplied error is returned.
func (e *Envelope) ParentTX(txID string) (*bt.Tx, error) {
	env, ok := e.Parents[txID]
	if !ok {
		return nil, errors.Wrapf(ErrNotAllInputsSupplied, "expected parent tx %s is missing", txID)
	}
	return bt.NewTxFromString(env.RawTx)
}

// CrunchyNutBytes takes an spvEnvelope struct and returns a pointer to the serialised bytes.
func (e *Envelope) CrunchyNutBytes() (*[]byte, error) {
	flake := make([]byte, 0)

	// Binary format version 1
	flake = append(flake, 1)

	initialTx := map[string]*Envelope{
		e.TxID: {
			TxID:          e.TxID,
			RawTx:         e.RawTx,
			Proof:         e.Proof,
			MapiResponses: e.MapiResponses,
			Parents:       e.Parents,
		},
	}

	err := serialiseCrunchyNutInputs(initialTx, &flake)
	if err != nil {
		fmt.Println(err)
	}
	return &flake, nil
}

// serialiseCrunchyNutInputs is a recursive input serialiser for spv Envelopes.
func serialiseCrunchyNutInputs(parents map[string]*Envelope, flake *[]byte) error {
	for _, input := range parents {
		currentTx, err := hex.DecodeString(input.RawTx)
		if err != nil {
			fmt.Print(err)
		}
		dataLength := bt.VarInt(uint64(len(currentTx)))
		*flake = append(*flake, flagTx)        // first data will always be a rawTx.
		*flake = append(*flake, dataLength...) // of this length.
		*flake = append(*flake, currentTx...)  // the data.
		if input.MapiResponses != nil && len(input.MapiResponses) > 0 {
			for _, mapiResponse := range input.MapiResponses {
				mapiR, err := mapiResponse.Bytes()
				if err != nil {
					return err
				}
				dataLength := bt.VarInt(uint64(len(mapiR)))
				*flake = append(*flake, flagMapi)      // next data will be a mapi response.
				*flake = append(*flake, dataLength...) // of this length.
				*flake = append(*flake, mapiR...)      // the data.
			}
		}
		if input.Proof != nil {
			proof, err := input.Proof.ToBytes()
			if err != nil {
				return errors.Wrap(err, "Failed to serialise this input's proof struct")
			}
			proofLength := bt.VarInt(uint64(len(proof)))
			*flake = append(*flake, flagProof)      // it's going to be a proof.
			*flake = append(*flake, proofLength...) // of this length.
			*flake = append(*flake, proof...)       // the data.
		} else if input.HasParents() {
			return serialiseCrunchyNutInputs(input.Parents, flake)
		}
	}
	return nil
}

// NewCrunchyNutEnvelopeFromBytes will encode an spv envelope byte slice into the Envelope structure.
func NewCrunchyNutEnvelopeFromBytes(b []byte) (*Envelope, error) {
	var envelope Envelope
	var offset uint64

	// the first byte is the version number.
	version := b[offset]
	if version != 1 {
		return nil, errors.New("We can only handle version 1 of the SPV Envelope Binary format")
	}
	offset++
	parseCrunchyNutFlakesRecursively(b, &offset, &envelope)
	return &envelope, nil
}

// parseCrunchyNutChunksRecursively will identify the next chunk of data's type and length,
// and pull out the stream into the appropriate struct.
func parseCrunchyNutFlakesRecursively(b []byte, offset *uint64, eCurrent *Envelope) {
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
			parseCrunchyNutFlakesRecursively(b, offset, eCurrent)
		} else {
			eCurrent.Parents = inputs
		}
		for _, input := range inputs {
			if uint64(len(b)) > *offset {
				parseCrunchyNutFlakesRecursively(b, offset, input)
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
			// ignoring proofType and compositeType for this version.
		}
		eCurrent.Proof = &proof
		*offset += l
	case flagMapi:
		mapiResponse, err := bc.NewMapiCallbackFromBytes(b[*offset : *offset+l])
		if err != nil {
			fmt.Println(err)
		}
		if eCurrent.MapiResponses != nil {
			eCurrent.MapiResponses = append(eCurrent.MapiResponses, *mapiResponse)
		} else {
			eCurrent.MapiResponses = []bc.MapiCallback{*mapiResponse}
		}
		*offset += l
	default:
		fmt.Printf("Unknown data type: %v, used for: %v", typeOfNextData, b[*offset:*offset+l])
		*offset += l
	}
	if uint64(len(b)) > *offset {
		parseCrunchyNutFlakesRecursively(b, offset, eCurrent)
	}
}

func flagType(flags byte) string {
	switch flags & targetTypeFlags {
	// if bits 1 and 2 of flags are NOT set, target should contain a block hash (32 bytes).
	// if bit 2 of flags is set, target should contain a merkle root (32 bytes).
	case 0, 4:
		return "blockhash"
	// if bit 1 of flags is set, target should contain a block header (80 bytes).
	case 2:
		return "header"
	default:
		return ""
	}
}

/* From Mark Smith
txs := map[string]*bt.Tx{}
go func() {
	select {
	case tx <- txChan:
		txs[tx.TxID()] = tx
	case <-done:
		break
	}
}()

wg := sync.WaitGroup{}

for i := 0; i < len(txs);i++{
	wg.Add(1)
	go func(){
		defer wg.Done()
		txChan <- deserialise(blob)
	}
}

wg.Wait()
done <- struct{}{}
// iterate map and build tree

*/

// SpecialKBytes takes an spvEnvelope struct and returns a pointer to the serialised bytes.
func (e *Envelope) SpecialKBytes() (*[]byte, error) {
	flake := make([]byte, 0)

	// Binary format version 1
	flake = append(flake, 1)

	initialTx := map[string]*Envelope{
		e.TxID: {
			TxID:          e.TxID,
			RawTx:         e.RawTx,
			Proof:         e.Proof,
			MapiResponses: e.MapiResponses,
			Parents:       e.Parents,
		},
	}

	err := serialiseSpecialKInputs(initialTx, &flake)
	if err != nil {
		fmt.Println(err)
	}
	return &flake, nil
}

// serialiseSpecialKInputs is a recursive input serialiser for spv Envelopes.
func serialiseSpecialKInputs(parents map[string]*Envelope, flake *[]byte) error {
	for _, input := range parents {
		currentTx, err := hex.DecodeString(input.RawTx)
		if err != nil {
			fmt.Print(err)
		}
		// the transaction itself
		dataLength := bt.VarInt(uint64(len(currentTx)))
		*flake = append(*flake, dataLength...) // of this length.
		*flake = append(*flake, currentTx...)  // the data.

		// proof or zero
		if input.Proof == nil {
			*flake = append(*flake, bt.VarInt(0)...)
		} else {
			proof, err := input.Proof.ToBytes()
			if err != nil {
				return errors.Wrap(err, "Failed to serialise this input's proof struct")
			}
			proofLength := bt.VarInt(uint64(len(proof)))
			*flake = append(*flake, proofLength...) // of this length.
			*flake = append(*flake, proof...)       // the data.
		}

		if input.MapiResponses == nil || len(input.MapiResponses) == 0 {
			*flake = append(*flake, bt.VarInt(0)...)
		} else {
			for _, mapiResponse := range input.MapiResponses {
				mapiR, err := mapiResponse.Bytes()
				if err != nil {
					return err
				}
				dataLength := bt.VarInt(uint64(len(mapiR)))
				*flake = append(*flake, dataLength...) // of this length.
				*flake = append(*flake, mapiR...)      // the data.
			}
		}

		if input.Proof == nil && input.HasParents() {
			return serialiseSpecialKInputs(input.Parents, flake)
		}
	}
	return nil
}

// NewSpecialKEnvelopeFromBytes will encode an spv envelope byte slice into the Envelope structure.
func NewSpecialKEnvelopeFromBytes(b []byte) (*Envelope, error) {
	allBinary := uint64(len(b))
	var envelope Envelope
	var offset uint64

	// the first byte is the version number.
	version := b[offset]
	if version != 1 {
		return nil, errors.New("We can only handle version 1 of the SPV Envelope Binary format")
	}
	offset++

	// split up the binary into flakes where each one is to be processed concurrently.
	var flakes = [][]byte{}
	for ok := true; ok; ok = allBinary > offset {
		l, size := bt.DecodeVarInt(b[offset:])
		offset += uint64(size)
		flake := b[offset : offset+l]
		offset += l
		flakes = append(flakes, flake)
	}

	mapiCallbackChan := make(chan bc.MapiCallback)
	proofChan := make(chan bc.MerkleProof)
	txChan := make(chan bt.Tx)
	done := make(chan bool)
	txs := map[string]*bt.Tx{}
	go func() {
		select {
		case tx := <-txChan:
			txs[tx.TxID()] = &tx
		case <-done:
			break
		}
	}()

	wg := sync.WaitGroup{}

	for idx, flake := range flakes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch idx % 3 {
			case 2:
				mapiCallbackChan <- parseSpecialKMapi(flake)
			case 1:
				proofChan <- parseSpecialKProof(flake)
			case 0:
			default:
				txChan <- parseSpecialKFlake(flake)
			}
		}()
	}

	wg.Wait()
	done <- true

	// construct something useful

	return &envelope, nil
}

// parseSpecialKFlakesRecursively will identify the next chunk of data's type and length,
// and pull out the stream into the appropriate struct.
func parseSpecialKFlakeTx(b []byte) *bt.Tx {
	tx, err := bt.NewTxFromBytes(b)
	if err != nil {
		fmt.Println(err)
		return &bt.Tx{}
	}
	return tx
}

func parseSpecialKFlake(b []byte) bt.Tx {
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

	// size of next blob of data (proof)
	l, size = bt.DecodeVarInt(b[*offset:])
	*offset += uint64(size)
	if l != 0 {
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
			// ignoring proofType and compositeType for this version.
		}
		eCurrent.Proof = &proof
		*offset += l
	}

	// number of mapiCallbacks
	l, size = bt.DecodeVarInt(b[*offset:])
	*offset += uint64(size)
	if l != 0 {
		for i := uint64(0); i < l; i++ {
			// size of next blob of data (mapiCallback)
			l, size = bt.DecodeVarInt(b[*offset:])
			*offset += uint64(size)
			mapiResponse, err := bc.NewMapiCallbackFromBytes(b[*offset : *offset+l])
			if err != nil {
				fmt.Println(err)
			}
			if eCurrent.MapiResponses != nil {
				eCurrent.MapiResponses = append(eCurrent.MapiResponses, *mapiResponse)
			} else {
				eCurrent.MapiResponses = []bc.MapiCallback{*mapiResponse}
			}
			*offset += l
		}
	}

	if uint64(len(b)) > *offset && b[*offset] != flagTx {
		parseSpecialKFlakesRecursively(b, offset, eCurrent)
	} else {
		eCurrent.Parents = inputs
	}
	for _, input := range inputs {
		if uint64(len(b)) > *offset {
			parseSpecialKFlakesRecursively(b, offset, input)
		}
	}

	if uint64(len(b)) > *offset {
		parseSpecialKFlakesRecursively(b, offset, eCurrent)
	}
}
