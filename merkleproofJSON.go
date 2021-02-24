package bc

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/libsv/go-bt"
)

// A MerkleProof is a structure that proves inclusion of a
// Bitcoin transaction in a block.
type MerkleProof struct {
	Index      uint64   `json:"index"`
	TxOrID     string   `json:"txOrId"`
	Target     string   `json:"target"`
	Nodes      []string `json:"nodes"`
	TargetType string   `json:"targetType,omitempty"`
	ProofType  string   `json:"proofType,omitempty"`
	Composite  bool     `json:"composite,omitempty"`
}

// ToBytes converts the JSON Merkle Proof
// into byte encoding.
func (mp *MerkleProof) ToBytes() ([]byte, error) {
	index := bt.VarInt(mp.Index)

	txOrID, err := hex.DecodeString(mp.TxOrID)
	if err != nil {
		return nil, err
	}
	txOrID = bt.ReverseBytes(txOrID)

	txLength := bt.VarInt(uint64(len(txOrID)))

	target, err := hex.DecodeString(mp.Target)
	if err != nil {
		return nil, err
	}
	target = bt.ReverseBytes(target)

	nodeCount := len(mp.Nodes)

	nodes := []byte{}

	for _, n := range mp.Nodes {
		if n == "*" {
			nodes = append(nodes, []byte{1}...)
			continue
		}

		nodes = append(nodes, []byte{0}...)
		nb, err := hex.DecodeString(n)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, bt.ReverseBytes(nb)...)

	}

	var flags uint8

	if len(mp.TxOrID) > 64 {
		// set bit at index 0
		flags |= (1 << 0)
	}

	if mp.TargetType == "header" {
		// set bit at index 1
		flags |= (1 << 1)
	} else if mp.TargetType == "merkleRoot" {
		// set bit at index 2
		flags |= (1 << 2)
	}

	// ignore proofType and compositeType for this version

	bytes := []byte{}
	bytes = append(bytes, flags)
	bytes = append(bytes, index...)
	bytes = append(bytes, txLength...)
	bytes = append(bytes, txOrID...)
	bytes = append(bytes, target...)
	bytes = append(bytes, byte(nodeCount))
	bytes = append(bytes, nodes...)

	return bytes, nil
}

// VerifyMerkleProofJSON verifies a Merkle Proof in standard JSON format.
func VerifyMerkleProofJSON(proof *MerkleProof) (bool, bool, error) {
	var txid string
	if len(proof.TxOrID) == 64 {
		// The `txOrId` field contains a transaction ID
		txid = proof.TxOrID
	} else if len(proof.TxOrID) > 64 {
		// The `txOrId` field contains a full transaction
		tx, err := bt.NewTxFromString(proof.TxOrID)
		if err != nil {
			return false, false, err
		}
		txid = tx.GetTxID()
	} else {
		return false, false, errors.New("invalid txOrId length - must be at least 64 chars (32 bytes)")
	}

	var merkleRoot string
	if proof.TargetType == "" || proof.TargetType == "hash" {
		// The `target` field contains a block hash

		if len(proof.Target) != 64 {
			return false, false, errors.New("invalid target field")
		}

		// You will need to get the block header corresponding
		// to this block hash in order to get the merkle root
		// from it. You can get this from from the headers
		// store of an SPV client or from a third party
		// provider like WhatsOnChain

		// TODO: make interface and do properly
		// const header = mapHashToHeader[proof.target]
		// if (!header) {
		// 	throw new Error('block hash map to header not found in `mapHashToHeader`')
		// }
		// merkleRoot = extractMerkleRootFromBlockHeader(header)
		merkleRoot = "96cbb75fd2ef98e4309eebc8a54d2386333d936ded2a0f3e06c23a91bb612f70"

	} else if proof.TargetType == "header" && len(proof.Target) == 160 {
		// The `target` field contains a block header
		var err error
		merkleRoot, err = ExtractMerkleRootFromBlockHeader(proof.Target)
		if err != nil {
			return false, false, err
		}

	} else if proof.TargetType == "merkleRoot" && len(proof.Target) == 64 {
		// the `target` field contains a merkle root
		merkleRoot = proof.Target

	} else {
		return false, false, errors.New("invalid TargetType or target field")
	}

	if proof.ProofType != "" && proof.ProofType != "branch" {
		return false, false, errors.New("only merkle branch supported in this version") // merkle tree proof type not supported
	}

	if proof.Composite { // OR if (proof.composite && proof.composite != false)
		return false, false, errors.New("only single proof supported in this version") // composite proof type not supported
	}

	if txid == "" {
		return false, false, errors.New("txid missing")
	}

	if merkleRoot == "" {
		return false, false, errors.New("merkleRoot missing")
	}

	nodes := proof.Nodes // different nodes used in the merkle proof
	index := proof.Index // index of node in current layer (will be changed on every iteration)
	c := txid            // first calculated node is the txid of the tx to prove
	isLastInTree := true

	for _, p := range nodes {
		// Check if the node is the left or the right child
		cIsLeft := index%2 == 0

		// Check for duplicate hash - this happens if the node (p) is
		// the last element of an uneven merkle tree layer
		if p == "*" {
			if !cIsLeft { // this shouldn't happen...
				return false, false, errors.New("invalid nodes")
			}
			p = c
		}

		// This check fails at least once if it's not the last element
		if cIsLeft && c != p {
			isLastInTree = false
		}

		var err error
		// Calculate the parent node
		if cIsLeft {
			// Concatenate left leaf (c) with right leaf (p)
			c, err = MerkleTreeParentStr(c, p)
			if err != nil {
				return false, false, err
			}
		} else {
			// Concatenate left leaf (p) with right leaf (c)
			c, err = MerkleTreeParentStr(p, c)
			if err != nil {
				return false, false, err
			}
		}

		// We need integer division here with remainder dropped.
		index = index / 2
	}

	// c is now the calculated merkle root
	return c == merkleRoot, isLastInTree, nil
}

