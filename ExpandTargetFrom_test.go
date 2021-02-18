package bc_test

import (
	"testing"

	"github.com/libsv/go-bc"
)

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
