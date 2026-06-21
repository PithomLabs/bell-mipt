package bellmipt

import (
	"math"
	"testing"
)

func TestNormPreservedSmallSystem(t *testing.T) {
	// Evolve a small system and check that ||ψ||² stays near 1
	basis, err := NewBasis(3)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Sites = 3
	cfg.Boundary = "open"
	cfg.Time.Dt = 0.01
	cfg.Time.Steps = 100

	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		t.Fatal(err)
	}

	psi := RandomNormalizedState(basis.Dim, 42)
	rho := Probabilities(psi)

	maxNormErr := 0.0
	for step := 0; step < cfg.Time.Steps; step++ {
		psi, rho, _ = RK4Step(H, psi, rho, cfg.Time.Dt)
		normErr := math.Abs(Norm(psi)*Norm(psi) - 1.0)
		if normErr > maxNormErr {
			maxNormErr = normErr
		}
	}

	if maxNormErr > 1e-8 {
		t.Errorf("max norm error %e exceeds tolerance 1e-8", maxNormErr)
	}
	t.Logf("max norm error after %d steps: %e", cfg.Time.Steps, maxNormErr)
}

func TestRhoSumPreservedSmallSystem(t *testing.T) {
	basis, err := NewBasis(3)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Sites = 3
	cfg.Boundary = "open"
	cfg.Time.Dt = 0.01
	cfg.Time.Steps = 100

	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		t.Fatal(err)
	}

	psi := RandomNormalizedState(basis.Dim, 42)
	rho := Probabilities(psi)

	maxSumErr := 0.0
	for step := 0; step < cfg.Time.Steps; step++ {
		psi, rho, _ = RK4Step(H, psi, rho, cfg.Time.Dt)
		sum := 0.0
		for _, r := range rho {
			sum += r
		}
		sumErr := math.Abs(sum - 1.0)
		if sumErr > maxSumErr {
			maxSumErr = sumErr
		}
	}

	if maxSumErr > 1e-8 {
		t.Errorf("max rho sum error %e exceeds tolerance 1e-8", maxSumErr)
	}
	t.Logf("max rho sum error after %d steps: %e", cfg.Time.Steps, maxSumErr)
}

func TestEquivarianceAuditSmallSystem(t *testing.T) {
	// The key test: ρ(t) should track |ψ(t)|²
	basis, err := NewBasis(3)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Sites = 3
	cfg.Boundary = "open"
	cfg.Time.Dt = 0.001
	cfg.Time.Steps = 500

	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		t.Fatal(err)
	}

	psi := RandomNormalizedState(basis.Dim, 42)
	rho := Probabilities(psi)

	maxL1 := 0.0
	for step := 0; step < cfg.Time.Steps; step++ {
		psi, rho, _ = RK4Step(H, psi, rho, cfg.Time.Dt)
		born := Probabilities(psi)
		l1 := L1Distance(rho, born)
		if l1 > maxL1 {
			maxL1 = l1
		}
	}

	if maxL1 > 1e-5 {
		t.Errorf("equivariance L1 error %e exceeds 1e-5", maxL1)
	}
	t.Logf("max equivariance L1 error after %d steps: %e", cfg.Time.Steps, maxL1)
}

func TestRhoTracksBornDistribution(t *testing.T) {
	// Verify at several intermediate checkpoints
	basis, err := NewBasis(2)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Sites = 2
	cfg.Boundary = "open"
	cfg.Time.Dt = 0.001
	cfg.Time.Steps = 200

	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		t.Fatal(err)
	}

	psi := RandomNormalizedState(basis.Dim, 99)
	rho := Probabilities(psi)

	checkpoints := map[int]bool{50: true, 100: true, 150: true, 200: true}

	for step := 1; step <= cfg.Time.Steps; step++ {
		psi, rho, _ = RK4Step(H, psi, rho, cfg.Time.Dt)
		if checkpoints[step] {
			born := Probabilities(psi)
			l1 := L1Distance(rho, born)
			if l1 > 1e-8 {
				t.Errorf("step %d: equivariance L1 error %e exceeds 1e-8", step, l1)
			}
		}
	}
}

func TestNoNaNInfDuringEvolution(t *testing.T) {
	basis, err := NewBasis(3)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Sites = 3
	cfg.Boundary = "periodic"
	cfg.Time.Dt = 0.001
	cfg.Time.Steps = 200

	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		t.Fatal(err)
	}

	psi := RandomNormalizedState(basis.Dim, 42)
	rho := Probabilities(psi)

	for step := 0; step < cfg.Time.Steps; step++ {
		psi, rho, _ = RK4Step(H, psi, rho, cfg.Time.Dt)
		if CheckNaNInf(psi, rho) {
			t.Fatalf("NaN/Inf detected at step %d", step)
		}
	}
}
