package bc

import "context"

// A BlockHeaderChain is a generic interface used to map things in the block header chain
// (chain of block headers). For example, it is used to get a block Header from a bitcoin
// block hash if it exists in the longest block header chain.
type BlockHeaderChain interface {
	BlockHeader(ctx context.Context, blockHash string) (blockHeader string, err error)
}
