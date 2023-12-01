package bc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/libsv/go-bt/v2"
	"github.com/libsv/go-p2p/chaincfg/chainhash"
)

// BUMP data model json format according to BRC-74.
type BUMP struct {
	BlockHeight uint64   `json:"blockHeight"`
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
	if len(bytes) < 37 {
		return nil, errors.New("BUMP bytes do not contain enough data to be valid")
	}
	bump := &BUMP{}

	// first bytes are the block height.
	var skip int
	index, size := bt.NewVarIntFromBytes(bytes[skip:])
	skip += size
	bump.BlockHeight = uint64(index)

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
		if nLeavesAtThisHeight == 0 {
			return nil, errors.New("There are no leaves at height: " + fmt.Sprint(lv) + " which makes this invalid")
		}
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
			} else {
				if len(bytes) < skip+32 {
					return nil, errors.New("BUMP bytes do not contain enough data to be valid")
				}
				h := StringFromBytesReverse(bytes[skip : skip+32])
				l.Hash = &h
				skip += 32
			}
			if txid {
				l.Txid = &txid
			}
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

// String encodes a BUMP as a hex string.
func (bump *BUMP) String() (string, error) {
	bytes, err := bump.Bytes()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Txids returns the txids within the BUMP which the client is expecting.
// This allows a client to receive one BUMP for a whole block and it will know which txids it should update.
func (bump *BUMP) Txids() []string {
	txids := make([]string, 0)
	for _, leaf := range bump.Path[0] {
		if leaf.Txid != nil {
			txids = append(txids, *leaf.Hash)
		}
	}
	return txids
}

// CalculateRootGivenTxid calculates the root of the Merkle tree given a txid.
func (bump *BUMP) CalculateRootGivenTxid(txid string) (string, error) {
	if len(bump.Path) == 1 {
		// if there is only one txid in the block then the root is the txid.
		if len(bump.Path[0]) == 1 {
			return txid, nil
		}
	}
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

// NewBUMPFromMerkleTreeAndIndex with merkle tree we calculate the merkle path for a given transaction.
func NewBUMPFromMerkleTreeAndIndex(blockHeight uint64, merkleTree []*chainhash.Hash, txIndex uint64) (*BUMP, error) {
	bump := &BUMP{
		BlockHeight: blockHeight,
		Path:        [][]leaf{},
	}
	t := true

	numOfTxids := (len(merkleTree) + 1) / 2
	treeHeight := int(math.Log2(float64(numOfTxids)))
	numOfHashes := numOfTxids

	if len(merkleTree) == 0 {
		return nil, errors.New("merkle tree is empty")
	}

	// these are the offsets for the txid we're interested in.
	offsets := make([]uint64, treeHeight)
	for i := 0; i < treeHeight; i++ {
		if txIndex>>uint64(i)&1 == 0 {
			offsets[i] = txIndex>>uint64(i) + 1
		} else {
			offsets[i] = txIndex>>uint64(i) - 1
		}
	}

	// if we have only one transaction in the block there is no merkle path to calculate
	if len(merkleTree) != 1 {
		// if our hash index is odd the next hash of the path is the previous element in the array otherwise the next element.
		levelOffset := 0
		for height := 0; height < treeHeight; height++ {
			leaves := []leaf{}
			bump.Path = append(bump.Path, leaves)
			for offset := 0; offset < numOfHashes; offset++ {
				o := uint64(offset)
				// only include the hashes for the txid we're interested in.
				if height == 0 {
					if o != txIndex && o != offsets[height] {
						continue
					}
				} else {
					if o != offsets[height] {
						continue
					}
				}
				thisLeaf := leaf{Offset: &o}
				hash := merkleTree[levelOffset+offset]
				if hash.IsEqual(nil) {
					thisLeaf.Duplicate = &t
				} else {
					sh := hash.String()
					thisLeaf.Hash = &sh
					if height == 0 && txIndex == o {
						thisLeaf.Txid = &t
					}
				}
				bump.Path[height] = append(bump.Path[height], thisLeaf)
			}
			levelOffset += numOfHashes
			numOfHashes >>= 1
		}
	} else {
		sh := merkleTree[0].String()
		o := uint64(0)
		bump.Path = [][]leaf{{leaf{Hash: &sh, Offset: &o, Txid: &t}}}
	}

	return bump, nil
}
