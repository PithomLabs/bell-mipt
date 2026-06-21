package bellmipt

import (
	"fmt"
	"math/bits"
)

// Basis represents the Fock space basis for N fermionic sites.
// Each basis state is a uint64 bitstring where bit j set means site j is occupied.
// The basis state index equals the bitstring value, so the dimension is 2^N.
type Basis struct {
	Sites int
	Dim   int // 1 << Sites
}

// NewBasis creates a Basis for the given number of fermionic sites.
// It returns an error if sites < 1 or sites > 63 (we need at least one site,
// and uint64 limits us to 63 sites since 1<<63 overflows int on most platforms).
func NewBasis(sites int) (Basis, error) {
	if sites < 1 {
		return Basis{}, fmt.Errorf("bellmipt: number of sites must be >= 1, got %d", sites)
	}
	if sites > 63 {
		return Basis{}, fmt.Errorf("bellmipt: number of sites must be <= 63, got %d", sites)
	}
	return Basis{
		Sites: sites,
		Dim:   1 << uint(sites),
	}, nil
}

// Occupied reports whether site j is occupied in the given Fock state.
func Occupied(state uint64, site int) bool {
	return state&(1<<uint(site)) != 0
}

// CountBelow returns the number of occupied sites strictly below site j.
// This is the popcount of bits 0..j-1, used for the Jordan-Wigner sign.
func CountBelow(state uint64, site int) int {
	// Mask with bits 0 through site-1 set.
	// When site == 0, the mask is 0 and the count is 0.
	mask := uint64((1 << uint(site)) - 1)
	return bits.OnesCount64(state & mask)
}

// PopCount returns the total number of occupied sites in the given Fock state.
func PopCount(state uint64) int {
	return bits.OnesCount64(state)
}

// EnumerateStates returns all 2^N basis states in order [0, 1, ..., 2^N - 1].
func EnumerateStates(b Basis) []uint64 {
	states := make([]uint64, b.Dim)
	for i := range states {
		states[i] = uint64(i)
	}
	return states
}
