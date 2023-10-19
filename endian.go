package bc

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/libsv/go-bt/v2"
)

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
