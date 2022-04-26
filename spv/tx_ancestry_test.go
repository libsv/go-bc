package spv

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/libsv/go-bc"
)

func TestEnvelope_IsAnchored(t *testing.T) {
	tests := map[string]struct {
		ancestry TxAncestry
		exp      bool
	}{
		"is anchored": {
			ancestry: TxAncestry{
				Proof: &bc.MerkleProof{},
			},
			exp: true,
		},
		"is not anchored": {
			ancestry: TxAncestry{},
			exp:      false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.exp, test.ancestry.IsAnchored())
		})
	}
}