// VerifyMerkleProof verifies a Merkle Proof in standard JSON format.
func VerifyMerkleProof(proof []byte) (valid, isLastInTree bool, err error) {

	mpb, err := ParseBinaryMerkleProof(proof)
	if err != nil {
		return false, false, err
	}

	var txid string
	// first calculated node is the txid of the tx to prove
	if len(mpb.txOrID) > 64 {
		tx, err := bt.NewTxFromString(mpb.txOrID)
		if err != nil {
			return false, false, err
		}

		txid = tx.GetTxID()

	} else if len(mpb.txOrID) == 64 {
		txid = mpb.txOrID

	} else {
		return false, false, errors.New("invalid txOrId length")
	}

	var merkleRoot string
	switch mpb.flags & (0x04 | 0x02) {
	// if bits 1 and 2 of flags are NOT set, target should contain a block hash (32 bytes)
	case 0:
		// The `target` field contains a block hash

		// You will need to get the block header corresponding
		// to this block hash in order to get the merkle root
		// from it. You can get this from from the headers
		// store of an SPV client or from a third party
		// provider like WhatsOnChain

		// TODO: make interface and do properly
		// const header = mapHashToHeader[proof.target]
		// if (!header) {
		// 	throw new Error('block hash map to header not found in `mapHashToHeader`')
		// }
		// merkleRoot = extractMerkleRootFromBlockHeader(header)
		merkleRoot = "96cbb75fd2ef98e4309eebc8a54d2386333d936ded2a0f3e06c23a91bb612f70"

	// if bit 2 of flags is set, target should contain a merkle root (32 bytes)
	case 4:
		// the `target` field contains a merkle root
		merkleRoot = mpb.target

	// if bit 1 of flags is set, target should contain a block header (80 bytes)
	case 2:
		// The `target` field contains a block header
		var err error
		merkleRoot, err = ExtractMerkleRootFromBlockHeader(mpb.target)
		if err != nil {
			return false, false, err
		}

	default:
		return false, false, errors.New("invalid flags")
	}

	// TODO: check flags for these types
	// if proof.ProofType != "" && proof.ProofType != "branch" {
	// 	return false, false, errors.New("only merkle branch supported in this version") // merkle tree proof type not supported
	// }

	// if proof.Composite { // OR if (proof.composite && proof.composite != false)
	// 	return false, false, errors.New("only single proof supported in this version") // composite proof type not supported
	// }

	if txid == "" {
		return false, false, errors.New("txid missing")
	}

	if merkleRoot == "" {
		return false, false, errors.New("merkleRoot missing")
	}

	nodes := mpb.nodes // different nodes used in the merkle proof
	index := mpb.index // index of node in current layer (will be changed on every iteration)
	c := txid          // first calculated node is the txid of the tx to prove

	isLastInTree = true

	for _, p := range nodes {
		// Check if the node is the left or the right child
		cIsLeft := index%2 == 0

		// Check for duplicate hash - this happens if the node (p) is
		// the last element of an uneven merkle tree layer
		if p == "*" {
			if !cIsLeft { // this shouldn't happen...
				return false, false, errors.New("invalid nodes")
			}
			p = c
		}

		// This check fails at least once if it's not the last element
		if cIsLeft && c != p {
			isLastInTree = false
		}

		var err error
		// Calculate the parent node
		if cIsLeft {
			// Concatenate left leaf (c) with right leaf (p)
			c, err = MerkleTreeParentStr(c, p)
			if err != nil {
				return false, false, err
			}
		} else {
			// Concatenate left leaf (p) with right leaf (c)
			c, err = MerkleTreeParentStr(p, c)
			if err != nil {
				return false, false, err
			}
		}

		// We need integer division here with remainder dropped.
		index = index / 2
	}

	// c is now the calculated merkle root
	return c == merkleRoot, isLastInTree, nil
}

