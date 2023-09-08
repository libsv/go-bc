package bc

// MerklePathData path data model json format according to BRC-58.
type MerklePathData struct {
	Index uint64   `json:"index"`
	Path  []string `json:"path"`
}

// getPathElements traverses the tree and returns the path to coinbase.
func getPathElements(txIndex int, hashes []string) []string {
	// if our hash index is odd the next hash of the path is the previous
	// element in the array otherwise the next element
	var path []string
	if txIndex%2 == 0 {
		path = append(path, hashes[txIndex+1])
	} else {
		path = append(path, hashes[txIndex-1])
	}

	// If we reach the coinbase hash stop path calculation
	if len(hashes) == 3 {
		return path
	}

	return append(path, getPathElements(txIndex/2, hashes[(len(hashes)+1)/2:])...)
}

// GetTxMerklePath with merkle tree we calculate the merkle path for a given transaction.
func GetTxMerklePath(txIndex int, merkleTree []string) *MerklePathData {
	return &MerklePathData{
		Index: uint64(txIndex),
		Path:  getPathElements(txIndex, merkleTree),
	}
}