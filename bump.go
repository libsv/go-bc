package bc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/libsv/go-bt/v2"
)

// BUMP data model json format according to BRC-74.
type BUMP struct {
	BlockHeight uint32   `json:"blockHeight"`
	Path        [][]leaf `json:"path"`
}

// It should be written such that the internal bytes are kept for calculations.
// and the JSON is generated from the internal struct to an external format.
// leaf represents a leaf in the Merkle tree.
type leaf struct {
	Offset    *uint64 `json:"offset,omitempty"`
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
	bump.Path = make([][]leaf, treeHeight)

	for lv := uint(0); lv < treeHeight; lv++ {
		// For each level we parse a bunch of nLeaves.
		n, size := bt.NewVarIntFromBytes(bytes[skip:])
		skip += size
		nLeavesAtThisHeight := uint64(n)
		bump.Path[lv] = make([]leaf, nLeavesAtThisHeight)
		for lf := uint64(0); lf < nLeavesAtThisHeight; lf++ {
			// For each leaf we parse the offset, hash, txid and duplicate.
			offset, size := bt.NewVarIntFromBytes(bytes[skip:])
			skip += size
			var l leaf
			o := uint64(offset)
			l.Offset = &o
			flags := bytes[skip]
			skip++
			dup := flags&1 > 0
			txid := flags&2 > 0
			if dup {
				l.Duplicate = &dup
			}
			if txid {
				l.Txid = &txid
			}
			h := StringFromBytesReverse(bytes[skip : skip+32])
			l.Hash = &h
			skip += 32
			bump.Path[lv][lf] = l
		}
	}

	// Sort each of the levels by the offset for consistency.
	for _, level := range bump.Path {
		sort.Slice(level, func(i, j int) bool {
			return *level[i].Offset < *level[j].Offset
		})
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

// Bytes encodes a BUMP as a slice of bytes. BUMP Binary Format according to BRC-74 https://brc.dev/74
func (bump *BUMP) Bytes() ([]byte, error) {
	bytes := []byte{}
	bytes = append(bytes, bt.VarInt(bump.BlockHeight).Bytes()...)
	treeHeight := len(bump.Path)
	bytes = append(bytes, byte(treeHeight))
	for level := 0; level < treeHeight; level++ {
		nLeaves := len(bump.Path[level])
		bytes = append(bytes, bt.VarInt(nLeaves).Bytes()...)
		for _, leaf := range bump.Path[level] {
			bytes = append(bytes, bt.VarInt(*leaf.Offset).Bytes()...)
			flags := byte(0)
			if leaf.Duplicate != nil {
				flags |= 1
			}
			if leaf.Txid != nil {
				flags |= 2
			}
			bytes = append(bytes, flags)
			if (flags & 1) == 0 {
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
	txidFound := false
	for _, l := range bump.Path[0] {
		if *l.Hash == txid {
			txidFound = true
			index = *l.Offset
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
		var leafAtThisLevel leaf
		offsetFound := false
		for _, l := range leaves {
			if *l.Offset == offset {
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
			leafBytes := BytesFromStringReverse(*leafAtThisLevel.Hash)
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

// NewBUMPFromMerkleTree with merkle tree we calculate the merkle path for a given transaction.
func NewBUMPFromMerkleTree(blockHeight uint32, merkleTree []string) (*BUMP, error) {
	bump := &BUMP{
		BlockHeight: blockHeight,
		Path:        [][]leaf{},
	}
	t := true

	numofHashes := len(merkleTree) / 2
	exponent := int(math.Log2(float64(numofHashes))) + 1

	// if we have only one transaction in the block there is no merkle path to calculate
	if len(merkleTree) != 1 {
		// if our hash index is odd the next hash of the path is the previous element in the array otherwise the next element.
		for height := 0; height < exponent; height++ {
			leaves := []leaf{}
			bump.Path = append(bump.Path, leaves)
			for offset := 0; offset <= numofHashes; offset++ {
				o := uint64(offset)
				thisLeaf := leaf{Offset: &o}
				hash := merkleTree[height*2+offset]
				if hash == "" {
					thisLeaf.Duplicate = &t
				} else {
					thisLeaf.Hash = &hash
					if height == 0 {
						thisLeaf.Txid = &t
					}
				}
				bump.Path[height] = append(bump.Path[height], thisLeaf)
			}
			numofHashes >>= 1
		}
	} else {
		h := merkleTree[0]
		o := uint64(0)
		bump.Path[0][0] = leaf{Hash: &h, Offset: &o, Txid: &t}
	}

	return bump, nil
}
