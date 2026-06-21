package bellmipt

import (
	"math"
	"testing"
)

func TestBellCurrentAntisymmetric(t *testing.T) {
	// Build a small Hamiltonian and random state, verify J is antisymmetric
	for _, sites := range []int{2, 3, 4} {
		basis, err := NewBasis(sites)
		if err != nil {
			t.Fatal(err)
		}
		cfg := DefaultConfig()
		cfg.Sites = sites
		cfg.Boundary = "open"
		H, err := BuildKitaevHamiltonian(cfg, basis)
		if err != nil {
			t.Fatal(err)
		}
		psi := RandomNormalizedState(basis.Dim, 42)
		J := BellCurrent(H, psi)

		for n := 0; n < basis.Dim; n++ {
			for m := n + 1; m < basis.Dim; m++ {
				sum := J[n][m] + J[m][n]
				if math.Abs(sum) > 1e-12 {
					t.Errorf("sites=%d: J[%d][%d]+J[%d][%d] = %e, expected ~0",
						sites, n, m, m, n, sum)
				}
			}
		}
	}
}

func TestBellCurrentDiagonalZero(t *testing.T) {
	basis, err := NewBasis(3)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Sites = 3
	cfg.Boundary = "open"
	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		t.Fatal(err)
	}
	psi := RandomNormalizedState(basis.Dim, 99)
	J := BellCurrent(H, psi)

	for n := 0; n < basis.Dim; n++ {
		if math.Abs(J[n][n]) > 1e-12 {
			t.Errorf("J[%d][%d] = %e, expected ~0", n, n, J[n][n])
		}
	}
}
