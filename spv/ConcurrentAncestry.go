package spv

// import (
// 	"encoding/hex"
// 	"fmt"
// 	"sync"

// 	"github.com/libsv/go-bk/crypto"
// 	"github.com/libsv/go-bt/v2"
// 	"github.com/libsv/go-bt/v2/bscript"
// 	"github.com/pkg/errors"

// 	"github.com/libsv/go-bc"
// )

// const (
// 	flagTx    = byte(1)
// 	flagProof = byte(2)
// 	flagMapi  = byte(3)
// )

// // TxContext is a payment and its ancestors
// type TxContext struct {
// 	PaymentTx *PaymentTx
// 	Ancestors map[[32]byte]*Ancestor
// }

// // Ancestor is an internal struct for validating transactions with their ancestors.
// type Ancestor struct {
// 	Tx            *bt.Tx
// 	Proof         *bc.MerkleProof
// 	MapiResponses []*bc.MapiCallback
// 	RawTx         []byte
// 	RawProof      []byte
// 	RawMapi       []byte
// 	Parsed        chan bool
// 	Verified      chan bool
// }

// // PaymentTx is the root or current payment to be verified.
// type PaymentTx struct {
// 	RawTx []byte
// 	Tx    *bt.Tx
// }

// // BinaryChunk is a clear way to pass around chunks while keeping their type.
// type BinaryChunk struct {
// 	ContentType byte
// 	Data        []byte
// }

// // NewTxContextFromBytes creates a new struct from the bytes of a txContext
// func NewTxContextFromBytes(b []byte) TxContext {
// 	offset := uint64(1)
// 	total := uint64(len(b))

// 	l, size := bt.DecodeVarInt(b[offset:])
// 	offset += uint64(size)
// 	txContext := &TxContext{
// 		PaymentTx: &PaymentTx{
// 			RawTx: b[offset : offset+l],
// 		},
// 		Ancestors: make(map[[32]byte]*Ancestor),
// 	}
// 	offset += l

// 	var TxID [32]byte

// 	for total > offset {
// 		chunk := parseChunk(b, &offset)
// 		switch chunk.ContentType {
// 		case flagTx:
// 			hash := crypto.Sha256d(chunk.Data)
// 			copy(TxID[:], bt.ReverseBytes(hash)) // fixed size array from slice.
// 			txContext.Ancestors[TxID] = &Ancestor{
// 				RawTx:    chunk.Data,
// 				Parsed:   make(chan bool),
// 				Verified: make(chan bool),
// 			}
// 		case flagProof:
// 			txContext.Ancestors[TxID].RawProof = chunk.Data
// 		case flagMapi:
// 			txContext.Ancestors[TxID].RawMapi = chunk.Data
// 		default:
// 			continue
// 		}
// 	}
// 	return *txContext
// }

// func parseChunk(b []byte, offset *uint64) BinaryChunk {
// 	typeOfNextData := b[*offset]
// 	*offset++
// 	l, size := bt.DecodeVarInt(b[*offset:])
// 	*offset += uint64(size)
// 	chunk := BinaryChunk{
// 		ContentType: typeOfNextData,
// 		Data:        b[*offset : *offset+l],
// 	}
// 	*offset += l
// 	return chunk
// }

// func flagProofType(flags byte) string {
// 	switch flags & targetTypeFlags {
// 	// if bits 1 and 2 of flags are NOT set, target should contain a block hash (32 bytes).
// 	// if bit 2 of flags is set, target should contain a merkle root (32 bytes).
// 	case 0, 4:
// 		return "blockhash"
// 	// if bit 1 of flags is set, target should contain a block header (80 bytes).
// 	case 2:
// 		return "header"
// 	default:
// 		return ""
// 	}
// }

// func parseMapiCallbacks(b []byte) ([]*bc.MapiCallback, error) {
// 	if len(b) == 0 {
// 		return nil, errors.New("There are no callback bytes")
// 	}
// 	var internalOffset uint64
// 	allBinary := uint64(len(b))
// 	numOfMapiResponses := b[internalOffset]
// 	if numOfMapiResponses == 0 && len(b) == 1 {
// 		return nil, errors.New("There are no callbacks")
// 	}
// 	internalOffset++

// 	// split up the binary into flakes where each one is to be processed concurrently.
// 	var responses = [][]byte{}
// 	for ok := true; ok; ok = allBinary > internalOffset {
// 		l, size := bt.DecodeVarInt(b[internalOffset:])
// 		internalOffset += uint64(size)
// 		response := b[internalOffset : internalOffset+l]
// 		internalOffset += l
// 		responses = append(responses, response)
// 	}

// 	mapiResponses := make([]*bc.MapiCallback, 0)
// 	for _, response := range responses {
// 		mapiResponse, err := bc.NewMapiCallbackFromBytes(response)
// 		if err != nil {
// 			fmt.Println(err)
// 			return nil, errors.New("couldn't parse the callback bytes")
// 		}
// 		mapiResponses = append(mapiResponses, mapiResponse)
// 	}
// 	return mapiResponses, nil
// }

