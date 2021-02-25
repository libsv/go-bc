package bc

import "context"

// A MerkleRootGetter in a generic interface used to get a Merkle Root
// from a bitcoin block hash.
type MerkleRootGetter interface {
	MerkleRoot(ctx context.Context, blockHash string) (merkleRoot string, err error)
}
