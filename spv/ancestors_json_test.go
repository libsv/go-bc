package spv_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/libsv/go-bc/spv"
	"github.com/libsv/go-bc/testing/data"
)

type TestData struct {
	PaymentTx string `json:"paymentTx,omitempty"`
	Ancestors string `json:"ancestors,omitempty"`
}

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
				testData := &TestData{}
				jBinary, err := data.SpvBinaryData.Load(test.testFile + ".json")
				assert.NoError(t, err)
				assert.NoError(t, json.NewDecoder(bytes.NewBuffer(jBinary)).Decode(&testData))

				binary, err := hex.DecodeString(testData.Ancestors)
				assert.NoError(t, err, "expected no error when creating binary from hex")

				j, err := spv.NewAncestoryJSONFromBytes(binary)
				assert.NoError(t, err, "expected no error when transforming to json struct")

				_, err = json.Marshal(j)
				assert.NoError(t, err, "expected no error when transforming to json bytes")
				assert.NoError(t, err)
			}
		})
	}
}

func TestAncestryJSONToBinary(t *testing.T) {
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
			testFile: "3_serial",
		},
		"1000 txs all using eachothers outputs": {
			exp:      true,
			testFile: "1000_serial",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.testFile != "" {
				ancestry := spv.AncestryJSON{}
				if test.testFile != "" {
					jBinary, err := data.SpvSerialJSONData.Load(test.testFile + ".json")
					assert.NoError(t, err)
					assert.NoError(t, json.NewDecoder(bytes.NewBuffer(jBinary)).Decode(&ancestry))
				}

				_, err := ancestry.Bytes()
				assert.NoError(t, err, "expected no error when converting ancestry to bytes")
			}
		})
	}
}
