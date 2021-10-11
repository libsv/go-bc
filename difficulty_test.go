package bc_test

import (
	"encoding/hex"
	"testing"

	"github.com/libsv/go-bc"
)

func TestDifficultyToHashratBSV(t *testing.T) {
	a := bc.DifficultyToHashrate("BSV", 22000, 7)
	b := bc.HumanHash(a)
	expected := "13.50 TH/s"
	if b != expected {
		t.Errorf("Failed to calculate hashrate, expected %s got %s", expected, b)
	}
}

func TestDifficultyToHashrateRSV(t *testing.T) {
	a := bc.DifficultyToHashrate("RSV", 22000, 7)
	b := bc.HumanHash(a)
	expected := "6.29 kH/s"
	if b != expected {
		t.Errorf("Failed to calculate hashrate, expected %s got %s", expected, b)
	}
}

func TestExpandTargetFrom_GenesisBlock(t *testing.T) {
	expected := "00000000ffff0000000000000000000000000000000000000000000000000000"
	got, _ := bc.ExpandTargetFrom("1d00ffff")

	if got != expected {
		t.Errorf("Expected result to be %s, got '%s", expected, got)
	}
}
func TestExpandTargetFrom(t *testing.T) {
	expected := "00000000000000002815ee000000000000000000000000000000000000000000"
	got, _ := bc.ExpandTargetFrom("182815ee")

	if got != expected {
		t.Errorf("Expected result to be %s, got '%s", expected, got)
	}
}
func TestExpandTargetFrom_InvalidBits(t *testing.T) {
	_, err := bc.ExpandTargetFrom("invalidBgolaits")
	if err == nil {
		t.Errorf("Expected an error to be thrown\n")
	}
}

// BenchmarkExpandTargetFrom-8   	 2000000	       667 ns/op	     224 B/op	       8 allocs/op
// BenchmarkExpandTargetFrom-8   	 5000000	       269 ns/op	     248 B/op	       6 allocs/op

func BenchmarkExpandTargetFrom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bc.ExpandTargetFrom("182815ee")
	}
}

func TestDifficultyFromBits(t *testing.T) {
	// genesis block should be difficulty 1
	testDifficulty("1d00ffff", float64(1), t)
	testDifficulty("1745fb53", float64(4.022059196164954e+12), t)
	testDifficulty("207fffff", float64(4.6565423739069247e-10), t)
}

func testDifficulty(bits string, expected float64, t *testing.T) {
	b, _ := hex.DecodeString(bits)
	d, _ := bc.DifficultyFromBits(b)

	if d != expected {
		t.Errorf("Expected difficulty of '%s' to be '%v', got %v", bits, expected, d)
	}
}

func TestValidBits(t *testing.T) {
	testValid := func(bits uint32, expected bool) {
		if bc.ValidBits(bits) != expected {
			t.Errorf("Expected ValidDifficulty(%x) to be %t", bits, expected)
		}
	}

	testValid(0x00000000, false)
	testValid(0x01000000, false)
	testValid(0xffffffff, false)
	testValid(0x00ffffff, false)
	testValid(0xff000000, false)
	testValid(0x01ff0000, false)
	testValid(0x017f0000, true)
	testValid(0x0100ff00, false)
	testValid(0x0200ff00, true)
	testValid(0x020000ff, false)
	testValid(0x030000ff, true)
	testValid(0x207f0000, true)
	testValid(0x217f0000, false)
	testValid(0x217fff00, false)
	testValid(0x2100ff00, true)
	testValid(0x2200ffff, false)
	testValid(0x220000ff, true)
	testValid(0x230000ff, false)

}
