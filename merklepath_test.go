package bc_test

import (
	"testing"

	"github.com/libsv/go-bc"

	"github.com/stretchr/testify/assert"
)

func TestGetMerklePath(t *testing.T) {
	txids := []string{
		"b6d4d13aa08bb4b6cdb3b329cef29b5a5d55d85a85c330d56fddbce78d99c7d6",
		"426f65f6a6ce79c909e54d8959c874a767db3076e76031be70942b896cc64052",
		"adc23d36cc457d5847968c2e4d5f017a6f12a2f165102d10d2843f5276cfe68e",
		"728714bbbddd81a54cae473835ae99eb92ed78191327eb11a9d7494273dcad2a",
		"e3aa0230aa81abd483023886ad12790acf070e2a9f92d7f0ae3bebd90a904361",
		"4848b9e94dd0e4f3173ebd6982ae7eb6b793de305d8450624b1d86c02a5c61d9",
		"912f77eefdd311e24f96850ed8e701381fc4943327f9cf73f9c4dec0d93a056d",
		"397fe2ae4d1d24efcc868a02daae42d1b419289d9a1ded3a5fe771efcc1219d9",
	}

	expected := "1a1e779cd7dfc59f603b4e88842121001af822b2dc5d3b167ae66152e586a6b0"

	merkles, err := bc.BuildMerkleTreeStore(txids)
	assert.NoError(t, err)

	// build path for tx index 4
	path := bc.GetTxMerklePath(4, merkles)
	coin, err := bc.MerkleRootFromBranches("e3aa0230aa81abd483023886ad12790acf070e2a9f92d7f0ae3bebd90a904361", 4, path.Path)
	assert.NoError(t, err)
	assert.Equal(t, expected, coin)

	// build path for tx index 3
	path = bc.GetTxMerklePath(3, merkles)
	coin, err = bc.MerkleRootFromBranches("728714bbbddd81a54cae473835ae99eb92ed78191327eb11a9d7494273dcad2a", 3, path.Path)
	assert.NoError(t, err)
	assert.Equal(t, expected, coin)
}
