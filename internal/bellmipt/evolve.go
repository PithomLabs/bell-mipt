package bellmipt

import (
	"math"
	"math/rand"
)

// RandomNormalizedState generates a deterministic random normalized state vector
// of the given dimension using the provided seed.
//
// Each complex component is generated from two independent normal random values:
//
//	psi[i] = complex(rng.NormFloat64(), rng.NormFloat64())
//
// The result is then normalized to unit L2 norm.
func RandomNormalizedState(dim int, seed int64) []complex128 {
	rng := rand.New(rand.NewSource(seed))
	psi := make([]complex128, dim)
	for i := range psi {
		re := rng.NormFloat64()
		im := rng.NormFloat64()
		psi[i] = complex(re, im)
	}

	// Normalize
	n := Norm(psi)
	for i := range psi {
		psi[i] /= complex(n, 0)
	}
	return psi
}

// DerivativeStats holds diagnostic statistics from a single derivative evaluation.
type DerivativeStats struct {
	RateStats RateStats
}

// StepStats holds diagnostic statistics accumulated over one RK4 step.
// Uses the stats from the first RK4 stage for simplicity.
type StepStats struct {
	DerivativeStats DerivativeStats
	TotalActivity   float64
}

// Derivative computes the simultaneous derivatives of ψ and ρ.
//
// For the Schrödinger equation:
//
//	dψ/dt = -i H ψ
//
// For the Bell master equation:
//
//	dρ_n/dt = Σ_m [σ(n←m) ρ_m - σ(m←n) ρ_n]
//
// The Bell rates σ are computed from the current ψ, ensuring consistent
// coupling at each RK4 sub-stage.
func Derivative(H Matrix, psi []complex128, rho []float64) (
	dpsi []complex128,
	drho []float64,
	stats DerivativeStats,
) {
	dim := H.Dim

	// 1. dψ = -i H ψ
	Hpsi := H.MatVec(psi)
	dpsi = make([]complex128, dim)
	for i := range dpsi {
		// -i * z = -i * (a + bi) = b - ai
		dpsi[i] = complex(imag(Hpsi[i]), -real(Hpsi[i]))
	}

	// 2. Compute Bell current J from ψ
	J := BellCurrent(H, psi)

	// 3. Compute Bell rates σ from J and ψ
	rates, rateStats := BellRates(J, psi)
	stats.RateStats = rateStats

	// 4. dρ using Bell master equation
	drho = make([]float64, dim)
	for n := 0; n < dim; n++ {
		var sumIn, sumOut float64
		for m := 0; m < dim; m++ {
			if m == n {
				continue
			}
			sumIn += rates[n][m] * rho[m]  // flow in: σ(n←m) ρ_m
			sumOut += rates[m][n] * rho[n] // flow out: σ(m←n) ρ_n
		}
		drho[n] = sumIn - sumOut
	}

	return dpsi, drho, stats
}

// RK4Step performs one step of the classic 4th-order Runge-Kutta method for the
// coupled (ψ, ρ) system.
//
// At each RK4 sub-stage, the Bell rates are recomputed from the stage wavefunction.
// This is the cleanest coupling strategy for the equivariance audit.
func RK4Step(H Matrix, psi []complex128, rho []float64, dt float64) (
	nextPsi []complex128,
	nextRho []float64,
	stepStats StepStats,
) {
	dim := H.Dim

	// Stage 1: k1 = f(ψ, ρ)
	k1psi, k1rho, stats1 := Derivative(H, psi, rho)
	stepStats.DerivativeStats = stats1

	// Stage 2: k2 = f(ψ + dt*k1ψ/2, ρ + dt*k1ρ/2)
	psi2 := make([]complex128, dim)
	rho2 := make([]float64, dim)
	halfDt := complex(dt/2, 0)
	for i := 0; i < dim; i++ {
		psi2[i] = psi[i] + halfDt*k1psi[i]
		rho2[i] = rho[i] + (dt/2)*k1rho[i]
	}
	k2psi, k2rho, _ := Derivative(H, psi2, rho2)

	// Stage 3: k3 = f(ψ + dt*k2ψ/2, ρ + dt*k2ρ/2)
	psi3 := make([]complex128, dim)
	rho3 := make([]float64, dim)
	for i := 0; i < dim; i++ {
		psi3[i] = psi[i] + halfDt*k2psi[i]
		rho3[i] = rho[i] + (dt/2)*k2rho[i]
	}
	k3psi, k3rho, _ := Derivative(H, psi3, rho3)

	// Stage 4: k4 = f(ψ + dt*k3ψ, ρ + dt*k3ρ)
	psi4 := make([]complex128, dim)
	rho4 := make([]float64, dim)
	fullDt := complex(dt, 0)
	for i := 0; i < dim; i++ {
		psi4[i] = psi[i] + fullDt*k3psi[i]
		rho4[i] = rho[i] + dt*k3rho[i]
	}
	k4psi, k4rho, _ := Derivative(H, psi4, rho4)

	// Combine: y_next = y + dt*(k1 + 2*k2 + 2*k3 + k4)/6
	nextPsi = make([]complex128, dim)
	nextRho = make([]float64, dim)
	sixth := complex(dt/6, 0)
	sixthR := dt / 6
	for i := 0; i < dim; i++ {
		nextPsi[i] = psi[i] + sixth*(k1psi[i]+2*complex128(k2psi[i])+2*complex128(k3psi[i])+k4psi[i])
		nextRho[i] = rho[i] + sixthR*(k1rho[i]+2*k2rho[i]+2*k3rho[i]+k4rho[i])
	}

	// Compute total Bell activity at end of step for diagnostics
	endJ := BellCurrent(H, nextPsi)
	endRates, _ := BellRates(endJ, nextPsi)
	stepStats.TotalActivity = TotalBellActivity(endRates, nextRho)

	return nextPsi, nextRho, stepStats
}

// L1Distance computes Σ|a[i] - b[i]|.
func L1Distance(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		sum += math.Abs(a[i] - b[i])
	}
	return sum
}

// CheckNaNInf returns true if any NaN or Inf is found in psi or rho.
func CheckNaNInf(psi []complex128, rho []float64) bool {
	for _, v := range psi {
		if math.IsNaN(real(v)) || math.IsNaN(imag(v)) ||
			math.IsInf(real(v), 0) || math.IsInf(imag(v), 0) {
			return true
		}
	}
	for _, v := range rho {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return true
		}
	}
	return false
}