// // VerifyTxContextBinary will verify a slice of bytes which is a binary spv envelope.
// func VerifyTxContextBinary(binaryData []byte) (bool, error) {
// 	inputsVerified := &sync.WaitGroup{}
// 	if binaryData[0] != 1 { // the first byte is the version number.
// 		return false, errors.New("We can only handle version 1 of the SPV Envelope Binary format")
// 	}
// 	txContext := NewTxContextFromBytes(binaryData)

// 	// deal with the paymentTx first
// 	paymentTx, err := bt.NewTxFromBytes(txContext.PaymentTx.RawTx)
// 	if err != nil {
// 		return false, err
// 	}

// 	verified := false
// 	go checkEveryInput(paymentTx, &txContext, inputsVerified, &verified)

// 	// then all its ancestors concurrently.
// 	for txid, ancestor := range txContext.Ancestors {
// 		go parseAndVerify(txid, ancestor, &txContext)
// 	}

// 	inputsVerified.Wait()

// 	return true, nil
// }

// func checkEveryInput(tx *bt.Tx, txContext *TxContext, inputsVerified *sync.WaitGroup, result *bool) {
// 	inputsToCheck := make(map[[32]byte]*bt.Input)
// 	inputsParsed := &sync.WaitGroup{}
// 	for _, input := range tx.Inputs {
// 		inputsParsed.Add(1)
// 		inputsVerified.Add(1)
// 		var inputID [32]byte
// 		copy(inputID[:], input.PreviousTxID())
// 		inputsToCheck[inputID] = input

// 		parseListenChan := txContext.Ancestors[inputID].Parsed
// 		verifiedListenChan := txContext.Ancestors[inputID].Verified

// 		// we need to listen for each input to be parsed before verifying input output pairs.
// 		go func(parseListenChan <-chan bool) {
// 		parsed:
// 			for {
// 				select {
// 				case _, ok := <-parseListenChan:
// 					if !ok {
// 						defer inputsParsed.Done()
// 						break parsed
// 					}
// 				}
// 			}
// 		}(parseListenChan)

// 		// we also need to listen for the input transaction to be verified via some proof.
// 		go func(verifiedListenChan <-chan bool) {
// 		verified:
// 			for {
// 				select {
// 				case _, ok := <-verifiedListenChan:
// 					if !ok {
// 						defer inputsVerified.Done()
// 						break verified
// 					}
// 				}
// 			}
// 		}(verifiedListenChan)
// 	}
// 	// wait here until all inputs have been parsed.
// 	inputsParsed.Wait()

// 	verifications := 0
// 	for inputID, input := range inputsToCheck {
// 		lockingScript := txContext.Ancestors[inputID].Tx.Outputs[input.PreviousTxOutIndex].LockingScript
// 		unlockingScript := input.UnlockingScript
// 		if verifyInputOutputPair(tx, lockingScript, unlockingScript) {
// 			verifications++
// 		} else {
// 			fmt.Println("verifyInputOutputPair failed for: ", inputID)
// 		}
// 	}
// 	*result = verifications == len(inputsToCheck)
// }

// func parseAndVerify(txid [32]byte, ancestor *Ancestor, txContext *TxContext) {
// 	// parse the data for the transaction
// 	tx, err := bt.NewTxFromBytes(ancestor.RawTx)
// 	if err != nil {
// 		fmt.Println(hex.EncodeToString(bt.ReverseBytes(txid[:])), err)
// 	}
// 	ancestor.Tx = tx

// 	// parse the proof
// 	if ancestor.RawProof != nil {
// 		binaryProof, err := parseBinaryMerkleProof(ancestor.RawProof)
// 		if err != nil {
// 			fmt.Print(err)
// 		}
// 		ancestor.Proof = &bc.MerkleProof{
// 			Index:      binaryProof.index,
// 			TxOrID:     binaryProof.txOrID,
// 			Target:     binaryProof.target,
// 			Nodes:      binaryProof.nodes,
// 			TargetType: flagProofType(binaryProof.flags),
// 			// ignoring proofType and compositeType for this version.
// 		}
// 	}

// 	if ancestor.RawMapi != nil {
// 		mapiResponses, err := parseMapiCallbacks(ancestor.RawMapi)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		ancestor.MapiResponses = mapiResponses
// 	}

// 	close(ancestor.Parsed) // broadcast completion to all listeners

// 	// we are going to wait for parsing of all inputs, and verification of all inputs at some point.
// 	ancestorInputsVerified := &sync.WaitGroup{}

// 	// default to checked
// 	scriptsVerified := true

// 	// we will go through all the parents to this ancestor in the shrubbery.
// 	if ancestor.Proof == nil {
// 		checkEveryInput(ancestor.Tx, txContext, ancestorInputsVerified, &scriptsVerified)
// 	}

// 	// if proof, then verify it and mark self as Verified.
// 	proofVerified := true
// 	// if mapi, then verify it and mark self as Verified.
// 	mapiVerified := true

// 	// wait for the input leaves to parse, then check validity of the script pair.
// 	ancestorInputsVerified.Wait()

// 	// if verification passed on all of these.
// 	if scriptsVerified && proofVerified && mapiVerified {
// 		close(ancestor.Verified) // broadcast verified to all listeners
// 	}
// }

// func verifyInputOutputPair(tx *bt.Tx, lock *bscript.Script, unlock *bscript.Script) bool {
// 	// TODO script interpreter?
// 	return true
// }
