package spv_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/libsv/go-bc/spv"
)

func TestEnvelopeCreator_NewEnvelopeCreator(t *testing.T) {
	tests := map[string]struct {
		txc    spv.TxStore
		mpc    spv.MerkleProofStore
		expErr error
	}{
		"successful create": {
			txc: &mockTxMerkleGetter{},
			mpc: &mockTxMerkleGetter{},
		},
		"undefined txc errors": {
			mpc:    &mockTxMerkleGetter{},
			expErr: errors.New("an spv.TxStore implementation is required"),
		},
		"undefined mpc errors": {
			txc:    &mockTxMerkleGetter{},
			expErr: errors.New("an spv.MerkleProofStore implementation is required"),
		},
		"both stores undefined errors": {
			expErr: errors.New("an spv.TxStore implementation is required"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := spv.NewEnvelopeCreator(test.txc, test.mpc)
			if test.expErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.EqualError(t, err, test.expErr.Error())
			}
		})
	}
}
