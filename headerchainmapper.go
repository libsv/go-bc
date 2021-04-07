package bc

import "context"

// A HeaderChainMapper is a generic interface used to map things in the block header chain.
// For example, it is used to get a Merkle Root from a bitcoin block hash by mapping the
// block hash to the block header and extracting the Merkle Root from it.
type HeaderChainMapper interface {
	MerkleRoot(ctx context.Context, blockHash string) (merkleRoot string, err error)
}
