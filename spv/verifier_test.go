package spv_test

import (
	"testing"

	"github.com/libsv/go-bc"
	"github.com/libsv/go-bc/spv"
	"github.com/libsv/go-bt/v2/bscript/interpreter"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockBScriptEngine struct {
	mockExecuteFunc func(interpreter.ExecutionParams) error
}

func (m *mockBScriptEngine) Execute(params interpreter.ExecutionParams) error {
	if m.mockExecuteFunc == nil {
		return errors.New("mockExecuteFunc not implemented for test")
	}

	return m.mockExecuteFunc(params)
}

func TestPaymentVerifier_NewPaymentVerifier(t *testing.T) {
	tests := map[string]struct {
		bhc    bc.BlockHeaderChain
		eng    interpreter.Engine
		expErr error
	}{
		"successful create": {
			bhc: &mockBlockHeaderClient{},
			eng: &mockBScriptEngine{},
		},
		"undefined bhc errors": {
			expErr: errors.New("at least one blockchain header implementation should be returned"),
		},
		"undefined eng errors": {
			bhc:    &mockBlockHeaderClient{},
			expErr: errors.New("at least one engine implementation should be provided"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := spv.NewPaymentVerifier(test.bhc, test.eng)
			if test.expErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.EqualError(t, err, test.expErr.Error())
			}
		})
	}
}

func TestMerkleProofVerifier_NewMerkleProofVerifier(t *testing.T) {
	tests := map[string]struct {
		bhc    bc.BlockHeaderChain
		expErr error
	}{
		"successful create": {
			bhc: &mockBlockHeaderClient{},
		},
		"undefined bhc errors": {
			expErr: errors.New("at least one blockchain header implementation should be returned"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := spv.NewMerkleProofVerifier(test.bhc)
			if test.expErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.EqualError(t, err, test.expErr.Error())
			}
		})
	}
}
