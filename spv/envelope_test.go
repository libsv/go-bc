package spv

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/libsv/go-bc"
)

func TestEnvelope_IsAnchored(t *testing.T) {
	tests := map[string]struct {
		envelope Envelope
		exp      bool
	}{
		"is anchored": {
			envelope: Envelope{
				Proof: &bc.MerkleProof{},
			},
			exp: true,
		},
		"is not anchored": {
			envelope: Envelope{},
			exp:      false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.exp, test.envelope.IsAnchored())
		})
	}
}
