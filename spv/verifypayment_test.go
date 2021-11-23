package spv_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/libsv/go-bc"
	"github.com/libsv/go-bc/spv"
	"github.com/libsv/go-bc/testing/data"
	"github.com/libsv/go-bt/v2"
)

type mockBlockHeaderClient struct {
	blockHeaderFunc func(context.Context, string) (*bc.BlockHeader, error)
}

func (m *mockBlockHeaderClient) BlockHeader(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
	if m.blockHeaderFunc != nil {
		return m.blockHeaderFunc(ctx, blockHash)
	}

	return nil, errors.New("blockHeaderFunc in test is undefined")
}

func TestSPVEnvelope_VerifyPayment(t *testing.T) {
	tests := map[string]struct {
		testFile string
		// setupOpts are passed to the NewVerifier func.
		setupOpts []spv.VerifyOpt
		// overrideOpts are passed to the VerifyPayment func to override the global settings.
		overrideOpts []spv.VerifyOpt
		exp          bool
		expErr       error
		expSerialErr error
	}{
		"valid envelope passes": {
			exp:      true,
			testFile: "valid",
		},
		"envelope without any proof fails": {
			exp:          false,
			testFile:     "invalid_missing_merkle_proof",
			expErr:       spv.ErrNoConfirmedTransaction,
			expSerialErr: spv.ErrProofOrInputMissing,
		},
		"envelope without any proof passes if proof disabled": {
			exp:      true,
			testFile: "invalid_missing_merkle_proof",
			setupOpts: []spv.VerifyOpt{
				spv.NoVerifyProofs(),
			},
		},
		"envelope without any proof passes if spv disabled": {
			exp:      true,
			testFile: "invalid_missing_merkle_proof",
			setupOpts: []spv.VerifyOpt{
				spv.NoVerifySPV(),
			},
		},
		"envelope without any proof passes if spv overridden": {
			exp:      true,
			testFile: "invalid_missing_merkle_proof",
			overrideOpts: []spv.VerifyOpt{
				spv.NoVerifyProofs(),
			},
		},
		// "valid envelope with merkle proof supplied as hex passes": {
		// 	exp:      true,
		// 	testFile: "valid_merkle_proof_hex",
		// },
		"valid envelope with fee check supplied and valid fees passes": {
			exp:      true,
			testFile: "valid",
			overrideOpts: []spv.VerifyOpt{
				spv.VerifyFees(bt.NewFeeQuote()),
			},
		},
		"valid envelope with fee check supplied and invalid fees fails": {
			exp:          false,
			testFile:     "valid",
			expErr:       spv.ErrFeePaidNotEnough,
			expSerialErr: spv.ErrFeePaidNotEnough,
			overrideOpts: []spv.VerifyOpt{
				spv.VerifyFees(bt.NewFeeQuote().AddQuote(bt.FeeTypeStandard, &bt.Fee{
					FeeType: bt.FeeTypeStandard,
					MiningFee: bt.FeeUnit{
						Satoshis: 10000000,
						Bytes:    1,
					},
				})),
			},
		},
		"envelope, no parents, no spv, fee check should fail": {
			exp:          false,
			testFile:     "invalid_missing_parents",
			expErr:       spv.ErrCannotCalculateFeePaid,
			expSerialErr: spv.ErrCannotCalculateFeePaid,
			overrideOpts: []spv.VerifyOpt{
				spv.VerifyFees(bt.NewFeeQuote().AddQuote(bt.FeeTypeStandard, &bt.Fee{
					FeeType: bt.FeeTypeStandard,
					MiningFee: bt.FeeUnit{
						Satoshis: 0,
						Bytes:    10000,
					},
					RelayFee: bt.FeeUnit{},
				})),
				spv.NoVerifySPV(),
			},
		},
		"invalid merkle proof fails": {
			exp:          false,
			testFile:     "invalid_merkle_proof",
			expErr:       spv.ErrInvalidProof,
			expSerialErr: spv.ErrInvalidProof,
		},
		"wrong tx supplied as input in envelope errs": {
			exp:          false,
			expErr:       spv.ErrNotAllInputsSupplied,
			expSerialErr: spv.ErrProofOrInputMissing,
			testFile:     "invalid_wrong_parent",
		},
		"wrong merkle proof supplied with otherwise correct input errors": {
			exp:          false,
			testFile:     "invalid_wrong_merkle_proof",
			expErr:       spv.ErrTxIDMismatch,
			expSerialErr: spv.ErrInvalidProof,
		},
		"wrong merkle proof supplied via hex with otherwise correct input errors": {
			exp:          false,
			testFile:     "invalid_wrong_merkle_proof_hex",
			expErr:       spv.ErrTxIDMismatch,
			expSerialErr: spv.ErrInvalidProof,
		},
		// "envelope with tx no inputs errs": {
		// 	exp:          false,
		// 	testFile:     "invalid_tx_missing_inputs",
		// 	expErr:       spv.ErrNoTxInputsToVerify,
		// 	expSerialErr: spv.ErrNoTxInputsToVerify,
		// },
		// "tx with input indexing out of bounds output errors": {
		// 	exp:          false,
		// 	testFile:     "invalid_tx_indexing_oob",
		// 	expErr:       spv.ErrInputRefsOutOfBoundsOutput,
		// 	expSerialErr: spv.ErrInputRefsOutOfBoundsOutput,
		// },
		// "valid multiple layer tx passes": {
		// 	exp:      true,
		// 	testFile: "valid_deep",
		// },
		// "invalid multiple layer tx false": {
		// 	exp:          false,
		// 	testFile:     "invalid_deep_merkle_proof_index",
		// 	expErr:       spv.ErrInvalidProof,
		// 	expSerialErr: spv.ErrProofOrInputMissing,
		// },
		"tx with input missing from envelope parents errors": {
			exp:          false,
			testFile:     "invalid_deep_parent_missing",
			expErr:       spv.ErrNotAllInputsSupplied,
			expSerialErr: spv.ErrProofOrInputMissing,
		},
		// "wrong merkle proof suppled with otherwise correct layered input errors": {
		// 	exp:          false,
		// 	testFile:     "invalid_deep_wrong_merkle_proof",
		// 	expErr:       spv.ErrTxIDMismatch,
		// 	expSerialErr: spv.ErrProofOrInputMissing,
		// },
		// "single missing merkle proof in layered and branching tx errors": {
		// 	exp:          false,
		// 	testFile:     "invalid_deep_missing_merkle_proof",
		// 	expErr:       spv.ErrNoConfirmedTransaction,
		// 	expSerialErr: spv.ErrProofOrInputMissing,
		// },
		// "tx with no inputs in multiple layer tx fails": {
		// 	exp:          false,
		// 	testFile:     "invalid_deep_tx_missing_inputs",
		// 	expErr:       spv.ErrNoTxInputsToVerify,
		// 	expSerialErr: spv.ErrNoTxInputsToVerify,
		// },
		// "envelope with confirmed root errs": {
		// 	exp:          false,
		// 	testFile:     "invalid_confirmed_root",
		// 	expErr:       spv.ErrTipTxConfirmed,
		// 	expSerialErr: spv.ErrTipTxConfirmed,
		// },
		// "nil initial payment errors": {
		// 	exp:          false,
		// 	expErr:       spv.ErrNilInitialPayment,
		// 	expSerialErr: spv.ErrNilInitialPayment,
		// },
	}

	mch := &mockBlockHeaderClient{
		blockHeaderFunc: func(_ context.Context, hash string) (*bc.BlockHeader, error) {
			bb, err := data.BlockHeaderData.Load(hash)
			if err != nil {
				return nil, err
			}
			return bc.NewBlockHeaderFromStr(string(bb[:160]))
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testData := struct {
				Envelope *spv.Envelope `json:"data"`
			}{}
			if test.testFile != "" {
				bb, err := data.SpvVerifyData.Load(test.testFile + ".json")
				assert.NoError(t, err)
				assert.NoError(t, json.NewDecoder(bytes.NewBuffer(bb)).Decode(&testData))

				binary, err := envelope.Bytes()
				if err != nil {
					fmt.Println("seralisation didn't work. properly")
				}

				hex.EncodeToString(binary)
				fmt.Printf("%v binary:\n%v\n\n--------------------------------------\n\n\n", test.testFile, hex.EncodeToString(binary))

				//os.WriteFile(test.testFile+".txt", binary, os.FileMode("w"))

			}

			v, err := spv.NewPaymentVerifier(mch, test.setupOpts...)
			assert.NoError(t, err, "expected no error when creating spv client")

			tx, err := v.VerifyPayment(context.Background(), testData.Envelope, test.overrideOpts...)
			if test.expErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, errors.Cause(err), test.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			if test.exp {
				assert.NotNil(t, tx)
			} else {
				assert.Nil(t, tx)
			}

			binary, err := envelope.Bytes()
			assert.NoError(t, err, "expected no error when creating binary from json")

			mpv, err := spv.NewMerkleProofVerifier(mch)
			assert.NoError(t, err, "expected no error when creating binary from json")

			// ---------------- THE NEW FUNCTION ----------------
			fmt.Print("|===\t", name, "\t")
			opts := append(test.setupOpts, test.overrideOpts...)
			valid, err := spv.VerifyAncestryBinary(binary, mpv, opts...)
			if test.expErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, errors.Cause(err), test.expSerialErr.Error())
				assert.False(t, valid)
			} else {
				assert.NoError(t, err)
				assert.True(t, valid)
			}
			fmt.Println("✅ PASS")
		})
	}

}
