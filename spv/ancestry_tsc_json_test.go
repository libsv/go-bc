package spv_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/libsv/go-bc/spv"
	"github.com/libsv/go-bc/testing/data"
)

type TestData struct {
	PaymentTx string `json:"paymentTx,omitempty"`
	Ancestors string `json:"ancestors,omitempty"`
}

func TestAncestryBinaryToTSCJSON(t *testing.T) {
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
				require.NoError(t, err)
				require.NoError(t, json.NewDecoder(bytes.NewBuffer(jBinary)).Decode(&testData))

				binary, err := hex.DecodeString(testData.Ancestors)
				require.NoError(t, err, "expected no error when creating binary from hex")

				j, err := spv.NewAncestryJSONFromBytes(binary)
				require.NoError(t, err, "expected no error when transforming to json struct")

				_, err = json.Marshal(j)
				require.NoError(t, err, "expected no error when transforming to json bytes")
				require.NoError(t, err)
			}
		})
	}
}

func TestTSCAncestryJSONToBinary(t *testing.T) {
	tests := map[string]struct {
		testFile string
		// setupOpts are passed to the NewVerifier func.
		setupOpts []spv.VerifyOpt
		// overrideOpts are passed to the VerifyPayment func to override the global settings.
		overrideOpts []spv.VerifyOpt
		exp          bool
		paymentTx    string
		expErr       error
		expErrBinary error
	}{
		"three txs all using eachothers outputs": {
			exp:       true,
			paymentTx: "0100000002365830667fca2033d8d458eb719790c72420d712619f72719125aa7498e435be000000006b483045022100d480d3ebc7117c99984bdb97094a21357865f8dc6982e7d5f7da7a4412a4880f022066aacafeb37e3014d10cd12b0899fb4a924a7eccc79f918e255849f98d3fe8814121033bb801f4b3efd3671635e5761214fd16de5df8b99c238cdc8947210d259a8055ffffffff39fdfe30154bbed57497bb04c841b4849d925d20fa5aefd912548a96d295f528020000006a47304402201eb4b93d0e56e0bbcef736d83a540b1d5b1744cc1454a56f39cd186bcefb77a502202b09c15f54ac0666124660de94d20859084fae6c73bc716114db6659963a5aca4121033bb801f4b3efd3671635e5761214fd16de5df8b99c238cdc8947210d259a8055ffffffff02e8030000000000001976a91438870321a9b128965462eac78ef2c41490ebcc5788ac15070000000000001976a91438870321a9b128965462eac78ef2c41490ebcc5788ac00000000",
			testFile:  "3_serial",
		},
		"1000 txs all using eachothers outputs": {
			exp:       true,
			paymentTx: "01000000025e67889028fad6d3a4939a5358eacc67d042df40c35ac6d5809f9c9d2c55fa2b000000006b48304502210081592bc0c66e242b73a369f2f3c89d8af0540634e80d0ed152eab747476e290b02202fe77c97ea88ff40fc9c3a326c051c67392743b1c3fc011d32e8f12ae7640fe34121030c064fa8459d77ad1b4da374b30b02d7ed8422175bde8d0968b96522acfc8bf0ffffffffc2aaf7634bd6b441c384b90a93b19c6bcab4a9ddda5afb6be1b49e542d8d0399e70300006a47304402201680f7134900c5fed582578b945f912be4184d3c7f9ab5cd36d8319141f10b6b022039cec38752d34c2d5dda07919c7a5fc3b7a5759567054141a9d63723f9537e074121030c064fa8459d77ad1b4da374b30b02d7ed8422175bde8d0968b96522acfc8bf0ffffffff02e8030000000000001976a914bdb1b777c7017f08264d04843bc64f32461d654c88ac15070000000000001976a914bdb1b777c7017f08264d04843bc64f32461d654c88ac00000000",
			testFile:  "1000_serial",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.testFile != "" {
				testFile := &spv.TSCAncestriesJSON{}
				if test.testFile != "" {
					jBinary, err := data.SpvSerialJSONData.Load(test.testFile + ".json")
					require.NoError(t, err)
					require.NoError(t, json.NewDecoder(bytes.NewBuffer(jBinary)).Decode(&testFile))
				}
				_, err := testFile.Bytes()
				require.NoError(t, err, "expected no error when converting ancestry to bytes")

				// FIXME:
				// tx, err := bt.NewTxFromString(test.paymentTx)
				// require.NoError(t, err)
				// m, err := spv.NewMerkleProofVerifier(&mockBlockHeaderClient{})
				// require.NoError(t, err)
				// err = spv.VerifyAncestry(context.Background(), &spv.Payment{
				// 	PaymentTx: tx,
				// 	Ancestry:  a,
				// }, m)
				// require.NoError(t, err)
			}
		})
	}
}
