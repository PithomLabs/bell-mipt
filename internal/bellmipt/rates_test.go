package bellmipt

import (
	"testing"
)

func TestBellRatesNonnegative(t *testing.T) {
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
	psi := RandomNormalizedState(basis.Dim, 42)
	J := BellCurrent(H, psi)
	rates, stats := BellRates(J, psi)

	for n := 0; n < basis.Dim; n++ {
		for m := 0; m < basis.Dim; m++ {
			if rates[n][m] < 0 {
				t.Errorf("negative rate: rates[%d][%d] = %e", n, m, rates[n][m])
			}
		}
	}

	if stats.MaxNegativeRateViolation > 0 {
		t.Errorf("unexpected negative rate violation: %e", stats.MaxNegativeRateViolation)
	}
}

func TestBellRatesPositiveCurrentDirection(t *testing.T) {
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
	psi := RandomNormalizedState(basis.Dim, 42)
	J := BellCurrent(H, psi)
	rates, _ := BellRates(J, psi)

	for n := 0; n < basis.Dim; n++ {
		for m := 0; m < basis.Dim; m++ {
			if n == m {
				continue
			}
			if J[n][m] <= 0 && rates[n][m] != 0 {
				t.Errorf("rate[%d][%d]=%e should be 0 when J[%d][%d]=%e <= 0",
					n, m, rates[n][m], n, m, J[n][m])
			}
			if J[n][m] > 0 && rates[n][m] <= 0 {
				t.Errorf("rate[%d][%d]=%e should be positive when J[%d][%d]=%e > 0",
					n, m, rates[n][m], n, m, J[n][m])
			}
		}
	}
}

func TestBellRatesProbabilityFloorSafe(t *testing.T) {
	// Create a state where one component is near zero
	dim := 4
	psi := make([]complex128, dim)
	psi[0] = 1e-20 + 0i // near zero
	psi[1] = 0.5 + 0.3i
	psi[2] = 0.4 + 0.2i
	psi[3] = 0.3 + 0.1i

	// Normalize
	n := Norm(psi)
	for i := range psi {
		psi[i] /= complex(n, 0)
	}

	// Create a fake J matrix
	J := make([][]float64, dim)
	for i := range J {
		J[i] = make([]float64, dim)
	}
	J[1][0] = 0.1 // current from 0 to 1, but 0 has tiny probability

	rates, stats := BellRates(J, psi)

	// Rate from state 0 should be zero due to probability floor
	if rates[1][0] != 0 {
		t.Errorf("expected rate[1][0]=0 due to probability floor, got %e", rates[1][0])
	}

	if stats.ProbabilityFloorHits == 0 {
		t.Error("expected probability floor hit to be recorded")
	}
}
