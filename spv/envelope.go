package spv

import (
	"encoding/hex"
	"fmt"
	"sync"

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

// Envelope is a struct which contains all information needed for a transaction to be verified.
//
type Envelope struct {
	TxID          string
	RawTx         string
	Proof         *bc.MerkleProof
	MapiResponses []bc.MapiCallback
	Parents       map[string]*Envelope
	Transactions  map[string]*Envelope
	Verified      bool
}

// SpvJSON spec at https://tsc.bitcoinassociation.net/standards/spv-envelope/
type SpvJSON struct {
	PaymentTx string `json:"root, omitempty"`
	Depth     uint64 `json:"depth, omitempty"`
	Txs       []struct {
		RawTx         string            `json:"hex,omitempty"`
		MapiResponses []bc.MapiCallback `json:"mapiResponses,omitempty"`
		Proof         *bc.MerkleProof   `json:"proof,omitempty"`
	} `json:"txs,omitempty"`
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

// Bytes takes an spvEnvelope struct and returns the serialised bytes.
func (e *Envelope) Bytes() ([]byte, error) {
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

	err := serialiseInputs(initialTx, &flake)
	if err != nil {
		fmt.Println(err)
	}
	return flake, nil
}

// serialiseInputs is a recursive input serialiser for spv Envelopes.
func serialiseInputs(parents map[string]*Envelope, flake *[]byte) error {
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
			return serialiseInputs(input.Parents, flake)
		}
	}
	return nil
}

type SpvBinaryChunk struct {
	ContentType byte
	Data        []byte
}

type SpvLeaf struct {
	RawTx         []byte
	Tx            *bt.Tx
	RawProof      []byte
	Proof         *bc.MerkleProof
	RawMapi       []byte
	MapiResponses []*bc.MapiCallback
	Parsed        chan bool
	Verified      chan bool
	Root          bool
}

type Shrubbery map[[256]byte]*SpvLeaf

// VerifyBinaryEnvelope will verify a slice of bytes which is a binary spv envelope
func VerifyBinaryEnvelope(binaryData []byte) (bool, error) {
	async := &sync.WaitGroup{}
	async.Add(1)
	valid := make(chan bool)
	if binaryData[0] != 1 { // the first byte is the version number.
		return false, errors.New("We can only handle version 1 of the SPV Envelope Binary format")
	}
	shrubbery := parseShrubbery(binaryData)
	for txid, leaf := range shrubbery {
		// for each leaf we need to parse the tx proof and mapi response data
		go func(txid [256]byte, leaf *SpvLeaf) {
			fmt.Printf("%v: %+v\n\n", txid, leaf)

			// parse the data for the transaction
			tx, err := bt.NewTxFromBytes(leaf.RawTx)
			if err != nil {
				fmt.Println(hex.EncodeToString(bt.ReverseBytes(txid[:])), err)
			}
			leaf.Tx = tx

			// parse the proof
			if leaf.RawProof != nil {
				binaryProof, err := parseBinaryMerkleProof(leaf.RawProof)
				if err != nil {
					fmt.Print(err)
				}
				leaf.Proof = &bc.MerkleProof{
					Index:      binaryProof.index,
					TxOrID:     binaryProof.txOrID,
					Target:     binaryProof.target,
					Nodes:      binaryProof.nodes,
					TargetType: flagType(binaryProof.flags),
					// ignoring proofType and compositeType for this version.
				}
			}

			if leaf.RawMapi != nil {
				mapiResponses, err := parseMapiCallbacks(leaf.RawMapi)
				if err != nil {
					fmt.Println(err)
				}
				leaf.MapiResponses = mapiResponses
			}

			close(leaf.Parsed) // broadcast completion to all listeners

			var inputsToCheck map[[256]byte]*bt.Input

			// we are going to wait for parsing of all inputs, and verification of all inputs at some point.
			leafInputsParsed := &sync.WaitGroup{}
			leafInputsVerified := &sync.WaitGroup{}

			// we will go through all the parents to this leaf in the shrubbery.
			for _, input := range tx.Inputs {
				leafInputsParsed.Add(1)
				leafInputsVerified.Add(1)
				var inputID [256]byte
				copy(inputID[:], input.PreviousTxID())
				inputsToCheck[inputID] = input

				// we need to listen for each input to be parsed before verifying input output pairs.
				go func(inputID [256]byte) {
				inputParsed:
					for {
						select {
						case _, ok := <-shrubbery[inputID].Parsed:
							if !ok {
								defer leafInputsParsed.Done()
								break inputParsed
							}
						}
					}
				}(inputID)

				// we also need to listen for the input transaction to be verified via some proof.
				go func(inputID [256]byte) {
				inputVerified:
					for {
						select {
						case _, ok := <-shrubbery[inputID].Verified:
							if !ok {
								defer leafInputsVerified.Done()
								break inputVerified
							}
						}
					}
				}(inputID)
			}

			// wait here until all inputs have been parsed.
			leafInputsParsed.Wait()

			verifications := 0
			for inputID, input := range inputsToCheck {
				lockingScript := shrubbery[inputID].Tx.Outputs[input.PreviousTxOutIndex].LockingScript
				unlockingScript := input.UnlockingScript
				if verifyInputOutputPair(tx, lockingScript, unlockingScript) {
					verifications++
				} else {
					fmt.Println("verifyInputOutputPair failed for: ", inputID)
				}
			}
			scriptsVerified := verifications == len(inputsToCheck)

			// if proof, then verify it and mark self as Verified
			proofVerified := true
			// if mapi, then verify it and mark self as Verified
			mapiVerified := true

			// wait for the input leaves to parse, then check validity of the script pair.
			leafInputsVerified.Wait()

			if scriptsVerified && proofVerified && mapiVerified {
				close(leaf.Verified) // broadcast verified to all listeners
				if leaf.Root {
					valid <- true
					async.Done()
				}
			}
			valid <- false
			async.Done()
		}(txid, leaf)
	}
	async.Wait()
	verification := <-valid
	return verification, nil
}

func verifyInputOutputPair(tx *bt.Tx, lock *bscript.Script, unlock *bscript.Script) bool {
	// TODO script interpreter?
	return true
}

func parseShrubbery(b []byte) Shrubbery {
	offset := uint64(1)
	total := uint64(len(b))
	shrubbery := make(Shrubbery)
	for total > offset {
		var TxID [256]byte
		root := offset == uint64(1)
		chunk := parseChunk(b, &offset)
		switch chunk.ContentType {
		case flagTx:
			hash := crypto.Sha256d(chunk.Data)
			copy(TxID[:], bt.ReverseBytes(hash)) // fixed size array from slice.
			shrubbery[TxID] = &SpvLeaf{RawTx: chunk.Data, Root: root}
		case flagProof:
			shrubbery[TxID].RawProof = chunk.Data
		case flagMapi:
			shrubbery[TxID].RawMapi = chunk.Data
		default:
			continue
		}
	}
	return shrubbery
}

func parseChunk(b []byte, offset *uint64) SpvBinaryChunk {
	typeOfNextData := b[*offset]
	*offset++
	l, size := bt.DecodeVarInt(b[*offset:])
	*offset += uint64(size)
	return SpvBinaryChunk{
		ContentType: typeOfNextData,
		Data:        b[*offset : *offset+l],
	}
}

// ParseChunksRecursively will identify the next chunk of data's type and length,
// and pull out the stream into the appropriate struct.
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
			// ignoring proofType and compositeType for this version.
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
