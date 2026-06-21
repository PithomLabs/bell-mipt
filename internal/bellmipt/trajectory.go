package bellmipt

import (
	"math"
	"math/rand"
)

const RateProbabilityFloor = 1e-14
const LambdaDTWarning = 0.05
const LambdaDTInconclusive = 0.20

// StepBellConfiguration performs one discrete-time step for a Bell trajectory.
// It takes the current configuration index (qIndex), the wavefunction at the
// beginning of the step (psi), the Bell probability current matrix (J), the
// time step (dt), and a random number generator.
//
// It returns the next configuration index (nextQIndex), the computed lambda * dt,
// and a boolean indicating whether the probability of the current configuration
// was below the floor (nearZero).
func StepBellConfiguration(
	qIndex int,
	psi []complex128,
	J [][]float64,
	dt float64,
	rng *rand.Rand,
) (nextQIndex int, lambdaDT float64, nearZero bool) {
	prob := real(psi[qIndex])*real(psi[qIndex]) + imag(psi[qIndex])*imag(psi[qIndex])

	if prob < RateProbabilityFloor {
		return qIndex, 0.0, true // nearZero = true, no jump
	}

	dim := len(psi)
	var lambda float64
	rates := make([]float64, dim)

	// Compute outgoing rates from qIndex.
	// Rate orientation: J[dest][src]
	for dest := 0; dest < dim; dest++ {
		if dest == qIndex {
			continue
		}
		if J[dest][qIndex] > 0 {
			rate := J[dest][qIndex] / prob
			rates[dest] = rate
			lambda += rate
		}
	}

	lambdaDT = lambda * dt

	// Discrete-time thinning: p_jump = 1 - exp(-lambda * dt)
	pJump := 1.0 - math.Exp(-lambdaDT)

	// Sample whether a jump occurs
	if rng.Float64() < pJump {
		// Jump occurs! Sample destination based on relative rates.
		nextQIndex = SampleDiscrete(rates, lambda, rng)
		return nextQIndex, lambdaDT, false
	}

	// No jump
	return qIndex, lambdaDT, false
}

// SampleDiscrete samples an index from a slice of unnormalized weights (rates).
// It assumes totalRate is the sum of all elements in rates.
func SampleDiscrete(rates []float64, totalRate float64, rng *rand.Rand) int {
	if totalRate <= 0 {
		return 0 // Fallback, shouldn't happen if pJump triggered
	}

	r := rng.Float64() * totalRate
	var accum float64
	for i, w := range rates {
		if w > 0 {
			accum += w
			if r <= accum {
				return i
			}
		}
	}
	// Fallback to last non-zero
	for i := len(rates) - 1; i >= 0; i-- {
		if rates[i] > 0 {
			return i
		}
	}
	return 0
}
