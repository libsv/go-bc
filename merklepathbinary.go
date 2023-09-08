package bc

import (
	"encoding/hex"

	"github.com/libsv/go-bt/v2"
)

// Merkle path data model json format according to BRC-58.
type MerklePathData struct {
	Index uint64   `json:"index"`
	Path  []string `json:"path"`
}

// Merkle Path Binary Format according to BRC-71 [index, nLeaves, [leaf0, leaf1, leaf2, ... leafnLeaves-1]].
type MerklePath string

// Based on merkle path data model builds merkle path binary format.
func BuildMerklePathBinary(merklePath *MerklePathData) (MerklePath, error) {
	index := bt.VarInt(merklePath.Index)
	nLeaves := bt.VarInt(len(merklePath.Path))

	// first two arguments in merkle path bynary format are index of the transaction and number of leaves
	bytes := []byte{}
	bytes = append(bytes, index.Bytes()...)
	bytes = append(bytes, nLeaves.Bytes()...)

	// now add each leave into the binary path
	for _, leave := range merklePath.Path {
		// decode hex leave into bytes
		leaveBytes, err := hex.DecodeString(leave)
		if err != nil {
			return "", err
		}

		// append leave bytes into binary path
		bytes = append(bytes, leaveBytes...)
	}

	return MerklePath(hex.EncodeToString(bytes)), nil
}

// from merkle path binary decodes MerklePathData.
func DecodeMerklePathBinary(merklePath MerklePath) (*MerklePathData, error) {
	// convert hex to byte array
	merklePathBinary, err := hex.DecodeString(string(merklePath))
	if err != nil {
		return nil, err
	}

	merklePathData := &MerklePathData{}
	merklePathData.Path = make([]string, 0)

	// start paring transaction index
	var offset int
	index, size := bt.NewVarIntFromBytes(merklePathBinary[offset:])
	merklePathData.Index = uint64(index)
	offset += size

	// next value in the byte array is nLeaves (number of leaves in merkle path)
	nLeaves, size := bt.NewVarIntFromBytes(merklePathBinary[offset:])
	offset += size

	// parse each leaf from the binary path
	for k := 0; k < int(nLeaves); k++ {
		leaf := merklePathBinary[offset : offset+32]
		merklePathData.Path = append(merklePathData.Path, hex.EncodeToString(leaf))
		offset += 32
	}

	return merklePathData, nil
}
