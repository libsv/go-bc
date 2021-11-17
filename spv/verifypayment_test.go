package spv_test

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/libsv/go-bt/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/libsv/go-bc"
	"github.com/libsv/go-bc/spv"
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
		envelope        *spv.Envelope
		testFile        string
		blockHeaderFunc func(context.Context, string) (*bc.BlockHeader, error)
		// setupOpts are passed to the NewVerifier func.
		setupOpts []spv.VerifyOpt
		// overrideOpts are passed to the VerifyPayment func to override the global settings.
		overrideOpts []spv.VerifyOpt
		exp          bool
		expErr       error
	}{
		"valid envelope passes": {
			exp:      true,
			testFile: "valid",
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		},
		"envelope without any proof fails": {
			exp:      false,
			testFile: "invalid_missing_merkle_proof",
			expErr:   spv.ErrNoConfirmedTransaction,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		}, "envelope without any proof passes if proof disabled": {
			exp:      true,
			testFile: "invalid_missing_merkle_proof",
			expErr:   nil,
			setupOpts: []spv.VerifyOpt{
				spv.NoVerifyProofs(),
			},
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		}, "envelope without any proof passes if spv disabled": {
			exp:      true,
			expErr:   nil,
			testFile: "invalid_missing_merkle_proof",
			setupOpts: []spv.VerifyOpt{
				spv.NoVerifySPV(),
			},
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		}, "envelope without any proof passes if spv overridden": {
			exp:      true,
			testFile: "invalid_missing_merkle_proof",
			expErr:   nil,
			overrideOpts: []spv.VerifyOpt{
				spv.NoVerifyProofs(),
			},
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		},
		"valid envelope with merkle proof supplied as hex passes": {
			exp:      true,
			testFile: "valid_merkle_proof_hex",
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		}, "valid envelope with fee check supplied and valid fees passes": {
			exp:      true,
			testFile: "valid",
			overrideOpts: []spv.VerifyOpt{
				spv.VerifyFees(bt.NewFeeQuote()),
			},
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		}, "valid envelope with fee check supplied and invalid fees fails": {
			exp:      false,
			testFile: "valid",
			expErr:   spv.ErrFeePaidNotEnough,
			overrideOpts: []spv.VerifyOpt{
				spv.VerifyFees(bt.NewFeeQuote().AddQuote(bt.FeeTypeStandard, &bt.Fee{
					FeeType: bt.FeeTypeStandard,
					MiningFee: bt.FeeUnit{
						Satoshis: 10000000,
						Bytes:    1,
					},
				})),
			},
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		}, "envelope, no parents, no spv, fee check should fail": {
			exp:      false,
			testFile: "spv_no_parents",
			expErr:   spv.ErrCannotCalculateFeePaid,
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
			exp:      false,
			testFile: "invalid_merkle_proof",
			expErr:   spv.ErrInvalidProof,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		},
		"wrong tx supplied as input in envelope errs": {
			exp:      false,
			expErr:   spv.ErrNotAllInputsSupplied,
			testFile: "invalid_wrong_parent",
			blockHeaderFunc: func(context.Context, string) (*bc.BlockHeader, error) {
				return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
			},
		},
		"wrong merkle proof supplied with otherwise correct input errors": {
			exp:      false,
			testFile: "invalid_wrong_merkle_proof",
			expErr:   spv.ErrTxIDMismatch,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		},
		"wrong merkle proof supplied via hex with otherwise correct input errors": {
			exp:      false,
			testFile: "invalid_wrong_merkle_proof_hex",
			expErr:   spv.ErrTxIDMismatch,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		},
		"envelope with tx no inputs errs": {
			exp:      false,
			testFile: "invalid_tx_missing_inputs",
			expErr:   spv.ErrNoTxInputsToVerify,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		},
		"tx with input indexing out of bounds output errors": {
			exp:      false,
			testFile: "invalid_tx_indexing_oob",
			expErr:   spv.ErrInputRefsOutOfBoundsOutput,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				if blockHash == "4100429a6a29fd8ddf480f124f02557df39d9d58a671c9ea0a8f1fcc8ace923f" {
					return bc.NewBlockHeaderFromStr("0000002092df08285c865746bd933a0a97bda382cbc3ad1cbf7d3c8957c24e55eaba652dfc6f46aebb62fe9004ffa1e91b0ab37d1a865454a151e6011ce50751d33b40d7e1ef1361ffff7f2001000000")
				}
				return bc.NewBlockHeaderFromStr("000000203f92ce8acc1f8f0aeac971a6589d9df37d55024f120f48df8dfd296a9a4200413ca2ca1e79b3a8ff441a9d89feaa39b9771a30032a30fb023894ea4618395611f2ef1361ffff7f2000000000")
			},
		},
		"valid multiple layer tx passes": {
			exp:      true,
			testFile: "valid_deep",
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				switch blockHash {
				case "4f35d06cd4d00dcba92ade34b4c507c2939d3d1393f490a370c5f4239050dbcb":
					return bc.NewBlockHeaderFromStr("000000209f42742eb51d06c40a42b443888eca5030ca0dbae77e34e47b145c2255608a2d43d011ecd04a8989b4cae204bf1bc5ff15d87a62b356d899ca9d0361c946d671aaf61361ffff7f2000000000")
				case "730548cc946deba119fcee6ab2415bbb5fd8e0b41c9c0d5cae1ab069f905f56d":
					return bc.NewBlockHeaderFromStr("00000020ef6289f06cd618cf6eca2c94aaed8f4fed7948be527d1776c2216338b6ee940949d8b42d929d966f8e10ec2e47af5f87a39c5b09b9bac8ff6375ac9a8612614408f71361ffff7f2002000000")
				}
				return bc.NewBlockHeaderFromStr("000000208aef5325a07e4ec9cca864fca51e14d050d9fb9a371be6c651549580a0e33476414a38a7ddb819a4f3011cd06b17877968100a819348edb2009a60d0e0a65294fdf61361ffff7f2000000000")
			},
		},
		"invalid multiple layer tx false": {
			exp:      false,
			testFile: "invalid_deep_merkle_proof_index",
			expErr:   spv.ErrInvalidProof,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				switch blockHash {
				case "4f35d06cd4d00dcba92ade34b4c507c2939d3d1393f490a370c5f4239050dbcb":
					return bc.NewBlockHeaderFromStr("000000209f42742eb51d06c40a42b443888eca5030ca0dbae77e34e47b145c2255608a2d43d011ecd04a8989b4cae204bf1bc5ff15d87a62b356d899ca9d0361c946d671aaf61361ffff7f2000000000")
				case "730548cc946deba119fcee6ab2415bbb5fd8e0b41c9c0d5cae1ab069f905f56d":
					return bc.NewBlockHeaderFromStr("00000020ef6289f06cd618cf6eca2c94aaed8f4fed7948be527d1776c2216338b6ee940949d8b42d929d966f8e10ec2e47af5f87a39c5b09b9bac8ff6375ac9a8612614408f71361ffff7f2002000000")
				}
				return bc.NewBlockHeaderFromStr("000000208aef5325a07e4ec9cca864fca51e14d050d9fb9a371be6c651549580a0e33476414a38a7ddb819a4f3011cd06b17877968100a819348edb2009a60d0e0a65294fdf61361ffff7f2000000000")
			},
		},
		"tx with input missing from envelope parents errors": {
			exp:      false,
			testFile: "invalid_deep_parent_missing",
			expErr:   spv.ErrNotAllInputsSupplied,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				switch blockHash {
				case "4f35d06cd4d00dcba92ade34b4c507c2939d3d1393f490a370c5f4239050dbcb":
					return bc.NewBlockHeaderFromStr("000000209f42742eb51d06c40a42b443888eca5030ca0dbae77e34e47b145c2255608a2d43d011ecd04a8989b4cae204bf1bc5ff15d87a62b356d899ca9d0361c946d671aaf61361ffff7f2000000000")
				case "730548cc946deba119fcee6ab2415bbb5fd8e0b41c9c0d5cae1ab069f905f56d":
					return bc.NewBlockHeaderFromStr("00000020ef6289f06cd618cf6eca2c94aaed8f4fed7948be527d1776c2216338b6ee940949d8b42d929d966f8e10ec2e47af5f87a39c5b09b9bac8ff6375ac9a8612614408f71361ffff7f2002000000")
				}
				return bc.NewBlockHeaderFromStr("000000208aef5325a07e4ec9cca864fca51e14d050d9fb9a371be6c651549580a0e33476414a38a7ddb819a4f3011cd06b17877968100a819348edb2009a60d0e0a65294fdf61361ffff7f2000000000")
			},
		},
		"wrong merkle proof suppled with otherwise correct layered input errors": {
			exp:      false,
			testFile: "invalid_deep_wrong_merkle_proof",
			expErr:   spv.ErrTxIDMismatch,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				switch blockHash {
				case "4f35d06cd4d00dcba92ade34b4c507c2939d3d1393f490a370c5f4239050dbcb":
					return bc.NewBlockHeaderFromStr("000000209f42742eb51d06c40a42b443888eca5030ca0dbae77e34e47b145c2255608a2d43d011ecd04a8989b4cae204bf1bc5ff15d87a62b356d899ca9d0361c946d671aaf61361ffff7f2000000000")
				case "730548cc946deba119fcee6ab2415bbb5fd8e0b41c9c0d5cae1ab069f905f56d":
					return bc.NewBlockHeaderFromStr("00000020ef6289f06cd618cf6eca2c94aaed8f4fed7948be527d1776c2216338b6ee940949d8b42d929d966f8e10ec2e47af5f87a39c5b09b9bac8ff6375ac9a8612614408f71361ffff7f2002000000")
				}
				return bc.NewBlockHeaderFromStr("000000208aef5325a07e4ec9cca864fca51e14d050d9fb9a371be6c651549580a0e33476414a38a7ddb819a4f3011cd06b17877968100a819348edb2009a60d0e0a65294fdf61361ffff7f2000000000")
			},
		},
		"single missing merkle proof in layered and branching tx errors": {
			exp:      false,
			testFile: "invalid_deep_missing_merkle_proof",
			expErr:   spv.ErrNoConfirmedTransaction,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				switch blockHash {
				case "4f35d06cd4d00dcba92ade34b4c507c2939d3d1393f490a370c5f4239050dbcb":
					return bc.NewBlockHeaderFromStr("000000209f42742eb51d06c40a42b443888eca5030ca0dbae77e34e47b145c2255608a2d43d011ecd04a8989b4cae204bf1bc5ff15d87a62b356d899ca9d0361c946d671aaf61361ffff7f2000000000")
				case "730548cc946deba119fcee6ab2415bbb5fd8e0b41c9c0d5cae1ab069f905f56d":
					return bc.NewBlockHeaderFromStr("00000020ef6289f06cd618cf6eca2c94aaed8f4fed7948be527d1776c2216338b6ee940949d8b42d929d966f8e10ec2e47af5f87a39c5b09b9bac8ff6375ac9a8612614408f71361ffff7f2002000000")
				}
				return bc.NewBlockHeaderFromStr("000000208aef5325a07e4ec9cca864fca51e14d050d9fb9a371be6c651549580a0e33476414a38a7ddb819a4f3011cd06b17877968100a819348edb2009a60d0e0a65294fdf61361ffff7f2000000000")
			},
		},
		"tx with no inputs in multiple layer tx fails": {
			exp:      false,
			testFile: "invalid_deep_tx_missing_inputs",
			expErr:   spv.ErrNoTxInputsToVerify,
			blockHeaderFunc: func(ctx context.Context, blockHash string) (*bc.BlockHeader, error) {
				switch blockHash {
				case "4f35d06cd4d00dcba92ade34b4c507c2939d3d1393f490a370c5f4239050dbcb":
					return bc.NewBlockHeaderFromStr("000000209f42742eb51d06c40a42b443888eca5030ca0dbae77e34e47b145c2255608a2d43d011ecd04a8989b4cae204bf1bc5ff15d87a62b356d899ca9d0361c946d671aaf61361ffff7f2000000000")
				case "730548cc946deba119fcee6ab2415bbb5fd8e0b41c9c0d5cae1ab069f905f56d":
					return bc.NewBlockHeaderFromStr("00000020ef6289f06cd618cf6eca2c94aaed8f4fed7948be527d1776c2216338b6ee940949d8b42d929d966f8e10ec2e47af5f87a39c5b09b9bac8ff6375ac9a8612614408f71361ffff7f2002000000")
				}
				return bc.NewBlockHeaderFromStr("000000208aef5325a07e4ec9cca864fca51e14d050d9fb9a371be6c651549580a0e33476414a38a7ddb819a4f3011cd06b17877968100a819348edb2009a60d0e0a65294fdf61361ffff7f2000000000")
			},
		},
		"envelope with confirmed root errs": {
			exp:      false,
			testFile: "invalid_confirmed_root",
			expErr:   spv.ErrTipTxConfirmed,
			blockHeaderFunc: func(context.Context, string) (*bc.BlockHeader, error) {
				return bc.NewBlockHeaderFromStr("00000020f274078cebf6b61dd94b2124d9e967f7a7b9ccf0e95f46535768e333295b1e0633c974e51079022676c9319cd1cabcbf033282934f2d4fb4846ee6521d652e51fc680161ffff7f2000000000")
			},
			envelope: &spv.Envelope{
				TxID:  "06894e08c0e4137d70274c538351f5cea2e82011fafb3cc0192c74447dda19fd",
				RawTx: "0200000002f16ba9c4f21683b6840400418d4a0d27422e410e4cd398e4c64941363072ce5b000000006b4830450221009d2e7e89c0e0545ff0906cbc47060d0a74ee08948691180f59d9171ced24601a02202566505eaa97b4fb54830e33bb41a644e5d4c16b9d59ac1a61c45836da2961df412102b6dd19e32923d694ee510aa73e2eedf437783fce648b7b53effe31bfa6fee724feffffffb037e485154b5ae41f7cf229d519cd28b8d0f41f2f195309b8794cea95965116000000006a4730440220117995a5050437e1fd3866af61bb53f637fafcd051fcebcc9f40cc72cd40b395022036694fabae9720b03ecce1bf8d9d28e58123bfeb28a79814d95904e754c424634121028300e674b820a0f0df1c3399e9ef26dbca6ca1fdd9a4c53e5dbc964dcc6f2111feffffff0280ddf505000000001976a9149e5408eb250a1f9980ec735765dec14407c195ec88ac00ca9a3b000000001976a9146e8f17ecfc40ef5b429d22c86ffe8acb2acc886988acda000000",
				Proof: &bc.MerkleProof{
					Index:  1,
					TxOrID: "06894e08c0e4137d70274c538351f5cea2e82011fafb3cc0192c74447dda19fd",
					Target: "4f40da9ccedebb65ec7c29e4188ca11461668d7f2ae2e4e35b59b0fe4d266406",
					Nodes: []string{
						"00a43044caef87323a3ddee74dc7917e1dfd2371e9c43f208040cfe3737ee5ec",
					},
				},
			},
		},
		"nil initial payment errors": {
			exp:    false,
			expErr: spv.ErrNilInitialPayment,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var envelope *spv.Envelope
			if test.testFile != "" {
				f, err := os.Open(path.Join("../testing/data/spv/", test.testFile+".json"))
				assert.NoError(t, err)
				assert.NoError(t, json.NewDecoder(f).Decode(&envelope))
			}
			v, err := spv.NewPaymentVerifier(&mockBlockHeaderClient{
				blockHeaderFunc: test.blockHeaderFunc,
			}, test.setupOpts...)
			assert.NoError(t, err, "expected no error when creating spv client")

			tx, err := v.VerifyPayment(context.Background(), envelope, test.overrideOpts...)
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
		})
	}
}
