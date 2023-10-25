package bc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/libsv/go-bt/v2"
)

// BUMP data model json format according to BRC-74.
type BUMP2 struct {
	BlockHeight uint32    `json:"blockHeight"`
	Path        [][]leaf2 `json:"path"`
}

// It should be written such that the internal bytes are kept for calculations.
// and the JSON is generated from the internal struct to an external format.
// leaf represents a leaf in the Merkle tree.
type leaf2 struct {
	Offset    uint64 `json:"offset,omitempty"`
	Hash      string `json:"hash,omitempty"`
	Txid      *bool  `json:"txid,omitempty"`
	Duplicate *bool  `json:"duplicate,omitempty"`
}

// NewBUMPFromBytes creates a new BUMP from a byte slice.
func NewBUMPFromBytes2(bytes []byte) (*BUMP2, error) {
	bump := &BUMP2{}

	// first bytes are the block height.
	var skip int
	index, size := bt.NewVarIntFromBytes(bytes[skip:])
	skip += size
	bump.BlockHeight = uint32(index)

	// Next byte is the tree height.
	treeHeight := uint(bytes[skip])
	skip++

	// We expect tree height levels.
	bump.Path = make([][]leaf2, treeHeight)

	for lv := uint(0); lv < treeHeight; lv++ {
		// For each level we parse a bunch of nLeaves.
		n, size := bt.NewVarIntFromBytes(bytes[skip:])
		skip += size
		nLeavesAtThisHeight := uint64(n)
		bump.Path[lv] = make([]leaf2, nLeavesAtThisHeight)
		for lf := uint64(0); lf < nLeavesAtThisHeight; lf++ {
			// For each leaf we parse the offset, hash, txid and duplicate.
			offset, size := bt.NewVarIntFromBytes(bytes[skip:])
			skip += size
			var l leaf2
			l.Offset = uint64(offset)
			flags := bytes[skip]
			skip++
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
			bump.Path[lv][lf] = l
		}
	}

	// Sort each of the levels by the offset for consistency.
	for _, level := range bump.Path {
		sort.Slice(level, func(i, j int) bool {
			return level[i].Offset < level[j].Offset
		})
	}

	return bump, nil
}

// NewBUMPFromStr creates a BUMP from hex string.
func NewBUMPFromStr2(str string) (*BUMP2, error) {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return NewBUMPFromBytes2(bytes)
}

// NewBUMPFromJSON creates a BUMP from a JSON string.
func NewBUMPFromJSON2(jsonStr string) (*BUMP2, error) {
	bump := &BUMP2{}
	err := json.Unmarshal([]byte(jsonStr), bump)
	if err != nil {
		return nil, err
	}
	return bump, nil
}

// Bytes encodes a BUMP as a slice of bytes. BUMP Binary Format according to BRC-74 https://brc.dev/74
func (bump *BUMP2) Bytes() ([]byte, error) {
	bytes := []byte{}
	bytes = append(bytes, bt.VarInt(bump.BlockHeight).Bytes()...)
	treeHeight := len(bump.Path)
	bytes = append(bytes, byte(treeHeight))
	for level := 0; level < treeHeight; level++ {
		nLeaves := len(bump.Path[level])
		bytes = append(bytes, bt.VarInt(nLeaves).Bytes()...)
		for _, leaf := range bump.Path[level] {
			bytes = append(bytes, bt.VarInt(leaf.Offset).Bytes()...)
			flags := byte(0)
			if leaf.Duplicate != nil {
				flags |= 1
			}
			if leaf.Txid != nil {
				flags |= 2
			}
			bytes = append(bytes, flags)
			if (flags & 1) == 0 {
				bytes = append(bytes, BytesFromStringReverse(leaf.Hash)...)
			}
		}
	}
	return bytes, nil
}

func (bump *BUMP2) String() (string, error) {
	bytes, err := bump.Bytes()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CalculateRootGivenTxid calculates the root of the Merkle tree given a txid.
func (bump *BUMP2) CalculateRootGivenTxid(txid string) (string, error) {
	// Find the index of the txid at the lowest level of the Merkle tree
	var index uint64
	txidFound := false
	for _, l := range bump.Path[0] {
		if l.Hash == txid {
			txidFound = true
			index = l.Offset
			break
		}
	}
	if !txidFound {
		return "", errors.New("The BUMP does not contain the txid: " + txid)
	}

	// Calculate the root using the index as a way to determine which direction to concatenate.
	workingHash := BytesFromStringReverse(txid)
	for height, leaves := range bump.Path {
		offset := (index >> height) ^ 1
		var leafAtThisLevel leaf2
		offsetFound := false
		for _, l := range leaves {
			if l.Offset == offset {
				offsetFound = true
				leafAtThisLevel = l
				break
			}
		}
		if !offsetFound {
			return "", fmt.Errorf("We do not have a hash for this index at height: %v", height)
		}

		var digest []byte
		if leafAtThisLevel.Duplicate != nil {
			digest = append(workingHash, workingHash...)
		} else {
			leafBytes := BytesFromStringReverse(leafAtThisLevel.Hash)
			if (offset % 2) != 0 {
				digest = append(workingHash, leafBytes...)
			} else {
				digest = append(leafBytes, workingHash...)
			}
		}
		workingHash = Sha256Sha256(digest)
	}
	return StringFromBytesReverse(workingHash), nil
}
