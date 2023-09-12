package spv

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/libsv/go-bt/v2"
	"github.com/pkg/errors"

	"github.com/libsv/go-bc"
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
		*flake = append(*flake, flagTx)                // first data will always be a rawTx.
		*flake = append(*flake, dataLength.Bytes()...) // of this length.
		*flake = append(*flake, currentTx...)          // the data.
		if input.MapiResponses != nil && len(input.MapiResponses) > 0 {
			for _, mapiResponse := range input.MapiResponses {
				mapiR, err := mapiResponse.Bytes()
				if err != nil {
					return err
				}
				dataLength := bt.VarInt(uint64(len(mapiR)))
				*flake = append(*flake, flagMapi)              // next data will be a mapi response.
				*flake = append(*flake, dataLength.Bytes()...) // of this length.
				*flake = append(*flake, mapiR...)              // the data.
			}
		}
		if input.Proof != nil {
			proof, err := input.Proof.Bytes()
			if err != nil {
				return errors.Wrap(err, "Failed to serialise this input's proof struct")
			}
			proofLength := bt.VarInt(uint64(len(proof)))
			*flake = append(*flake, flagProof)              // it's going to be a proof.
			*flake = append(*flake, proofLength.Bytes()...) // of this length.
			*flake = append(*flake, proof...)               // the data.
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
	l, size := bt.NewVarIntFromBytes(b[*offset:])
	*offset += uint64(size)
	switch typeOfNextData {
	case flagTx:
		tx, err := bt.NewTxFromBytes(b[*offset : *offset+uint64(l)])
		if err != nil {
			fmt.Println(err)
		}
		txid := tx.TxID()
		inputs := map[string]*Envelope{}
		for _, input := range tx.Inputs {
			inputs[input.PreviousTxIDStr()] = &Envelope{}
		}
		eCurrent.TxID = txid
		eCurrent.RawTx = tx.String()
		*offset += uint64(l)
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
		binaryProof, err := parseBinaryMerkleProof(b[*offset : *offset+uint64(l)])
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
		*offset += uint64(l)
	case flagMapi:
		mapiResponse, err := bc.NewMapiCallbackFromBytes(b[*offset : *offset+uint64(l)])
		if err != nil {
			fmt.Println(err)
		}
		if eCurrent.MapiResponses != nil {
			eCurrent.MapiResponses = append(eCurrent.MapiResponses, *mapiResponse)
		} else {
			eCurrent.MapiResponses = []bc.MapiCallback{*mapiResponse}
		}
		*offset += uint64(l)
	default:
		fmt.Printf("Unknown data type: %v, used for: %v", typeOfNextData, b[*offset:*offset+uint64(l)])
		*offset += uint64(l)
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
		return h
	default:
		return ""
	}
}

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
		*flake = append(*flake, dataLength.Bytes()...) // of this length.
		*flake = append(*flake, currentTx...)          // the data.

		// proof or zero
		if input.Proof == nil {
			*flake = append(*flake, 0)
		} else {
			proof, err := input.Proof.Bytes()
			if err != nil {
				return errors.Wrap(err, "Failed to serialise this input's proof struct")
			}
			proofLength := bt.VarInt(uint64(len(proof)))
			*flake = append(*flake, proofLength.Bytes()...) // of this length.
			*flake = append(*flake, proof...)               // the data.
		}

		if input.MapiResponses == nil || len(input.MapiResponses) == 0 {
			*flake = append(*flake, 0)
		} else {
			numOfMapiResponses := bt.VarInt(uint64(len(input.MapiResponses)))
			var mapiResponsesBinary []byte
			mapiResponsesBinary = append(mapiResponsesBinary, numOfMapiResponses.Bytes()...) // this many mapi responses follow
			for _, mapiResponse := range input.MapiResponses {
				mapiR, err := mapiResponse.Bytes()
				if err != nil {
					return err
				}
				dataLength := bt.VarInt(uint64(len(mapiR)))
				mapiResponsesBinary = append(mapiResponsesBinary, dataLength.Bytes()...) // of this length.
				mapiResponsesBinary = append(mapiResponsesBinary, mapiR...)              // the data.
			}
			fullDataLength := bt.VarInt(uint64(len(mapiResponsesBinary)))
			*flake = append(*flake, fullDataLength.Bytes()...)
			*flake = append(*flake, mapiResponsesBinary...)
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
		l, size := bt.NewVarIntFromBytes(b[offset:])
		offset += uint64(size)
		flake := b[offset : offset+uint64(l)]
		offset += uint64(l)
		flakes = append(flakes, flake)
	}

	mapiCallbackChan := make(chan []bc.MapiCallback)
	proofChan := make(chan *bc.MerkleProof)
	txChan := make(chan *bt.Tx)
	done := make(chan bool)

	txs := make(map[string]*bt.Tx)
	proofs := make(map[string]*bc.MerkleProof)
	mapiCallbacks := make(map[string][]bc.MapiCallback)

	wg := sync.WaitGroup{}

	// listen to these channels in perpetuity until we're done.
	go func() {
	L:
		for {
			select {
			case tx := <-txChan:
				txs[tx.TxID()] = tx
			case proof := <-proofChan:
				txid := proof.TxOrID
				if txid != "" {
					if len(txid) > 64 {
						tr, err := bt.NewTxFromString(txid)
						if err != nil {
							fmt.Println(err)
						}
						txid = tr.TxID()
					}
					proofs[txid] = proof
				}
			case mcbs := <-mapiCallbackChan:
				if len(mcbs) > 0 {
					txid := (mcbs)[0].CallbackTxID
					mapiCallbacks[txid] = mcbs
				}
			case <-done:
				break L
			}
		}
	}()

	for idx, flake := range flakes {
		// filter out the null values
		if len(flake) < 2 {
			continue
		}
		wg.Add(1)
		go func(flake []byte, idx int) {
			defer wg.Done()
			switch idx % 3 {
			case 2:
				mcb, err := parseSpecialKMapi(flake)
				if err != nil {
					fmt.Println(err)
				}
				mapiCallbackChan <- mcb
			case 1:
				proof, err := parseSpecialKProof(flake)
				if err != nil {
					fmt.Println(err)
				}
				proofChan <- proof
			case 0:
				tx, err := parseSpecialKFlakeTx(flake)
				if err != nil {
					fmt.Println(err)
				}
				txid := tx.TxID()
				if idx == 0 {
					envelope.TxID = txid
					envelope.RawTx = tx.String()
					inputs := make(map[string]*Envelope)
					for _, input := range tx.Inputs {
						inputs[input.PreviousTxIDStr()] = &Envelope{}
					}
					envelope.Parents = inputs
				}
				txChan <- tx
			}
		}(flake, idx)
	}

	wg.Wait()
	done <- true

	// construct something useful
	// iterate through all the transactions, addiong them to the struct's Parents
	// if they're in the struct delete them, if they're not then leave them, and go one input deep.
	// then run through again until they're all done.
	// // keep building struct unless all txs are in the struct.
	for ok := true; ok; ok = len(txs) > 0 {
		// iterate through txs
		for txid, tx := range txs {
			proof := proofs[txid]
			mapiCallback := mapiCallbacks[txid]
			_ = searchParents(&txs, &envelope, txid, tx, proof, mapiCallback)
		}
	}

	return &envelope, nil
}

