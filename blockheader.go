package bc

import (
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/libsv/go-bt"
)

// A BlockHeader in the Bitcoin blockchain.
type BlockHeader struct {
	Version        uint32
	HashPrevBlock  string
	HashMerkleRoot string
	Time           uint32
	Bits           string
	Nonce          uint32
}

// EncodeBlockHeaderStr will encode a block header byte slice
// into the bitcoin block header structure.
// See https://en.bitcoin.it/wiki/Block_hashing_algorithm
func EncodeBlockHeaderStr(headerStr string) (*BlockHeader, error) {
	if len(headerStr) != 160 {
		return nil, errors.New("block header should be 80 bytes long")
	}

	headerBytes, err := hex.DecodeString(headerStr)
	if err != nil {
		return nil, err
	}

	return EncodeBlockHeader(headerBytes)
}

// EncodeBlockHeader will encode a block header byte slice
// into the bitcoin block header structure.
// See https://en.bitcoin.it/wiki/Block_hashing_algorithm
func EncodeBlockHeader(headerBytes []byte) (*BlockHeader, error) {
	if len(headerBytes) != 80 {
		return nil, errors.New("block header should be 80 bytes long")
	}

	return &BlockHeader{
		Version:        binary.LittleEndian.Uint32(headerBytes[:4]),
		HashPrevBlock:  hex.EncodeToString(bt.ReverseBytes(headerBytes[4:36])),
		HashMerkleRoot: hex.EncodeToString(bt.ReverseBytes(headerBytes[36:68])),
		Time:           binary.LittleEndian.Uint32(headerBytes[68:72]),
		Bits:           hex.EncodeToString(bt.ReverseBytes(headerBytes[72:76])),
		Nonce:          binary.LittleEndian.Uint32(headerBytes[76:]),
	}, nil
}

// DecodeBlockHeader will decode a bitcoin block header struct
// into a byte slice.
// See https://en.bitcoin.it/wiki/Block_hashing_algorithm
func DecodeBlockHeader(header *BlockHeader) ([]byte, error) {
	bytes := []byte{}

	v := make([]byte, 4)
	binary.LittleEndian.PutUint32(v, header.Version)
	bytes = append(bytes, v...)

	p, err := hex.DecodeString(header.HashPrevBlock)
	if err != nil {
		return nil, err
	}
	p = bt.ReverseBytes(p)
	bytes = append(bytes, p...)

	m, err := hex.DecodeString(header.HashMerkleRoot)
	if err != nil {
		return nil, err
	}
	m = bt.ReverseBytes(m)
	bytes = append(bytes, m...)

	t := make([]byte, 4)
	binary.LittleEndian.PutUint32(t, header.Time)
	bytes = append(bytes, t...)

	b, err := hex.DecodeString(header.Bits)
	if err != nil {
		return nil, err
	}
	b = bt.ReverseBytes(b)
	bytes = append(bytes, b...)

	n := make([]byte, 4)
	binary.LittleEndian.PutUint32(t, header.Nonce)
	bytes = append(bytes, n...)

	return bytes, nil
}

// BuildBlockHeader builds the block header byte array from the specific fields in the header.
// TODO: check if still needed
func BuildBlockHeader(version uint32, previousBlockHash string, merkleRoot []byte, time []byte, bits []byte, nonce []byte) []byte {
	v := make([]byte, 4)
	binary.LittleEndian.PutUint32(v, version)
	p, _ := hex.DecodeString(previousBlockHash)

	p = bt.ReverseBytes(p)

	a := []byte{}
	a = append(a, v...)
	a = append(a, p...)
	a = append(a, merkleRoot...)
	a = append(a, time...)
	a = append(a, bits...)
	a = append(a, nonce...)
	return a
}
