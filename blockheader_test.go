package bc_test

import (
	"encoding/hex"
	"testing"

	"github.com/libsv/go-bc"
	"github.com/stretchr/testify/assert"
)

func TestEncodeBlockHeader(t *testing.T) {
	headerBytes := "0000002074a17794e7890e9124d87e122b7f67b9d707dcb6c5b9d542b22eff3d13054678e9d8afa92026c2c0873524b18cbf2479720a8471952770c847d9ec8e1e939dfc1f593460ffff7f2000000000"
	ebh := &bc.BlockHeader{
		Version:        536870912,
		HashPrevBlock:  "784605133dff2eb242d5b9c5b6dc07d7b9677f2b127ed824910e89e79477a174",
		HashMerkleRoot: "fc9d931e8eecd947c870279571840a727924bf8cb1243587c0c22620a9afd8e9",
		Time:           1614043423,
		Bits:           "207fffff",
		Nonce:          0,
	}

	bh, err := bc.EncodeBlockHeaderStr(headerBytes)

	assert.NoError(t, err)
	assert.Equal(t, ebh, bh)
}

func TestDecodeBlockHeader(t *testing.T) {
	bh := &bc.BlockHeader{
		Version:        536870912,
		HashPrevBlock:  "3a6e853acab8968d13c639b47898cf21fb9b5ba433164f4a29ccc187eaac9efb",
		HashMerkleRoot: "8ed914cd792b4eca6c5b2710c1d5588435a1ee9fd223de4aaa4e7223d17c3037",
		Time:           1614042937,
		Bits:           "207fffff",
		Nonce:          0,
	}
	expectedHeader := "00000020fb9eacea87c1cc294a4f1633a45b9bfb21cf9878b439c6138d96b8ca3a856e3a37307cd123724eaa4ade23d29feea1358458d5c110275b6cca4e2b79cd14d98e39573460ffff7f2000000000"

	headerBytes, err := bc.DecodeBlockHeader(bh)
	header := hex.EncodeToString(headerBytes)

	assert.NoError(t, err)
	assert.Equal(t, expectedHeader, header)
}
