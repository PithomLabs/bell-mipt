package bellmipt

import (
	"math"
	"math/rand"
	"testing"
)

func TestSampleDiscrete(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	rates := []float64{0.0, 1.0, 0.0, 3.0} // Total 4.0

	counts := make(map[int]int)
	trials := 10000
	for i := 0; i < trials; i++ {
		idx := SampleDiscrete(rates, 4.0, rng)
		counts[idx]++
	}

	// Expect 1 to be ~25%, 3 to be ~75%
	freq1 := float64(counts[1]) / float64(trials)
	freq3 := float64(counts[3]) / float64(trials)

	if math.Abs(freq1-0.25) > 0.05 {
		t.Errorf("expected idx=1 to be ~0.25, got %v", freq1)
	}
	if math.Abs(freq3-0.75) > 0.05 {
		t.Errorf("expected idx=3 to be ~0.75, got %v", freq3)
	}
	if counts[0] != 0 || counts[2] != 0 {
		t.Errorf("sampled zero-rate indices")
	}
}

func TestStepBellConfiguration_NoJump(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	dim := 2
	psi := []complex128{complex(1.0, 0), complex(0.0, 0)} // |0> is occupied
	J := make([][]float64, dim)
	for i := range J {
		J[i] = make([]float64, dim)
	}
	// J[1][0] = 0.0 -> rate 0

	nextQ, lambdaDT, nearZero := StepBellConfiguration(0, psi, J, 0.1, rng)

	if nextQ != 0 {
		t.Errorf("expected no jump, got %d", nextQ)
	}
	if lambdaDT != 0.0 {
		t.Errorf("expected lambdaDT=0, got %v", lambdaDT)
	}
	if nearZero {
		t.Errorf("expected nearZero=false")
	}
}

func TestStepBellConfiguration_NearZeroFloor(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	dim := 2
	psi := []complex128{complex(1e-8, 0), complex(1.0, 0)} // prob(0) = 1e-16 < 1e-14
	J := make([][]float64, dim)
	for i := range J {
		J[i] = make([]float64, dim)
	}
	J[1][0] = 1.0 // Huge current, but prob is too low

	nextQ, lambdaDT, nearZero := StepBellConfiguration(0, psi, J, 0.1, rng)

	if nextQ != 0 {
		t.Errorf("expected no jump, got %d", nextQ)
	}
	if lambdaDT != 0.0 {
		t.Errorf("expected lambdaDT=0, got %v", lambdaDT)
	}
	if !nearZero {
		t.Errorf("expected nearZero=true")
	}
}

func TestRateOrientationDestinationSource(t *testing.T) {
	// Verify that outgoing rates correctly use the source column
	rng := rand.New(rand.NewSource(42)) // Deterministic
	dim := 3
	psi := []complex128{complex(math.Sqrt(0.5), 0), complex(math.Sqrt(0.5), 0), 0}

	J := make([][]float64, dim)
	for i := range J {
		J[i] = make([]float64, dim)
	}

	// Set J[dest][src]
	J[2][0] = 0.5  // Outgoing from 0 to 2
	J[1][0] = -0.5 // Incoming to 0 from 1 (negative outgoing)

	// We want to force a jump by having a huge dt, so p_jump is near 1
	// Then we should always jump to 2.
	dt := 100.0

	nextQ, lambdaDT, nearZero := StepBellConfiguration(0, psi, J, dt, rng)

	if nearZero {
		t.Fatal("unexpected nearZero")
	}
	// rate = 0.5 / 0.5 = 1.0
	// lambdaDT = 1.0 * 100.0 = 100.0
	if math.Abs(lambdaDT - 100.0) > 1e-9 {
		t.Errorf("expected lambdaDT=100.0, got %v", lambdaDT)
	}
	if nextQ != 2 {
		t.Errorf("expected jump to 2, got %d", nextQ)
	}
}
