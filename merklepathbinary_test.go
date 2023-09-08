package bc_test

import (
	"testing"

	"github.com/libsv/go-bc"

	"github.com/stretchr/testify/assert"
)

func TestBuildingMerklePathBinary(t *testing.T) {
	t.Parallel()

	// build example merkle path data
	merklePathData := bc.MerklePathData{
		Index: 136,
		Path: []string{"0c82025f47b31054e9ad52109ef25b00fd9aaae7153564619bab031d4112f56c",
			"3b6ea708d7b84a078179b53cf2cb2f0636162ffd60a96f81815564bbc6c073cd",
			"efac0f077fca2a10730400da62ebaebaba852bd5fc3fb7770e090a1919d9c8b4",
			"1e81e396da7f63e3989a8bc9bdbefddf95c75da1eb3936944b6a55cf82d87034"},
	}

	// build binary path from it
	merklePathBinary, err := bc.BuildMerklePathBinary(&merklePathData)
	if err != nil {
		t.Error(err)
		return
	}

	// assert binary path is expected
	assert.Equal(t, bc.MerklePath("88040c82025f47b31054e9ad52109ef25b00fd9aaae7153564619bab031d4112f56c3b6ea708d7b84a078179b53cf2cb2f0636162ffd60a96f81815564bbc6c073cdefac0f077fca2a10730400da62ebaebaba852bd5fc3fb7770e090a1919d9c8b41e81e396da7f63e3989a8bc9bdbefddf95c75da1eb3936944b6a55cf82d87034"), merklePathBinary)
}

func TestDecodingMerklePathBinary(t *testing.T) {
	t.Parallel()

	merklePath := bc.MerklePath("88040c82025f47b31054e9ad52109ef25b00fd9aaae7153564619bab031d4112f56c3b6ea708d7b84a078179b53cf2cb2f0636162ffd60a96f81815564bbc6c073cdefac0f077fca2a10730400da62ebaebaba852bd5fc3fb7770e090a1919d9c8b41e81e396da7f63e3989a8bc9bdbefddf95c75da1eb3936944b6a55cf82d87034")
	merklePathData, err := bc.DecodeMerklePathBinary(merklePath)
	if err != nil {
		t.Error(err)
		return
	}

	// data we are expecting to deserialize
	// merklePathData := bc.MerklePathData{
	// 	Index: 136,
	// 	Path: []string{"0c82025f47b31054e9ad52109ef25b00fd9aaae7153564619bab031d4112f56c",
	// 		"3b6ea708d7b84a078179b53cf2cb2f0636162ffd60a96f81815564bbc6c073cd",
	// 		"efac0f077fca2a10730400da62ebaebaba852bd5fc3fb7770e090a1919d9c8b4",
	// 		"1e81e396da7f63e3989a8bc9bdbefddf95c75da1eb3936944b6a55cf82d87034"},
	// }

	// assert binary path is expected
	assert.Equal(t, uint64(136), merklePathData.Index)
	assert.Equal(t, 4, len(merklePathData.Path))
	assert.Equal(t, "0c82025f47b31054e9ad52109ef25b00fd9aaae7153564619bab031d4112f56c", merklePathData.Path[0])
	assert.Equal(t, "3b6ea708d7b84a078179b53cf2cb2f0636162ffd60a96f81815564bbc6c073cd", merklePathData.Path[1])
	assert.Equal(t, "efac0f077fca2a10730400da62ebaebaba852bd5fc3fb7770e090a1919d9c8b4", merklePathData.Path[2])
	assert.Equal(t, "1e81e396da7f63e3989a8bc9bdbefddf95c75da1eb3936944b6a55cf82d87034", merklePathData.Path[3])
}
