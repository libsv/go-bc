package bc_test

import (
	"encoding/hex"
	"testing"

	"github.com/libsv/go-bc"

	"github.com/stretchr/testify/assert"
)

func TestBuildingMerklePathBinary(t *testing.T) {
	t.Parallel()

	// build example merkle path data
	merklePath := bc.MerklePath{
		Index: 136,
		Path: []string{"6cf512411d03ab9b61643515e7aa9afd005bf29e1052ade95410b3475f02820c",
			"cd73c0c6bb645581816fa960fd2f1636062fcbf23cb57981074ab8d708a76e3b",
			"b4c8d919190a090e77b73ffcd52b85babaaeeb62da000473102aca7f070facef",
			"3470d882cf556a4b943639eba15dc795dffdbebdc98b9a98e3637fda96e3811e"},
	}

	// build binary path from it
	merklePathBinary, err := merklePath.Bytes()
	if err != nil {
		t.Error(err)
		return
	}

	mp, _ := hex.DecodeString("88040c82025f47b31054e9ad52109ef25b00fd9aaae7153564619bab031d4112f56c3b6ea708d7b84a078179b53cf2cb2f0636162ffd60a96f81815564bbc6c073cdefac0f077fca2a10730400da62ebaebaba852bd5fc3fb7770e090a1919d9c8b41e81e396da7f63e3989a8bc9bdbefddf95c75da1eb3936944b6a55cf82d87034")
	if err != nil {
		t.Error(err)
		return
	}

	// assert binary path is expected
	assert.Equal(t, mp, merklePathBinary)
}

func TestDecodingMerklePathBinary(t *testing.T) {
	t.Parallel()

	merklePath, err := bc.NewMerklePathFromStr("88040c82025f47b31054e9ad52109ef25b00fd9aaae7153564619bab031d4112f56c3b6ea708d7b84a078179b53cf2cb2f0636162ffd60a96f81815564bbc6c073cdefac0f077fca2a10730400da62ebaebaba852bd5fc3fb7770e090a1919d9c8b41e81e396da7f63e3989a8bc9bdbefddf95c75da1eb3936944b6a55cf82d87034")
	if err != nil {
		t.Error(err)
		return
	}

	// data we are expecting to deserialize
	// merklePathData := bc.MerklePathData{
	// 	Index: 136,
	// 	Path: []string{"6cf512411d03ab9b61643515e7aa9afd005bf29e1052ade95410b3475f02820c",
	// 		"cd73c0c6bb645581816fa960fd2f1636062fcbf23cb57981074ab8d708a76e3b",
	// 		"b4c8d919190a090e77b73ffcd52b85babaaeeb62da000473102aca7f070facef",
	// 		"3470d882cf556a4b943639eba15dc795dffdbebdc98b9a98e3637fda96e3811e"},
	// }

	// assert binary path is expected
	assert.Equal(t, uint64(136), merklePath.Index)
	assert.Equal(t, 4, len(merklePath.Path))
	assert.Equal(t, "6cf512411d03ab9b61643515e7aa9afd005bf29e1052ade95410b3475f02820c", merklePath.Path[0])
	assert.Equal(t, "cd73c0c6bb645581816fa960fd2f1636062fcbf23cb57981074ab8d708a76e3b", merklePath.Path[1])
	assert.Equal(t, "b4c8d919190a090e77b73ffcd52b85babaaeeb62da000473102aca7f070facef", merklePath.Path[2])
	assert.Equal(t, "3470d882cf556a4b943639eba15dc795dffdbebdc98b9a98e3637fda96e3811e", merklePath.Path[3])
}
