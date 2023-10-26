package bc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
// leaf represents a leaf in the Merkle tree.
type leaf struct {
	Hash      *string `json:"hash,omitempty"`
	Txid      *bool   `json:"txid,omitempty"`
	Duplicate *bool   `json:"duplicate,omitempty"`
}

type leafWithOffset struct {
	Offset    uint64  `json:"offset,omitempty"`
	Hash      *string `json:"hash,omitempty"`
	Txid      *bool   `json:"txid,omitempty"`
	Duplicate *bool   `json:"duplicate,omitempty"`
}

// NewBUMPFromBytes creates a new BUMP from a byte slice.
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
		bump.Path[lv] = make(map[string]leaf, nLeavesAtThisHeight)
		// For each level we parse a bunch of leaves.
		for lf := uint64(0); lf < nLeavesAtThisHeight; lf++ {
			// For each leaf we need to parse the offset, hash, txid and duplicate.
			offset, size := bt.NewVarIntFromBytes(bytes[skip:])
			skip += size
			flags := bytes[skip]
			skip++
			var l leaf
			var dup bool
			var txid bool
			dup = flags&1 > 0
			txid = flags&2 > 0
			if dup {
				l.Duplicate = &dup
			} else {
				hash := StringFromBytesReverse(bytes[skip : skip+32])
				l.Hash = &hash
				skip += 32
			}
			if txid {
				l.Txid = &txid
			}
			bump.Path[lv][fmt.Sprint(uint64(offset))] = l
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

// NewBUMPFromJSON creates a BUMP from a JSON string.
func NewBUMPFromJSON(jsonStr string) (*BUMP, error) {
	bump := &BUMP{}
	err := json.Unmarshal([]byte(jsonStr), bump)
	if err != nil {
		return nil, err
	}
	return bump, nil
}

func sortLeavesByOffset(leaves map[string]leaf) []leafWithOffset {
	orderedLeaves := make([]leafWithOffset, 0)
	for offset, leaf := range leaves {
		offsetInt, err := strconv.ParseUint(offset, 10, 64)
		if err != nil {
			panic(err)
		}
		orderedLeaves = append(orderedLeaves, leafWithOffset{
			Offset:    offsetInt,
			Hash:      leaf.Hash,
			Txid:      leaf.Txid,
			Duplicate: leaf.Duplicate,
		})
	}
	// sort by offset
	sort.Slice(orderedLeaves, func(i, j int) bool {
		return orderedLeaves[i].Offset < orderedLeaves[j].Offset
	})
	return orderedLeaves
}

// Bytes encodes a BUMP as a slice of bytes. BUMP Binary Format according to BRC-74 https://brc.dev/74
func (bump *BUMP) Bytes() ([]byte, error) {
	bytes := []byte{}
	bytes = append(bytes, bt.VarInt(bump.BlockHeight).Bytes()...)
	treeHeight := len(bump.Path)
	bytes = append(bytes, byte(treeHeight))
	for level := 0; level < treeHeight; level++ {
		nLeaves := len(bump.Path[level])
		bytes = append(bytes, bt.VarInt(nLeaves).Bytes()...)
		// order by offset
		orderedLeaves := sortLeavesByOffset(bump.Path[level])
		for _, leaf := range orderedLeaves {
			bytes = append(bytes, bt.VarInt(leaf.Offset).Bytes()...)
			flags := byte(0)
			if leaf.Duplicate != nil {
				flags |= 1
			}
			if leaf.Txid != nil {
				flags |= 2
			}
			bytes = append(bytes, flags)
			if leaf.Duplicate == nil {
				bytes = append(bytes, BytesFromStringReverse(*leaf.Hash)...)
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

// CalculateRootGivenTxid calculates the root of the Merkle tree given a txid.
func (bump *BUMP) CalculateRootGivenTxid(txid string) (string, error) {
	// Find the index of the txid at the lowest level of the Merkle tree
	var index uint64
	found := false
	for offset, leaf := range bump.Path[0] {
		if *leaf.Hash == txid {
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
		lf, exists := leaves[fmt.Sprint(offset)]
		if !exists {
			return "", fmt.Errorf("We do not have a hash for this index at height: %v", height)
		}
		var digest []byte
		if lf.Duplicate != nil {
			digest = append(workingHash, workingHash...)
		} else {
			leafBytes := BytesFromStringReverse(*lf.Hash)
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

// compareRoots compares the roots of two BUMP objects.
func compareRoots(bump *BUMP, another *BUMP) error {
	var firstTxid string
	// keys of map
	for _, lf := range bump.Path[0] {
		if lf.Hash != nil {
			firstTxid = *lf.Hash
			break
		}
	}
	// keys of the first level are the txids
	root1, err := bump.CalculateRootGivenTxid(firstTxid)
	if err != nil {
		return err
	}
	var firstTxidInOther string
	// keys of map
	for _, lf := range another.Path[0] {
		if lf.Hash != nil {
			firstTxidInOther = *lf.Hash
			break
		}
	}
	root2, err := another.CalculateRootGivenTxid(firstTxidInOther)
	if err != nil {
		return err
	}
	if root1 != root2 {
		return errors.New("roots mismatch")
	}
	return nil
}

// Add combines two BUMP objects.
func (bump *BUMP) Add(another *BUMP) (*BUMP, error) {
	if bump.BlockHeight != another.BlockHeight {
		return nil, errors.New("block height mismatch")
	}
	if len(bump.Path) != len(another.Path) {
		return nil, errors.New("tree height mismatch")
	}
	err := compareRoots(bump, another)
	if err != nil {
		return nil, err
	}
	combinedPath := make([]map[string]leaf, 0)
	for level, leaves := range another.Path {
		leavesAtThisLevel := bump.Path[level]
		for offset, anotherLeaf := range leaves {
			if lf, exists := bump.Path[level][offset]; exists {
				if lf.Txid == nil && lf.Txid != nil {
					lf.Txid = anotherLeaf.Txid
				}
			} else {
				leavesAtThisLevel[offset] = anotherLeaf
			}
		}
		orderedLeaves := sortLeavesByOffset(leavesAtThisLevel)
		latl := make(map[string]leaf, 0)
		for _, l := range orderedLeaves {
			latl[fmt.Sprint(l.Offset)] = leaf{
				Hash:      l.Hash,
				Txid:      l.Txid,
				Duplicate: l.Duplicate,
			}
		}
		combinedPath = append(combinedPath, latl)
	}
	return &BUMP{
		BlockHeight: bump.BlockHeight,
		Path:        combinedPath,
	}, nil
}
