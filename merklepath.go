package bc

import (
	"encoding/hex"

	"crypto/sha256"

	"github.com/libsv/go-bt/v2"
)

// MerklePath data model json format according to BRC-58.
type MerklePath struct {
	Index uint64   `json:"index"`
	Path  []string `json:"path"`
}

// BytesFromStringReverse decodes a hex string into a byte slice and reverses it.
func BytesFromStringReverse(s string) []byte {
	bytes, _ := hex.DecodeString(s)
	rev := bt.ReverseBytes(bytes)
	return rev
}

// StringFromBytesReverse reverses a byte slice and encodes it as a hex string.
func StringFromBytesReverse(h []byte) string {
	rev := bt.ReverseBytes(h)
	return hex.EncodeToString(rev)
}

// Sha256Sha256 calculates the double sha256 hash of a byte slice.
func Sha256Sha256(digest []byte) []byte {
	sha := sha256.Sum256(digest)
	dsha := sha256.Sum256(sha[:])
	return dsha[:]
}

// NewMerklePathFromBytes creates a new MerklePath from a byte slice.
func NewMerklePathFromBytes(bytes []byte) (*MerklePath, error) {
	mp := &MerklePath{}
	mp.Path = make([]string, 0)

	// start paring transaction index.
	var offset int
	index, size := bt.NewVarIntFromBytes(bytes[offset:])
	offset += size
	mp.Index = uint64(index)

	// next value in the byte array is nLeaves (number of leaves in merkle path).
	nLeaves, size := bt.NewVarIntFromBytes(bytes[offset:])
	offset += size

	// parse each leaf from the binary path
	for k := 0; k < int(nLeaves); k++ {
		leaf := bytes[offset : offset+32]
		mp.Path = append(mp.Path, StringFromBytesReverse(leaf))
		offset += 32
	}

	return mp, nil
}

// NewMerklePathFromStr creates a MerklePath from hex string.
func NewMerklePathFromStr(str string) (*MerklePath, error) {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return NewMerklePathFromBytes(bytes)
}

// Bytes encodes a MerklePath as a slice of bytes. MerklePath Binary Format according to BRC-71 https://brc.dev/71
func (mp *MerklePath) Bytes() ([]byte, error) {
	index := bt.VarInt(mp.Index)
	nLeaves := bt.VarInt(len(mp.Path))

	// first two arguments in merkle path bynary format are index of the transaction and number of leaves.
	bytes := []byte{}
	bytes = append(bytes, index.Bytes()...)
	bytes = append(bytes, nLeaves.Bytes()...)

	// now add each leaf into the binary path.
	for _, leaf := range mp.Path {
		// append leaf bytes into binary path, little endian.
		bytes = append(bytes, BytesFromStringReverse(leaf)...)
	}

	return bytes, nil
}

// String encodes a MerklePath as a hex string.
func (mp *MerklePath) String() (string, error) {
	bytes, err := mp.Bytes()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CalculateRoot calculates the merkle root from a transaction ID and a MerklePath.
func (mp *MerklePath) CalculateRoot(txid string) (string, error) {
	// start with txid
	workingHash := BytesFromStringReverse(txid)
	lsb := mp.Index
	// hash with each path branch
	for _, leaf := range mp.Path {
		var digest []byte
		leafBytes := BytesFromStringReverse(leaf)
		// if the least significant bit is 1 then the working hash is on the right.
		if lsb&1 > 0 {
			digest = append(leafBytes, workingHash...)
		} else {
			digest = append(workingHash, leafBytes...)
		}
		workingHash = Sha256Sha256(digest)
		lsb = lsb >> 1
	}
	return StringFromBytesReverse(workingHash), nil
}

// getPathElements traverses the tree and returns the path to coinbase.
func getPathElements(txIndex int, hashes []string) []string {
	// if our hash index is odd the next hash of the path is the previous element in the array otherwise the next element.
	var path []string
	if txIndex%2 == 0 {
		path = append(path, hashes[txIndex+1])
	} else {
		path = append(path, hashes[txIndex-1])
	}

	// If we reach the coinbase hash stop path calculation.
	if len(hashes) == 3 {
		return path
	}

	return append(path, getPathElements(txIndex/2, hashes[(len(hashes)+1)/2:])...)
}

// GetTxMerklePath with merkle tree we calculate the merkle path for a given transaction.
func GetTxMerklePath(txIndex int, merkleTree []string) *MerklePath {
	return &MerklePath{
		Index: uint64(txIndex),
		Path:  getPathElements(txIndex, merkleTree),
	}
}
