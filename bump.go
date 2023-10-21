package bc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/libsv/go-bt/v2"
)

// BUMP data model json format according to BRC-74.
type BUMP struct {
	BlockHeight uint32            `json:"blockHeight"`
	Path        []map[string]leaf `json:"path"`
}

// It should be written such that the internal bytes are kept for calculations.
// and the JSON is generated from the internal struct to an external format.
type leaf struct {
	Hash      string `json:"hash"`
	Txid      *bool  `json:"txid,omitempty"`
	Duplicate *bool  `json:"duplicate,omitempty"`
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
			var dup bool
			var txid bool
			dup = flags&1 > 0
			txid = flags&2 > 0
			if dup {
				l.Duplicate = &dup
			}
			if txid {
				l.Txid = &txid
			}
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

func (bump *BUMP) Bytes() ([]byte, error) {
	bytes := []byte{}
	bytes = append(bytes, bt.VarInt(bump.BlockHeight).Bytes()...)
	treeHeight := len(bump.Path)
	bytes = append(bytes, byte(treeHeight))
	for level := 0; level < treeHeight; level++ {
		nLeaves := len(bump.Path[level])
		bytes = append(bytes, bt.VarInt(nLeaves).Bytes()...)
		for offset, leaf := range bump.Path[level] {
			offsetInt, err := strconv.ParseUint(offset, 10, 64)
			if err != nil {
				return nil, err
			}
			bytes = append(bytes, bt.VarInt(offsetInt).Bytes()...)
			flags := byte(0)
			if leaf.Duplicate != nil {
				flags |= 1
			}
			if leaf.Txid != nil {
				flags |= 2
			}
			bytes = append(bytes, byte(flags))
			if (flags & 1) == 0 {
				bytes = append(bytes, BytesFromStringReverse(leaf.Hash)...)
			}
		}
	}
	return bytes, nil
}

func (bump *BUMP) String() (string, error) {
	bytes, err := bump.Bytes()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (bump *BUMP) CalculateRootGivenTxid(txid string) (string, error) {
	// Find the index of the txid at the lowest level of the Merkle tree
	var index uint64
	found := false
	for offset, leaf := range bump.Path[0] {
		if leaf.Hash == txid {
			found = true
			i, err := strconv.ParseUint(offset, 10, 64)
			if err != nil {
				return "", err
			}
			index = i
			break
		}
	}
	if !found {
		return "", errors.New("The BUMP does not contain the txid: " + txid)
	}

	// Calculate the root using the index as a way to determine which direction to concatenate.
	workingHash := BytesFromStringReverse(txid)
	for height, leaves := range bump.Path {
		offset := (index >> height) ^ 1
		leaf, exists := leaves[string(offset)]
		if !exists {
			return "", fmt.Errorf("We do not have a hash for this index at height: %v", height)
		}
		var digest []byte
		if leaf.Duplicate != nil {
			digest = append(workingHash, workingHash...)
		} else {
			leafBytes := BytesFromStringReverse(leaf.Hash)
			if (offset % 2) != 0 {
				digest = append(leafBytes, workingHash...)
			} else {
				digest = append(workingHash, leafBytes...)
			}
		}
		workingHash = Sha256Sha256(digest)
	}
	return StringFromBytesReverse(workingHash), nil
}
