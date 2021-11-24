package spv_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/libsv/go-bc/spv"
	"github.com/libsv/go-bc/testing/data"
	"github.com/stretchr/testify/assert"
)

func Test_BinaryEnvelopeVerification(t *testing.T) {
	h, err := data.SpvCreateBinaryData.Load("valid_txContext_hex.txt")
	if err != nil {
		fmt.Println(err)
	}
	binary, err := hex.DecodeString(string(h))
	if err != nil {
		fmt.Println(err)
	}
	valid, err := spv.VerifyTxContextBinary(binary)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(valid, hex.EncodeToString(binary))
	assert.NoError(t, err)
}
