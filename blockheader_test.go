package bc_test

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/libsv/go-bc"
	"github.com/stretchr/testify/assert"
)

func TestNewBlockHeader(t *testing.T) {
	ebh := &bc.BlockHeader{
		Version: 536870912,
		HashPrevBlock: func() []byte {
			t, _ := hex.DecodeString("784605133dff2eb242d5b9c5b6dc07d7b9677f2b127ed824910e89e79477a174")
			return t
		}(),
		HashMerkleRoot: func() []byte {
			t, _ := hex.DecodeString("fc9d931e8eecd947c870279571840a727924bf8cb1243587c0c22620a9afd8e9")
			return t
		}(),
		Time: 1614043423,
		Bits: func() []byte {
			t, _ := hex.DecodeString("207fffff")
			return t
		}(),
		Nonce: 0,
	}

	headerBytes := "0000002074a17794e7890e9124d87e122b7f67b9d707dcb6c5b9d542b22eff3d13054678e9d8afa92026c2c0873524b18cbf2479720a8471952770c847d9ec8e1e939dfc1f593460ffff7f2000000000"
	bh, err := bc.NewBlockHeaderFromStr(headerBytes)

	assert.NoError(t, err)
	assert.Equal(t, ebh, bh)
}

func TestBlockHeaderString(t *testing.T) {
	expectedHeader := "00000020fb9eacea87c1cc294a4f1633a45b9bfb21cf9878b439c6138d96b8ca3a856e3a37307cd123724eaa4ade23d29feea1358458d5c110275b6cca4e2b79cd14d98e39573460ffff7f2000000000"

	bh := &bc.BlockHeader{
		Version: 536870912,
		HashPrevBlock: func() []byte {
			t, _ := hex.DecodeString("3a6e853acab8968d13c639b47898cf21fb9b5ba433164f4a29ccc187eaac9efb")
			return t
		}(),
		HashMerkleRoot: func() []byte {
			t, _ := hex.DecodeString("8ed914cd792b4eca6c5b2710c1d5588435a1ee9fd223de4aaa4e7223d17c3037")
			return t
		}(),
		Time: 1614042937,
		Bits: func() []byte {
			t, _ := hex.DecodeString("207fffff")
			return t
		}(),
		Nonce: 0,
	}

	assert.Equal(t, expectedHeader, bh.String())
}

func TestBlockHeaderStringAndBytesMatch(t *testing.T) {
	headerStr := "0000002074a17794e7890e9124d87e122b7f67b9d707dcb6c5b9d542b22eff3d13054678e9d8afa92026c2c0873524b18cbf2479720a8471952770c847d9ec8e1e939dfc1f593460ffff7f2000000000"
	bh, err := bc.NewBlockHeaderFromStr(headerStr)
	assert.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(bh.Bytes()), bh.String())
}

func TestBlockHeaderInvalid(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		expectedHeader string
		expErr         error
	}{
		"empty string": {
			expectedHeader: "",
			expErr:         errors.New("block header should be 80 bytes long"),
		},
		"too long": {
			expectedHeader: "00000020fb9eacea87c1cc294a4f1633a45b9bfb21cf9878b439c61123221312312312396b8ca3a856e3a37307cd123724eaa4ade23d29feea1358458d5c110275b6cca4e2b79cd14d98e39573460ffff7f2000000000",
			expErr:         errors.New("block header should be 80 bytes long"),
		},
		"too short": {
			expectedHeader: "00000020fb9eacea87c1c3a856e3a37307cd123724eaa4ade23d29feea1358458d5c110275b6cca4e2b79cd14d98e39573460ffff7f2000000000",
			expErr:         errors.New("block header should be 80 bytes long"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := bc.NewBlockHeaderFromStr(test.expectedHeader)
			assert.Error(t, err)
			assert.EqualError(t, err, test.expErr.Error())
		})
	}
}

func TestExtractMerkleRootFromBlockHeader(t *testing.T) {
	header := "000000208e33a53195acad0ab42ddbdbe3e4d9ca081332e5b01a62e340dbd8167d1a787b702f61bb913ac2063e0f2aed6d933d3386234da5c8eb9e30e498efd25fb7cb96fff12c60ffff7f2001000000"

	merkleRoot, err := bc.ExtractMerkleRootFromBlockHeader(header)

	assert.NoError(t, err)
	assert.Equal(t, merkleRoot, "96cbb75fd2ef98e4309eebc8a54d2386333d936ded2a0f3e06c23a91bb612f70")
}

func TestEncodeAndDecodeBlockHeader(t *testing.T) {
	// the genesis block
	genesisHex := "0100000000000000000000000000000000000000000000000000000000000000000000003ba3edfd7a7b12b27ac72c3e67768f617fc81bc3888a51323a9fb8aa4b1e5e4a29ab5f49ffff001d1dac2b7c"
	genesis, err := bc.NewBlockHeaderFromStr(genesisHex)
	assert.NoError(t, err)
	assert.Equal(t, genesisHex, genesis.String())
}

func TestVerifyBlockHeader(t *testing.T) {
	// the genesis block
	genesisHex := "0100000000000000000000000000000000000000000000000000000000000000000000003ba3edfd7a7b12b27ac72c3e67768f617fc81bc3888a51323a9fb8aa4b1e5e4a29ab5f49ffff001d1dac2b7c"
	header, err := hex.DecodeString(genesisHex)
	assert.NoError(t, err)
	genesis, err := bc.NewBlockHeaderFromBytes(header)
	assert.NoError(t, err)

	assert.Equal(t, genesisHex, genesis.String())
	assert.True(t, genesis.Valid())

	// change one letter
	header[0] = 222
	genesisInvalid, err := bc.NewBlockHeaderFromBytes(header)
	assert.NoError(t, err)
	assert.False(t, genesisInvalid.Valid())
}
