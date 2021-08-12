package bc

import (
	"encoding/hex"

	"github.com/libsv/go-bt"
)

/*
Field 													Purpose 									 														Size (Bytes)
----------------------------------------------------------------------------------------------------
Version 							Block version number 																									4
hashPrevBlock 				256-bit hash of the previous block header 	 													32
hashMerkleRoot 				256-bit hash based on all of the transactions in the block 	 					32
Time 									Current block timestamp as seconds since 1970-01-01T00:00 UTC 				4
Bits 									Current target in compact format 	 																		4
Nonce 								32-bit number (starts at 0) 	 																				4
*/

// A Block in the Bitcoin blockchain.
type Block struct {
	BlockHeader *BlockHeader
	Txs         []*bt.Tx
}

// TODO: make fields private and make getters and setters

// String returns the Block Header encoded as hex string.
func (b *Block) String() (string, error) {
	bb, err := b.Bytes()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bb), nil
}

// Bytes will decode a bitcoin block struct into a byte slice.
//
// See https://btcinformation.org/en/developer-reference#serialized-blocks
func (b *Block) Bytes() ([]byte, error) {
	bytes := []byte{}

	bh, err := b.BlockHeader.Bytes()
	if err != nil {
		return nil, err
	}
	bytes = append(bytes, bh...)

	txCount := uint64(len(b.Txs))
	bytes = append(bytes, bt.VarInt(txCount)...)

	for _, tx := range b.Txs {
		bytes = append(bytes, tx.ToBytes()...)
	}

	return bytes, nil
}

// EncodeBlockStr will encode a block header hash
// into the bitcoin block header structure.
//
// See https://btcinformation.org/en/developer-reference#serialized-blocks
func EncodeBlockStr(blockStr string) (*Block, error) {
	blockBytes, err := hex.DecodeString(blockStr)
	if err != nil {
		return nil, err
	}

	return EncodeBlock(blockBytes)
}

// EncodeBlock will encode a block header byte slice
// into the bitcoin block header structure.
//
// See https://btcinformation.org/en/developer-reference#serialized-blocks
func EncodeBlock(b []byte) (*Block, error) {

	var offset int
	bh, err := EncodeBlockHeader(b[:80])
	if err != nil {
		return nil, err
	}
	offset += 80

	txCount, size := bt.DecodeVarInt(b[offset:])
	offset += size

	var txs []*bt.Tx

	for i := 0; i < int(txCount); i++ {
		tx, size, err := bt.NewTxFromStream(b[offset:])
		if err != nil {
			return nil, err
		}

		txs = append(txs, tx)
		offset += size
	}

	return &Block{
		BlockHeader: bh,
		Txs:         txs,
	}, nil
}
