package bc

import (
	"encoding/hex"

	"github.com/libsv/go-bk/crypto"
	"github.com/libsv/go-bt/v2"
	"github.com/libsv/go-p2p/chaincfg/chainhash"
)

// MerkleTreeParentStr returns the Merkle Tree parent of two Merkle
// Tree children using hex strings instead of just bytes.
func MerkleTreeParentStr(leftNode, rightNode string) (string, error) {
	l, err := hex.DecodeString(leftNode)
	if err != nil {
		return "", err
	}
	r, err := hex.DecodeString(rightNode)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(MerkleTreeParent(l, r)), nil
}

// MerkleTreeParent returns the Merkle Tree parent of two Merkle
// Tree children.
func MerkleTreeParent(leftNode, rightNode []byte) []byte {
	// swap endianness before concatenating
	l := bt.ReverseBytes(leftNode)
	r := bt.ReverseBytes(rightNode)

	// concatenate leaves
	concat := append(l, r...)

	// hash the concatenation
	hash := crypto.Sha256d(concat)

	// swap endianness at the end and convert to hex string
	return bt.ReverseBytes(hash)
}

// MerkleTreeParentBytes returns the Merkle Tree parent of two Merkle Tree children.
// The expectation is that the bytes are not reversed.
func MerkleTreeParentBytes(l *chainhash.Hash, r *chainhash.Hash) *chainhash.Hash {
	lb := l.CloneBytes()
	rb := r.CloneBytes()
	concat := append(lb, rb...)
	hash, err := chainhash.NewHash(crypto.Sha256d(concat))
	if err != nil {
		return &chainhash.Hash{}
	}
	return hash
}