func searchParents(txs *map[string]*bt.Tx, currentEnvelope *Envelope, txid string, tx *bt.Tx, p *bc.MerkleProof, m []bc.MapiCallback) bool {
	// is this the route transaction, and do we know it?
	if txid == currentEnvelope.TxID {
		currentEnvelope.RawTx = tx.String()
		if m != nil { // don't add proof unless it's a non empty struct.
			currentEnvelope.MapiResponses = m
		}
		if p != nil { // don't add proof unless it's a non empty struct.
			currentEnvelope.Proof = p
		}
		delete(*txs, txid)
		return true
	}
	// iterate through inputs
	for k := range currentEnvelope.Parents {
		// if we find the correct place, add the tx.
		if k == txid {
			var nextEnvelope Envelope
			nextEnvelope.TxID = txid
			nextEnvelope.RawTx = tx.String()
			if m != nil { // don't add proof unless it's a non empty struct.
				nextEnvelope.MapiResponses = m
			}
			if p != nil { // don't add proof unless it's a non empty struct.
				nextEnvelope.Proof = p
			} else {
				inputs := map[string]*Envelope{}
				for _, input := range tx.Inputs {
					inputs[input.PreviousTxIDStr()] = &Envelope{}
				}
				nextEnvelope.Parents = inputs
				currentEnvelope.Parents[txid] = &nextEnvelope
			}
			delete(*txs, txid)
			return true
		}
	}
	// if we didn't find it at this level, go one deeper into the struct to find a parent of a parent... etc.
	if currentEnvelope.Parents != nil && len(currentEnvelope.Parents) > 0 {
		for _, parent := range currentEnvelope.Parents {
			// this means the context of the next iteration through will be from the parent here, which is one step deeper.
			return searchParents(txs, parent, txid, tx, p, m)
		}
	}
	return false
}

// parseSpecialKFlakesRecursively will identify the next chunk of data's type and length,
// and pull out the stream into the appropriate struct.
func parseSpecialKFlakeTx(b []byte) (*bt.Tx, error) {
	if len(b) == 0 {
		return nil, errors.New("tx bytes have no length")
	}
	tx, err := bt.NewTxFromBytes(b)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return tx, nil
}

func parseSpecialKProof(b []byte) (*bc.MerkleProof, error) {
	if len(b) == 0 {
		return nil, errors.New("proof bytes have no length")
	}
	if b[0] == 0 && len(b) == 1 {
		return nil, errors.New("proof number is 0")
	}
	binaryProof, err := parseBinaryMerkleProof(b)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("couldn't parse the proof bytes")
	}
	proof := bc.MerkleProof{
		Index:      binaryProof.index,
		TxOrID:     binaryProof.txOrID,
		Target:     binaryProof.target,
		Nodes:      binaryProof.nodes,
		TargetType: flagType(binaryProof.flags),
		// ignoring proofType and compositeType for this version.
	}
	return &proof, nil
}

func parseSpecialKMapi(b []byte) ([]bc.MapiCallback, error) {
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
		l, size := bt.NewVarIntFromBytes(b[internalOffset:])
		internalOffset += uint64(size)
		response := b[internalOffset : internalOffset+uint64(l)]
		internalOffset += uint64(l)
		responses = append(responses, response)
	}

	mapiResponses := make([]bc.MapiCallback, 0)
	for _, response := range responses {
		mapiResponse, err := bc.NewMapiCallbackFromBytes(response)
		if err != nil {
			fmt.Println(err)
			return nil, errors.New("couldn't parse the callback bytes")
		}
		mapiResponses = append(mapiResponses, *mapiResponse)
	}
	return mapiResponses, nil
}
