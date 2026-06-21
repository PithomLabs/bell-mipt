package bellmipt

import "math"

// EmpiricalTrajectoryEquivariance computes the L1 distance between the
// empirical distribution of sampled trajectories and the exact Born probabilities.
//
//	empirical(q,t) = count trajectories with Q(t)=q / total trajectories
//	born(q,t) = |ψ_q(t)|²
//	L1(t) = Σ_q |empirical(q,t) - born(q,t)|
//
// This is evaluated only at sampled time steps.
func EmpiricalTrajectoryEquivariance(trajectoryCounts map[int]int, totalTrajectories int, bornProbabilities []float64) float64 {
	if totalTrajectories == 0 {
		return 0.0
	}
	l1 := 0.0
	for q, prob := range bornProbabilities {
		empirical := float64(trajectoryCounts[q]) / float64(totalTrajectories)
		l1 += math.Abs(empirical - prob)
	}
	return l1
}
