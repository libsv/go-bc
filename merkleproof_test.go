package bc_test

import (
	"testing"

	"github.com/libsv/go-bc"
	"github.com/stretchr/testify/assert"
)

func TestVerifyMerkleProofBytes(t *testing.T) {
	proofJSON := &bc.MerkleProof{
		Index:  12,
		TxOrID: "ffeff11c25cde7c06d407490d81ef4d0db64aad6ab3d14393530701561a465ef",
		Target: "75edb0a69eb195cdd81e310553aa4d25e18450e08f168532a2c2e9cf447bf169",
		Nodes: []string{
			"b9ef07a62553ef8b0898a79c291b92c60f7932260888bde0dab2dd2610d8668e",
			"0fc1c12fb1b57b38140442927fbadb3d1e5a5039a5d6db355ea25486374f104d",
			"60b0e75dd5b8d48f2d069229f20399e07766dd651ceeed55ee3c040aa2812547",
			"c0d8dbda46366c2050b430a05508a3d96dc0ed55aea685bb3d9a993f8b97cc6f",
			"391e62b3419d8a943f7dbc7bddc90e30ec724c033000dc0c8872253c27b03a42",
		},
	}

	proof, _ := proofJSON.ToBytes()
	valid, _, err := bc.VerifyMerkleProof(proof)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestParseBinaryMerkleProof(t *testing.T) {
	// proofStr := "000c20ef65a4611570303539143dabd6aa64dbd0f41ed89074406dc0e7cd251cf1efff69f17b44cfe9c2a23285168fe05084e1254daa5305311ed8cd95b19ea6b0ed7505008e66d81026ddb2dae0bd88082632790fc6921b299ca798088bef5325a607efb9004d104f378654a25e35dbd6a539505a1e3ddbba7f92420414387bb5b12fc1c10f00472581a20a043cee55edee1c65dd6677e09903f22992062d8fd4b8d55de7b060006fcc978b3f999a3dbb85a6ae55edc06dd9a30855a030b450206c3646dadbd8c000423ab0273c2572880cdc0030034c72ec300ec9dd7bbc7d3f948a9d41b3621e39"
	// proof, _ := hex.DecodeString(proofStr)

	// flags, index, txOrId, target, nodes := bc.ParseBinaryMerkleProof(proof)
	// valid, _, err := bc.VerifyMerkleProof(proof)

	// assert.NoError(t, err)
	// assert.True(t, valid)
}
