package bellmipt

import (
	"math"
	"testing"
)

func TestConditionalVectorDimension(t *testing.T) {
	sites := 4
	split, _ := NewSiteSplit([]int{1, 3}, []int{0, 2}, sites)
	basis := make([]uint64, 1<<sites)
	for i := range basis {
		basis[i] = uint64(i)
	}

	proj := BuildConditionalProjector(split, basis)
	if len(proj.ByEnvironment) != 4 {
		t.Errorf("expected 4 environment configs, got %d", len(proj.ByEnvironment))
	}

	psi := make([]complex128, 1<<sites)
	psi[10] = 1.0 // subQ=3, envQ=0

	cv, err := EnvironmentProjectedConditionalVector(psi, 0, proj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cv.Vector) != 4 {
		t.Errorf("expected vector dimension 4, got %d", len(cv.Vector))
	}
	if cv.Vector[3] != 1.0 {
		t.Errorf("expected Vector[3] to be 1.0, got %v", cv.Vector[3])
	}
}

func TestConditionalProjectionPreservesPhase(t *testing.T) {
	sites := 3
	split, _ := NewSiteSplit([]int{0}, []int{1, 2}, sites)
	basis := make([]uint64, 1<<sites)
	for i := range basis {
		basis[i] = uint64(i)
	}

	proj := BuildConditionalProjector(split, basis)

	psi := make([]complex128, 1<<sites)
	psi[1] = complex(0, 1) // q=1 -> subQ=1, envQ=0

	cv, err := EnvironmentProjectedConditionalVector(psi, 0, proj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cv.Vector[1] != complex(0, 1) {
		t.Errorf("expected Vector[1] to be 0+1i, got %v", cv.Vector[1])
	}
}

func TestConditionalProjectorGroupsByEnvironment(t *testing.T) {
	sites := 3
	split, _ := NewSiteSplit([]int{1}, []int{0, 2}, sites)
	basis := make([]uint64, 1<<sites)
	for i := range basis {
		basis[i] = uint64(i)
	}

	proj := BuildConditionalProjector(split, basis)

	// q=0 (000) -> subQ=0, envQ=0
	// q=1 (001) -> subQ=0, envQ=1
	// q=2 (010) -> subQ=1, envQ=0
	// envQ=0 should have fullIndex 0 and 2.

	pairs := proj.ByEnvironment[0]
	if len(pairs) != 2 {
		t.Errorf("expected 2 pairs for envQ=0, got %d", len(pairs))
	}

	if pairs[0].FullIndex != 0 || pairs[0].SubIndex != 0 {
		t.Errorf("unexpected pair: %+v", pairs[0])
	}
	if pairs[1].FullIndex != 2 || pairs[1].SubIndex != 1 {
		t.Errorf("unexpected pair: %+v", pairs[1])
	}
}

func TestConditionalNormFailureThresholds(t *testing.T) {
	sites := 2
	split, _ := NewSiteSplit([]int{0}, []int{1}, sites)
	basis := make([]uint64, 1<<sites)
	for i := range basis {
		basis[i] = uint64(i)
	}

	proj := BuildConditionalProjector(split, basis)

	psi := make([]complex128, 4)
	psi[0] = complex(1e-13, 0) // Norm is below 1e-12

	cv, err := EnvironmentProjectedConditionalVector(psi, 0, proj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cv.Normalized {
		t.Errorf("expected vector to not be normalized")
	}

	if cv.Vector[0] != complex(1e-13, 0) {
		t.Errorf("expected unnormalized value, got %v", cv.Vector[0])
	}
}

func TestConditionalFidelity(t *testing.T) {
	a := []complex128{complex(1/math.Sqrt2, 0), complex(1/math.Sqrt2, 0)}
	b := []complex128{complex(1/math.Sqrt2, 0), complex(0, 1/math.Sqrt2)}

	fid, err := ConditionalFidelity(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// dot = 0.5 + 0.5i -> |dot|^2 = 0.25 + 0.25 = 0.5
	if math.Abs(fid-0.5) > 1e-12 {
		t.Errorf("expected fidelity 0.5, got %v", fid)
	}
}
