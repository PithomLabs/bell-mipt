package bellmipt

import "testing"

func TestGoalStatusPassed(t *testing.T) {
	a := &AuditAccumulator{}
	a.RecordHermitianError(0)
	a.RecordStep(StepAudit{
		NormError:                1e-12,
		RhoSumError:              1e-12,
		EquivarianceL1Error:      1e-8,
		RhoNegativeViolation:     0,
		CurrentAntisymmetryError: 1e-15,
		NaNOrInfDetected:         false,
		TotalBellActivity:        10.0,
	})

	cfg := AuditConfig{
		HermitianTolerance:    1e-10,
		NormTolerance:         1e-8,
		EquivarianceTolerance: 1e-5,
	}

	status := a.DetermineGoalStatus(cfg, true)
	if status != "toy_goal_passed" {
		t.Errorf("expected toy_goal_passed, got %s", status)
	}
}

func TestGoalStatusFailed(t *testing.T) {
	a := &AuditAccumulator{}
	a.RecordHermitianError(0)
	a.RecordStep(StepAudit{
		NormError:                1e-12,
		RhoSumError:              1e-12,
		EquivarianceL1Error:      0.1, // way too high
		RhoNegativeViolation:     0,
		CurrentAntisymmetryError: 1e-15,
		NaNOrInfDetected:         false,
	})

	cfg := AuditConfig{
		HermitianTolerance:    1e-10,
		NormTolerance:         1e-8,
		EquivarianceTolerance: 1e-5,
	}

	status := a.DetermineGoalStatus(cfg, true)
	if status != "toy_goal_failed" {
		t.Errorf("expected toy_goal_failed, got %s", status)
	}
}

func TestGoalStatusInconclusiveNaN(t *testing.T) {
	a := &AuditAccumulator{}
	a.RecordHermitianError(0)
	a.RecordStep(StepAudit{
		NaNOrInfDetected: true,
	})

	cfg := AuditConfig{
		HermitianTolerance:    1e-10,
		NormTolerance:         1e-8,
		EquivarianceTolerance: 1e-5,
	}

	status := a.DetermineGoalStatus(cfg, true)
	if status != "toy_goal_inconclusive" {
		t.Errorf("expected toy_goal_inconclusive, got %s", status)
	}
}

func TestGoalStatusInconclusiveSevereRhoNegativity(t *testing.T) {
	a := &AuditAccumulator{}
	a.RecordHermitianError(0)
	a.RecordStep(StepAudit{
		RhoNegativeViolation: 1e-5, // > 1e-6 threshold
	})

	cfg := AuditConfig{
		HermitianTolerance:    1e-10,
		NormTolerance:         1e-8,
		EquivarianceTolerance: 1e-5,
	}

	status := a.DetermineGoalStatus(cfg, true)
	if status != "toy_goal_inconclusive" {
		t.Errorf("expected toy_goal_inconclusive, got %s", status)
	}
}

func TestGoalStatusInconclusiveFloorHits(t *testing.T) {
	a := &AuditAccumulator{}
	a.RecordHermitianError(0)
	a.RecordStep(StepAudit{
		NormError:           1e-12,
		RhoSumError:         1e-12,
		EquivarianceL1Error: 1e-8,
		RateStats: RateStats{
			ProbabilityFloorHits:        5,
			ProbabilityFloorCurrentMass: 1e-10, // > 1e-12, significant
		},
	})

	cfg := AuditConfig{
		HermitianTolerance:    1e-10,
		NormTolerance:         1e-8,
		EquivarianceTolerance: 1e-5,
	}

	status := a.DetermineGoalStatus(cfg, true)
	if status != "toy_goal_inconclusive" {
		t.Errorf("expected toy_goal_inconclusive, got %s", status)
	}
}

func TestGoalStatusFloorHitsNotSignificant(t *testing.T) {
	a := &AuditAccumulator{}
	a.RecordHermitianError(0)
	a.RecordStep(StepAudit{
		NormError:                1e-12,
		RhoSumError:              1e-12,
		EquivarianceL1Error:      1e-8,
		CurrentAntisymmetryError: 1e-15,
		TotalBellActivity:        5.0,
		RateStats: RateStats{
			ProbabilityFloorHits:        2,
			ProbabilityFloorCurrentMass: 1e-14, // < 1e-12, not significant
		},
	})

	cfg := AuditConfig{
		HermitianTolerance:    1e-10,
		NormTolerance:         1e-8,
		EquivarianceTolerance: 1e-5,
	}

	status := a.DetermineGoalStatus(cfg, true)
	if status != "toy_goal_passed" {
		t.Errorf("expected toy_goal_passed despite floor hits, got %s", status)
	}
}

func TestGoalStatusFailedForbiddenLanguage(t *testing.T) {
	a := &AuditAccumulator{}
	a.RecordHermitianError(0)
	a.RecordStep(StepAudit{
		NormError:                1e-12,
		RhoSumError:              1e-12,
		EquivarianceL1Error:      1e-8,
		CurrentAntisymmetryError: 1e-15,
		TotalBellActivity:        5.0,
	})

	cfg := AuditConfig{
		HermitianTolerance:    1e-10,
		NormTolerance:         1e-8,
		EquivarianceTolerance: 1e-5,
	}

	status := a.DetermineGoalStatus(cfg, false) // forbidden language failed
	if status != "toy_goal_failed" {
		t.Errorf("expected toy_goal_failed when forbidden language fails, got %s", status)
	}
}
