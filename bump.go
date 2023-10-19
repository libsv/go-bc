package bc

import (
	"encoding/hex"
	"fmt"

	"github.com/libsv/go-bt/v2"
)

// BumpJSON data model json format according to BRC-74.
type BumpJSON struct {
	BlockHeight uint32            `json:"blockHeight"`
	Path        []map[string]leaf `json:"path"`
}

// It should be written such that the internal bytes are kept for calculations.
// and the JSON is generated from the internal struct to an external format.

type leaf struct {
	Hash      string `json:"hash"`
	Txid      bool
	Duplicate bool
}

// NewMerklePathFromBytes creates a new MerklePath from a byte slice.
func NewBUMPFromBytes(bytes []byte) (*BUMP, error) {
	bump := &BUMP{}

	// first bytes are the block height.
	var skip int
	index, size := bt.NewVarIntFromBytes(bytes[skip:])
	skip += size
	bump.BlockHeight = uint32(index)

	// Next byte is the tree height.
	treeHeight := uint(bytes[skip])
	skip++

	// We expect tree height levels.
	bump.Path = make([]map[string]leaf, treeHeight)

	for lv := uint(0); lv < treeHeight; lv++ {
		n, size := bt.NewVarIntFromBytes(bytes[skip:])
		skip += size
		nLeavesAtThisHeight := uint64(n)
		// For each level we parse a bunch of leaves.
		for lf := uint64(0); lf < nLeavesAtThisHeight; lf++ {
			// For each leaf we need to parse the offset, hash, txid and duplicate.
			offset, size := bt.NewVarIntFromBytes(bytes[skip:])
			skip += size
			flags := uint8(bytes[skip])
			skip++
			var l leaf
			l.Duplicate = flags&1 > 0
			l.Txid = flags&2 > 0
			l.Hash = StringFromBytesReverse(bytes[skip : skip+32])
			skip += 32
			bump.Path[lv] = map[string]leaf{fmt.Sprint(uint64(offset)): l}
		}
	}

	return bump, nil
}

// NewBUMPFromStr creates a BUMP from hex string.
func NewBUMPFromStr(str string) (*BUMP, error) {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return NewBUMPFromBytes(bytes)
}

// // Bytes encodes a MerklePath as a slice of bytes. MerklePath Binary Format according to BRC-71 https://brc.dev/71
// func (mp *MerklePath) Bytes() ([]byte, error) {
// 	index := bt.VarInt(mp.Index)
// 	nLeaves := bt.VarInt(len(mp.Path))

// 	// first two arguments in merkle path bynary format are index of the transaction and number of leaves.
// 	bytes := []byte{}
// 	bytes = append(bytes, index.Bytes()...)
// 	bytes = append(bytes, nLeaves.Bytes()...)

// 	// now add each leaf into the binary path.
// 	for _, leaf := range mp.Path {
// 		// append leaf bytes into binary path, little endian.
// 		bytes = append(bytes, BytesFromStringReverse(leaf)...)
// 	}

// 	return bytes, nil
// }

// // String encodes a MerklePath as a hex string.
// func (mp *MerklePath) String() (string, error) {
// 	bytes, err := mp.Bytes()
// 	if err != nil {
// 		return "", err
// 	}
// 	return hex.EncodeToString(bytes), nil
// }

// // CalculateRoot calculates the merkle root from a transaction ID and a MerklePath.
// func (mp *MerklePath) CalculateRoot(txid string) (string, error) {
// 	// start with txid
// 	workingHash := BytesFromStringReverse(txid)
// 	lsb := mp.Index
// 	// hash with each path branch
// 	for _, leaf := range mp.Path {
// 		var digest []byte
// 		leafBytes := BytesFromStringReverse(leaf)
// 		// if the least significant bit is 1 then the working hash is on the right.
// 		if lsb&1 > 0 {
// 			digest = append(leafBytes, workingHash...)
// 		} else {
// 			digest = append(workingHash, leafBytes...)
// 		}
// 		workingHash = Sha256Sha256(digest)
// 		lsb = lsb >> 1
// 	}
// 	return StringFromBytesReverse(workingHash), nil
// }

// // getPathElements traverses the tree and returns the path to Merkle root.
// func getPathElements(txIndex int, hashes []string) []string {
// 	// if our hash index is odd the next hash of the path is the previous element in the array otherwise the next element.
// 	var path []string
// 	var hash string
// 	if txIndex%2 == 0 {
// 		hash = hashes[txIndex+1]
// 	} else {
// 		hash = hashes[txIndex-1]
// 	}

// 	// when generating path if the neighbour is empty we append itself
// 	if hash == "" {
// 		path = append(path, hashes[txIndex])
// 	} else {
// 		path = append(path, hash)
// 	}

// 	// If we reach the Merkle root hash stop path calculation.
// 	if len(hashes) == 3 {
// 		return path
// 	}

// 	return append(path, getPathElements(txIndex/2, hashes[(len(hashes)+1)/2:])...)
// }

// // GetTxMerklePath with merkle tree we calculate the merkle path for a given transaction.
// func GetTxMerklePath(txIndex int, merkleTree []string) *MerklePath {
// 	merklePath := &MerklePath{
// 		Index: uint64(txIndex),
// 		Path:  nil,
// 	}

// 	// if we have only one transaction in the block there is no merkle path to calculate
// 	if len(merkleTree) != 1 {
// 		merklePath.Path = getPathElements(txIndex, merkleTree)
// 	}

// 	return merklePath
// }
