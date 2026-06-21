package bellmipt

import "math"

// AuditAccumulator accumulates per-step audit metrics over the full evolution.
type AuditAccumulator struct {
	MaxHermitianError           float64
	MaxNormError                float64
	MaxRhoSumError              float64
	MaxRhoNegativeViolation     float64
	MaxCurrentAntisymmetryError float64
	MaxRateNegativeViolation    float64
	MaxEquivarianceL1Error      float64
	FinalEquivarianceL1Error    float64
	SumEquivarianceL1Error      float64
	MeanTotalBellActivity       float64
	MaxTotalBellActivity        float64
	SumTotalBellActivity        float64
	NaNOrInfDetected            bool
	TotalProbabilityFloorHits   int
	TotalFloorCurrentMass       float64
	StepCount                   int
}

// StepAudit holds per-step audit results.
type StepAudit struct {
	NormError                float64
	RhoSumError              float64
	EquivarianceL1Error      float64
	RhoNegativeViolation     float64
	CurrentAntisymmetryError float64
	NaNOrInfDetected         bool
	TotalBellActivity        float64
	RateStats                RateStats
}

// RecordHermitianError records the initial Hermiticity check result.
func (a *AuditAccumulator) RecordHermitianError(err float64) {
	a.MaxHermitianError = err
}

// RecordStep records one evolution step's audit results.
func (a *AuditAccumulator) RecordStep(s StepAudit) {
	a.StepCount++

	if s.NormError > a.MaxNormError {
		a.MaxNormError = s.NormError
	}
	if s.RhoSumError > a.MaxRhoSumError {
		a.MaxRhoSumError = s.RhoSumError
	}
	if s.RhoNegativeViolation > a.MaxRhoNegativeViolation {
		a.MaxRhoNegativeViolation = s.RhoNegativeViolation
	}
	if s.CurrentAntisymmetryError > a.MaxCurrentAntisymmetryError {
		a.MaxCurrentAntisymmetryError = s.CurrentAntisymmetryError
	}
	if s.EquivarianceL1Error > a.MaxEquivarianceL1Error {
		a.MaxEquivarianceL1Error = s.EquivarianceL1Error
	}
	a.FinalEquivarianceL1Error = s.EquivarianceL1Error
	a.SumEquivarianceL1Error += s.EquivarianceL1Error

	if s.RateStats.MaxNegativeRateViolation > a.MaxRateNegativeViolation {
		a.MaxRateNegativeViolation = s.RateStats.MaxNegativeRateViolation
	}

	if s.TotalBellActivity > a.MaxTotalBellActivity {
		a.MaxTotalBellActivity = s.TotalBellActivity
	}
	a.SumTotalBellActivity += s.TotalBellActivity

	if s.NaNOrInfDetected {
		a.NaNOrInfDetected = true
	}

	a.TotalProbabilityFloorHits += s.RateStats.ProbabilityFloorHits
	a.TotalFloorCurrentMass += s.RateStats.ProbabilityFloorCurrentMass
}

// MeanEquivarianceL1Error returns the mean equivariance error.
func (a *AuditAccumulator) MeanEquivarianceL1Error() float64 {
	if a.StepCount == 0 {
		return 0
	}
	return a.SumEquivarianceL1Error / float64(a.StepCount)
}

// MeanBellActivity returns the mean total Bell activity.
func (a *AuditAccumulator) MeanBellActivity() float64 {
	if a.StepCount == 0 {
		return 0
	}
	return a.SumTotalBellActivity / float64(a.StepCount)
}

// CurrentAntisymmetryTolerance returns max(hermitianTolerance, 1e-12).
func CurrentAntisymmetryTolerance(hermitianTolerance float64) float64 {
	return math.Max(hermitianTolerance, 1e-12)
}

// DetermineGoalStatus determines the final toy goal status from accumulated audit data.
//
// Returns one of:
//   - "toy_goal_inconclusive" if NaN/Inf detected, significant probability-floor hits,
//     or severe rho negativity (> 1e-6)
//   - "toy_goal_passed" if all checks pass within configured tolerances
//   - "toy_goal_failed" otherwise
func (a *AuditAccumulator) DetermineGoalStatus(cfg AuditConfig, forbiddenPassed bool) string {
	// Inconclusive conditions
	if a.NaNOrInfDetected {
		return "toy_goal_inconclusive"
	}
	if a.TotalProbabilityFloorHits > 0 && a.TotalFloorCurrentMass > 1e-12 {
		return "toy_goal_inconclusive"
	}
	if a.MaxRhoNegativeViolation > 1e-6 {
		return "toy_goal_inconclusive"
	}

	// Pass conditions — all must be true
	currentTol := CurrentAntisymmetryTolerance(cfg.HermitianTolerance)

	pass := true
	pass = pass && a.MaxHermitianError <= cfg.HermitianTolerance
	pass = pass && a.MaxNormError <= cfg.NormTolerance
	pass = pass && a.MaxRhoSumError <= cfg.NormTolerance
	pass = pass && a.MaxCurrentAntisymmetryError <= currentTol
	pass = pass && a.MaxRateNegativeViolation <= 0
	pass = pass && a.MaxEquivarianceL1Error <= cfg.EquivarianceTolerance
	pass = pass && a.MaxRhoNegativeViolation <= RhoNegativeTolerance
	pass = pass && forbiddenPassed

	if pass {
		return "toy_goal_passed"
	}
	return "toy_goal_failed"
}

// ComputeStepAudit computes the per-step audit from the current state.
func ComputeStepAudit(H Matrix, psi []complex128, rho []float64, stepStats StepStats) StepAudit {
	dim := len(psi)
	born := Probabilities(psi)

	// Norm error: |‖ψ‖² - 1|
	normSq := 0.0
	for _, v := range psi {
		normSq += real(v)*real(v) + imag(v)*imag(v)
	}
	normErr := math.Abs(normSq - 1.0)

	// Rho sum error
	rhoSum := 0.0
	for _, r := range rho {
		rhoSum += r
	}
	rhoSumErr := math.Abs(rhoSum - 1.0)

	// Rho negative violation
	maxNeg := 0.0
	for _, r := range rho {
		if r < 0 && -r > maxNeg {
			maxNeg = -r
		}
	}

	// Equivariance L1 error
	l1 := L1Distance(rho, born)

	// Current antisymmetry
	J := BellCurrent(H, psi)
	antisymErr := CurrentAntisymmetryError(J)

	// NaN/Inf check
	nanInf := CheckNaNInf(psi, rho)

	// Rate stats from the step
	_, rateStats := BellRates(J, psi)

	_ = dim // suppress unused

	return StepAudit{
		NormError:                normErr,
		RhoSumError:              rhoSumErr,
		EquivarianceL1Error:      l1,
		RhoNegativeViolation:     maxNeg,
		CurrentAntisymmetryError: antisymErr,
		NaNOrInfDetected:         nanInf,
		TotalBellActivity:        stepStats.TotalActivity,
		RateStats:                rateStats,
	}
}
