package spv_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/libsv/go-bc/spv"
	"github.com/libsv/go-bc/testing/data"
)

func TestAncestryBinaryToJSON(t *testing.T) {
	tests := map[string]struct {
		testFile string
		// setupOpts are passed to the NewVerifier func.
		setupOpts []spv.VerifyOpt
		// overrideOpts are passed to the VerifyPayment func to override the global settings.
		overrideOpts []spv.VerifyOpt
		exp          bool
		expErr       error
		expErrBinary error
	}{
		"three txs all using eachothers outputs": {
			exp:      true,
			testFile: "valid_3_nested",
		},
		"1000 txs all using eachothers outputs": {
			exp:      true,
			testFile: "valid_1000_nested",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.testFile != "" {
				hexBinary, err := data.SpvBinaryData.Load(test.testFile + ".hex")
				assert.NoError(t, err)

				hexString := string(hexBinary)
				binary, err := hex.DecodeString(hexString)
				assert.NoError(t, err, "expected no error when creating binary from hex")

				j, err := spv.NewAncestoryJSONFromBytes(binary)
				assert.NoError(t, err, "expected no error when transforming to json struct")

				jsonBytes, err := json.Marshal(j)
				assert.NoError(t, err, "expected no error when transforming to json bytes")
				fmt.Println(string(jsonBytes))
				assert.NoError(t, err)
			}
		})
	}
}