type MerkleProofBinary struct {
	flags  byte
	index  uint64
	txOrID string
	target string
	nodes  []string
}

func ParseBinaryMerkleProof(proof []byte) (*MerkleProofBinary, error) {
	mpb := &MerkleProofBinary{}

	var offset, size int

	// flags is first byte
	mpb.flags = proof[offset]
	offset++

	// index is the next varint after the 1st byte
	mpb.index, size = bt.DecodeVarInt(proof[offset:])
	offset += size

	// txLength is the next varint after the 1st byte + index size
	txLength, size := bt.DecodeVarInt(proof[offset:])
	offset += size

	// if bit 1 of flags is NOT set, txOrId should contain txid (= 32 bytes)
	if mpb.flags&0 == 0 && txLength < 32 {
		return nil, errors.New("invalid tx length (should be greater than 32 bytes)")
	}

	// if bit 1 of flags is set, txOrId should contain tx hex (> 32 bytes)
	if mpb.flags&1 == 1 && txLength <= 32 {
		return nil, errors.New("invalid tx length (should be greater than 32 bytes)")
	}

	// txOrID is the next txLength bytes after 1st byte + index size + txLength size
	mpb.txOrID = hex.EncodeToString(bt.ReverseBytes(proof[offset : offset+int(txLength)]))
	offset += int(txLength)

	switch mpb.flags & (0x04 | 0x02) {
	// if bits 1 and 2 of flags are NOT set, target should contain a block hash (32 bytes)
	// if bit 2 of flags is set, target should contain a merkle root (32 bytes)
	case 0, 4:
		mpb.target = hex.EncodeToString(bt.ReverseBytes(proof[offset : offset+32]))
		offset += 32

	// if bit 1 of flags is set, target should contain a block header (80 bytes)
	case 2:
		mpb.target = hex.EncodeToString(bt.ReverseBytes(proof[offset : offset+80]))
		offset += 80

	default:
		return nil, errors.New("invalid flags")
	}

	nodeCount, size := bt.DecodeVarInt(proof[offset:])
	offset += size

	for i := 0; i < int(nodeCount); i++ {
		t := proof[offset]
		offset++

		var n string
		switch t {
		case 0:
			n = hex.EncodeToString(bt.ReverseBytes(proof[offset : offset+32]))
			offset += 32
		case 1:
			n = "*"

		default:
			return nil, fmt.Errorf("invalid value in node type at index: %q", i)
		}

		mpb.nodes = append(mpb.nodes, n)
	}

	return mpb, nil
}
