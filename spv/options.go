package spv

import "github.com/libsv/go-bc"

// SPVOpts can be implemented to provided functional options for an SPVClient.
type ClientOpts func(*spvclient)

// WithBlockHeaderChain will inject the provided BlockHeaderChain into the SPVClient.
func WithBlockHeaderChain(bhc bc.BlockHeaderChain) ClientOpts {
	return func(s *spvclient) {
		s.bhc = bhc
	}
}

func WithTXGetter(txg TXGetter) ClientOpts {
	return func(s *spvclient) {
		s.txg = txg
	}
}

func WithMerkleProofGetter(mpg MerkleProofGetter) ClientOpts {
	return func(s *spvclient) {
		s.mpg = mpg
	}
}
