package bc_test

import (
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
