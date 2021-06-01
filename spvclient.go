package bc

import (
	"context"
	"errors"
)

// An SPVClient is a struct used to specify interfaces
// used to complete Simple Payment Verification (SPV)
// in conjunction with a Merkle Proof.
//
// The implementation of BlockHeaderChain which is supplied will depend on the client
// you are using, some may return a HeaderJSON response others may return the blockhash.
type SPVClient struct {
	// BlockHeaderChain will be set when an implementation returning a bc.BlockHeader type is provided.
	bhc BlockHeaderChain
	// BlockHeaderChainStringer will be set when an implementation returning a block header hash is provided.
	bhchash BlockHeaderChainStringer
}

// SPVOpts can be implemented to provided functional options for an SPVClient.
type SPVOpts func(*SPVClient)

// WithBlockHeaderChain will inject the provided BlockHeaderChain into the SPVClient.
func WithBlockHeaderChain(bhc BlockHeaderChain) SPVOpts {
	return func(s *SPVClient) {
		s.bhc = bhc
	}
}

// WithBlockHeaderChainStringer will inject the provided BlockHeaderChainStringer into the SPVClient.
func WithBlockHeaderChainStringer(bhc BlockHeaderChainStringer) SPVOpts {
	return func(s *SPVClient) {
		s.bhchash = bhc
	}
}

// NewSPVClient creates a new SPVClient based on the options provided.
// If no BlockHeaderChain implementation is provided, the setup will return an error.
// If both a BlockHeaderChain AND a WithBlockHeaderChainStringer are provided it will
// attempt to use BlockHeaderChain first before falling back to the WithBlockHeaderChainStringer
// in the event of an error.
func NewSPVClient(opts ...SPVOpts) (*SPVClient, error) {
	cli := &SPVClient{}
	for _, opt := range opts {
		opt(cli)
	}
	if cli.bhc == nil && cli.bhchash == nil {
		return nil, errors.New("at least one blockchain header implementation should be returned")
	}
	return cli, nil
}

// BlockHeader will return the block header using the client implementation provided to the SPVClient.
func (spvc *SPVClient) BlockHeader(ctx context.Context, blockHash string) (*BlockHeader, error) {
	if spvc.bhc == nil && spvc.bhchash == nil {
		return nil, errors.New("no BlockHeaderChain implementation provided, setup the SPVClient with at least one")
	}
	// try the strong typed version first.
	var err error
	if spvc.bhc != nil {
		bh, e := spvc.bhc.BlockHeader(ctx, blockHash)
		if e == nil {
			return bh, nil
		}
		err = e
	}
	// if strong typed failed, or isn't set, try the hash version, which will then convert to a
	// strong typed bc.BlockHeader
	if spvc.bhchash != nil {
		bh, e := spvc.bhchash.BlockHeaderHash(ctx, blockHash)
		if e == nil {
			return EncodeBlockHeaderStr(bh)
		}
		err = e
	}
	return nil, err
}
