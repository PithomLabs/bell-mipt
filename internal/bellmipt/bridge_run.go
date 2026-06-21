package bellmipt

import (
	"fmt"
	"math/rand"
)

// RunBridgeAudit executes Pass 2 of the BELL-MIPT-0002A implementation.
// It uses the psi snapshots from Pass 1 to sample Bell trajectories and compute
// the environment-projected conditional vectors.
func RunBridgeAudit(
	cfg *BridgeConfig,
	basis Basis,
	H Matrix,
	psiSnapshots [][]complex128,
	dt float64,
) *BridgeReport {
	// Initialize report and metrics
	report := &BridgeReport{
		Enabled:                   true,
		BridgeGoal:                "sample_bell_trajectories_and_audit_environment_projected_conditional_vectors",
		BridgeStatus:              "bridge_audit_completed", // Assume success initially
		SubsystemSitesRequested:   cfg.SubsystemSites,
		EnvironmentSitesRequested: cfg.EnvironmentSites,
		SubsystemSitesCanonical:   cfg.CanonicalSubsystemSites(),
		EnvironmentSitesCanonical: cfg.CanonicalEnvironmentSites(),
		Trajectories:              cfg.Trajectories,
		SampleEverySteps:          cfg.SampleEverySteps,
		Metrics:                   &BridgeMetrics{},
	}

	split, err := NewSiteSplit(cfg.SubsystemSites, cfg.EnvironmentSites, basis.Sites)
	if err != nil {
		report.BridgeStatus = "bridge_audit_failed"
		report.Warnings = append(report.Warnings, BridgeWarning{Code: "invalid_bridge_config", Message: err.Error()})
		return report
	}

	// Basis configurations
	configs := EnumerateStates(basis)
	projector := BuildConditionalProjector(split, configs)

	rng := rand.New(rand.NewSource(cfg.Seed))
	acc := BridgeAccumulator{
		Trajectories: cfg.Trajectories,
	}

	steps := len(psiSnapshots) - 1

	// For empirical equivariance tracking
	empiricalCounts := make([]map[int]int, steps+1)
	for i := range empiricalCounts {
		empiricalCounts[i] = make(map[int]int)
	}

	for traj := 0; traj < cfg.Trajectories; traj++ {
		// Sample initial configuration based on |psi[0]|^2
		prob0 := Probabilities(psiSnapshots[0])
		qIndex := SampleDiscrete(prob0, 1.0, rng)

		empiricalCounts[0][qIndex]++

		// Track conditional vector state for fidelity drops
		qConfig := configs[qIndex]
		_, envConfig := SplitConfig(qConfig, split)

		prevCondVec, err := EnvironmentProjectedConditionalVector(psiSnapshots[0], envConfig, projector)
		if err != nil {
			report.BridgeStatus = "bridge_audit_failed"
			report.Warnings = append(report.Warnings, BridgeWarning{Code: "structural_error", Message: err.Error()})
			return report
		}

		// Trajectory integration
		for k := 0; k < steps; k++ {
			psiK := psiSnapshots[k]
			J := BellCurrent(H, psiK)

			nextQIndex, lambdaDT, nearZero := StepBellConfiguration(qIndex, psiK, J, dt, rng)

			if nearZero {
				acc.NearZeroProbabilityCount++
			}
			if lambdaDT > acc.MaxLambdaDT {
				acc.MaxLambdaDT = lambdaDT
			}

			qIndex = nextQIndex
			empiricalCounts[k+1][qIndex]++

			// Conditional vector logic at sampled intervals
			if (k+1)%cfg.SampleEverySteps == 0 {
				nextConfig := configs[qIndex]
				_, nextEnvConfig := SplitConfig(nextConfig, split)

				currCondVec, _ := EnvironmentProjectedConditionalVector(psiSnapshots[k+1], nextEnvConfig, projector)

				// Compute fidelity drop if both are normalized
				if prevCondVec.Normalized && currCondVec.Normalized {
					fidelity, _ := ConditionalFidelity(prevCondVec.Vector, currCondVec.Vector)
					fidelityDrop := 1.0 - fidelity

					// Clamp tiny numerical drift
					if fidelityDrop < 0 && fidelityDrop > -1e-12 {
						fidelityDrop = 0.0
					}

					class := ClassifyJump(qConfig, nextConfig, split)
					acc.RecordTransition(class, fidelityDrop, false)
				} else {
					acc.RecordTransition(NoJump, 0.0, true) // Records a norm failure
				}

				prevCondVec = currCondVec
				qConfig = nextConfig
			}
		}
	}

	// Compute final empirical equivariance
	born0 := Probabilities(psiSnapshots[0])
	bornFinal := Probabilities(psiSnapshots[steps])
	acc.InitialL1 = EmpiricalTrajectoryEquivariance(empiricalCounts[0], cfg.Trajectories, born0)
	acc.FinalL1 = EmpiricalTrajectoryEquivariance(empiricalCounts[steps], cfg.Trajectories, bornFinal)

	maxL1 := 0.0
	for k := 0; k <= steps; k += cfg.SampleEverySteps {
		l1 := EmpiricalTrajectoryEquivariance(empiricalCounts[k], cfg.Trajectories, Probabilities(psiSnapshots[k]))
		if l1 > maxL1 {
			maxL1 = l1
		}
	}
	acc.MaxL1 = maxL1

	*report.Metrics = acc.FinalizeMetrics()

	// Assign interpretations and handle warnings
	interp := InterpretConditionalUpdate(report.Metrics.ConditionalUpdateRatio, report.BridgeStatus)
	report.Interpretation = &interp

	// Apply statuses and warnings based on required thresholds
	if acc.MaxLambdaDT > LambdaDTWarning {
		report.Warnings = append(report.Warnings, BridgeWarning{Code: "large_lambda_dt_warning", Message: fmt.Sprintf("max_lambda_dt=%.4f > 0.05", acc.MaxLambdaDT)})
	}
	if acc.MaxLambdaDT > LambdaDTInconclusive {
		report.BridgeStatus = "bridge_audit_inconclusive"
		report.Warnings = append(report.Warnings, BridgeWarning{Code: "severe_lambda_dt_inconclusive", Message: fmt.Sprintf("max_lambda_dt=%.4f > 0.20", acc.MaxLambdaDT)})
	}

	totalTransitions := acc.EventCountAnyJump + acc.EventCountNoJump
	if totalTransitions > 0 && float64(acc.ConditionalNormFailures)/float64(totalTransitions) > 0.10 {
		report.BridgeStatus = "bridge_audit_inconclusive"
		report.Warnings = append(report.Warnings, BridgeWarning{Code: "conditional_norm_failures", Message: "norm failure rate > 10%"})
	}

	totalMicrosteps := cfg.Trajectories * steps
	if float64(acc.NearZeroProbabilityCount)/float64(totalMicrosteps) > 0.01 {
		report.BridgeStatus = "bridge_audit_inconclusive"
		report.Warnings = append(report.Warnings, BridgeWarning{Code: "near_zero_current_configuration_probability", Message: "near zero probability rate > 1%"})
	} else if acc.NearZeroProbabilityCount > 0 {
		report.Warnings = append(report.Warnings, BridgeWarning{Code: "near_zero_current_configuration_probability", Message: "near zero configuration probability encountered"})
	}

	if acc.EventCountStrictEnvironment == 0 {
		report.BridgeStatus = "bridge_audit_inconclusive"
		report.Warnings = append(report.Warnings, BridgeWarning{Code: "low_strict_environment_event_count", Message: "strict_environment_jump_transitions == 0"})
	}
	if acc.EventCountNoJump == 0 {
		report.BridgeStatus = "bridge_audit_inconclusive"
		report.Warnings = append(report.Warnings, BridgeWarning{Code: "low_no_jump_event_count", Message: "no_jump_transitions == 0"})
	}

	report.Warnings = append(report.Warnings, BridgeWarning{Code: "jordan_wigner_partition_convention", Message: "Conditional vectors are projected in the canonical occupation basis. No additional Jordan-Wigner sign is applied during projection; all phases are inherited from the full wavefunction amplitudes."})

	if acc.TotalBoundaryCrossingJumps > 0 {
		report.Warnings = append(report.Warnings, BridgeWarning{Code: "boundary_crossing_jumps_observed", Message: "boundary_crossing jumps occurred"})
	}

	return report
}
