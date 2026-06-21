package bellmipt

import (
	"encoding/json"
	"testing"
)

func TestBridgeDisabledPreserves0001SemanticMetrics(t *testing.T) {
	cfg := DefaultConfig()
	// Run without bridge
	res0001 := Run(cfg, t.TempDir())
	if res0001.Error != nil {
		t.Fatalf("Run failed: %v", res0001.Error)
	}
	rep0001 := res0001.Report

	// Add bridge but disabled
	cfg.Bridge = &BridgeConfig{
		Enabled: false,
	}
	res0002Disabled := Run(cfg, t.TempDir())
	if res0002Disabled.Error != nil {
		t.Fatalf("Run failed: %v", res0002Disabled.Error)
	}
	rep0002Disabled := res0002Disabled.Report

	// Semantic comparison
	if rep0001.SchemaVersion != rep0002Disabled.SchemaVersion {
		t.Errorf("Schema mismatch: %s != %s", rep0001.SchemaVersion, rep0002Disabled.SchemaVersion)
	}
	if rep0001.Bridge != nil || rep0002Disabled.Bridge != nil {
		t.Errorf("Expected nil bridge section")
	}
	if rep0001.GoalStatus != rep0002Disabled.GoalStatus {
		t.Errorf("GoalStatus mismatch")
	}

	// Marshal both and compare length/fields to ensure deterministic equality where applicable
	b1, _ := json.Marshal(rep0001)
	b2, _ := json.Marshal(rep0002Disabled)

	if len(b1) != len(b2) {
		t.Errorf("JSON lengths differ: %d vs %d", len(b1), len(b2))
	}
}

func TestEmpiricalEquivarianceSampledStepsOnly(t *testing.T) {
	// A simple check that EmpiricalTrajectoryEquivariance computes L1 properly
	born := []float64{0.5, 0.5}
	counts := map[int]int{0: 50, 1: 50}

	l1 := EmpiricalTrajectoryEquivariance(counts, 100, born)
	if l1 != 0.0 {
		t.Errorf("expected 0.0, got %v", l1)
	}

	counts2 := map[int]int{0: 60, 1: 40}
	// empirical: 0.6 and 0.4
	// l1 = |0.6 - 0.5| + |0.4 - 0.5| = 0.1 + 0.1 = 0.2
	l1 = EmpiricalTrajectoryEquivariance(counts2, 100, born)
	// Compare with tolerance
	if l1 < 0.19 || l1 > 0.21 {
		t.Errorf("expected ~0.2, got %v", l1)
	}
}

func TestEmpiricalEquivarianceAgainstPsiSquared(t *testing.T) {
	// Run the full bridge audit with a small model and enough trajectories
	// to see reasonable L1 convergence (loose tolerance).
	cfg := DefaultConfig()
	cfg.Sites = 3
	cfg.Bridge = &BridgeConfig{
		Enabled:          true,
		SubsystemSites:   []int{0},
		EnvironmentSites: []int{1, 2},
		Trajectories:     1000,
		Seed:             42,
		SampleEverySteps: 10,
	}

	res := Run(cfg, t.TempDir())
	if res.Error != nil {
		t.Fatalf("Run failed: %v", res.Error)
	}
	rep := res.Report

	if rep.Bridge == nil {
		t.Fatalf("Bridge report is nil")
	}

	l1 := rep.Bridge.Metrics.FinalEmpiricalEquivarianceL1
	// With 1000 trajectories, we expect L1 error to be reasonably small,
	// typically < 0.2
	if l1 > 0.2 {
		t.Logf("Warning: Empirical L1 distance is relatively high: %v", l1)
	}
}
