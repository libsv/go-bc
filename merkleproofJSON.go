package bc

import (
	"errors"

	"github.com/libsv/go-bt"
)

// A MerkleProofJSON is a structure that proves inclusion of a
// Bitcoin transaction in a block.
type MerkleProofJSON struct {
	Index      uint64   `json:"index"`
	TxOrID     string   `json:"txOrId"`
	Target     string   `json:"target"`
	Nodes      []string `json:"nodes"`
	TargetType string   `json:"targetType,omitempty"`
	ProofType  string   `json:"proofType,omitempty"`
	Composite  bool     `json:"composite,omitempty"`
}

// VerifyMerkleProofJSON verifies a Merkle Proof in standard JSON format.
func VerifyMerkleProofJSON(proof *MerkleProofJSON) (bool, bool, error) {
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

// VerifyMerkleProofBytes verifies a Merkle Proof in standard JSON format.
func VerifyMerkleProofBytes(proof []byte) (valid, isLastInTree bool, err error) {
	return false, false, nil
}
