package bellmipt

// ProbabilityFloor is the minimum |ψ_m|² below which outgoing rates from
// state m are set to zero to avoid division by near-zero.
const ProbabilityFloor = 1e-14

// RhoNegativeTolerance is the tolerance for negative values in the ρ distribution.
// Values more negative than this indicate a problem.
const RhoNegativeTolerance = 1e-10

// RateStats holds diagnostic statistics from a Bell rate computation.
type RateStats struct {
	MaxNegativeRateViolation    float64 `json:"max_negative_rate_violation"`
	MeanTotalActivity           float64 `json:"mean_total_bell_activity"`
	MaxTotalActivity            float64 `json:"max_total_bell_activity"`
	ProbabilityFloorHits        int     `json:"probability_floor_hits"`
	ProbabilityFloorCurrentMass float64 `json:"probability_floor_current_mass"`
}

// BellRates computes the Bell jump rate matrix from the Bell current and wavefunction.
//
// The rate for transition n ← m is:
//
//	σ(n←m) = max(J[n][m], 0) / |ψ[m]|²
//
// When |ψ[m]|² < ProbabilityFloor, all outgoing rates from m are set to zero
// and a probability-floor hit is recorded.
//
// All rates are nonnegative by construction.
func BellRates(J [][]float64, psi []complex128) ([][]float64, RateStats) {
	dim := len(J)
	rates := make([][]float64, dim)
	for i := range rates {
		rates[i] = make([]float64, dim)
	}

	var stats RateStats
	prob := Probabilities(psi)

	for m := 0; m < dim; m++ {
		if prob[m] < ProbabilityFloor {
			// Record floor hit and track current mass that was suppressed
			stats.ProbabilityFloorHits++
			for n := 0; n < dim; n++ {
				if J[n][m] > 0 {
					stats.ProbabilityFloorCurrentMass += J[n][m]
				}
			}
			// All rates from m are zero (already initialized to 0)
			continue
		}

		for n := 0; n < dim; n++ {
			if n == m {
				continue
			}
			if J[n][m] > 0 {
				rates[n][m] = J[n][m] / prob[m]
			}
			// rates[n][m] stays 0 if J[n][m] <= 0
		}
	}

	// Compute activity statistics
	// Total activity = Σ_{n≠m} σ(n←m) * ρ_m, but we compute it per-state here
	// using just σ(n←m) since we don't have ρ at this level.
	// Instead, compute sum of all rates as a proxy.
	totalRate := 0.0
	for n := 0; n < dim; n++ {
		for m := 0; m < dim; m++ {
			if rates[n][m] < 0 {
				v := -rates[n][m]
				if v > stats.MaxNegativeRateViolation {
					stats.MaxNegativeRateViolation = v
				}
			}
			totalRate += rates[n][m]
		}
	}
	// We'll compute proper activity (with ρ weighting) in the audit layer

	return rates, stats
}

// TotalBellActivity computes Σ_{n≠m} σ(n←m) * ρ_m, the total jump activity.
func TotalBellActivity(rates [][]float64, rho []float64) float64 {
	dim := len(rho)
	total := 0.0
	for n := 0; n < dim; n++ {
		for m := 0; m < dim; m++ {
			if n != m {
				total += rates[n][m] * rho[m]
			}
		}
	}
	return total
}
