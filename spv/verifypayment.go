package spv

import (
	"context"

	"github.com/libsv/go-bt/v2"
)

// VerifyPaymentWithAncestry is a method for parsing a binary payment transaction and its corresponding ancestry in binary.
// It will return the paymentTx struct if all validations pass.
func (v *verifier) VerifyPaymentWithAncestry(ctx context.Context, pTx []byte, ancestors []byte, opts ...VerifyOpt) (*bt.Tx, error) {
	o := &verifyOptions{
		proofs: true,
		script: true,
		fees:   false,
	}
	for _, opt := range opts {
		opt(o)
	}
	ancestry, err := NewAncestryFromBytes(ancestors)
	if err != nil {
		return nil, err
	}
	paymentTx, err := bt.NewTxFromBytes(pTx)
	if err != nil {
		return nil, err
	}
	ancestry.PaymentTx = paymentTx
	err = VerifyAncestors(ctx, ancestry, v, o)
	if err != nil {
		return nil, err
	}
	return paymentTx, nil
}
