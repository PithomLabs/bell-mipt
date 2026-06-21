package bellmipt

import "testing"

func TestJumpClassification(t *testing.T) {
	// Let's use a simple fake SplitConfig via a mock SiteSplit
	// Assuming sites 0,1 are subsystem and 2,3 are environment.
	// We won't actually call SplitConfig, we'll mock the classification
	// but since we need to test ClassifyJump we need SplitConfig to work.
	// Wait, SplitConfig is implemented in site_split.go.
	// We'll construct a valid SiteSplit.
	split, _ := NewSiteSplit([]int{0, 1}, []int{2, 3}, 4)

	// 0b0000 -> 0b0000: No jump
	if c := ClassifyJump(0, 0, split); c != NoJump {
		t.Errorf("Expected NoJump, got %v", c)
	}

	// 0b0000 -> 0b0100 (Env changed: site 2)
	if c := ClassifyJump(0, 4, split); c != StrictEnvironmentJump {
		t.Errorf("Expected StrictEnvironmentJump, got %v", c)
	}

	// 0b0000 -> 0b0001 (Sub changed: site 0)
	if c := ClassifyJump(0, 1, split); c != StrictSubsystemJump {
		t.Errorf("Expected StrictSubsystemJump, got %v", c)
	}

	// 0b0000 -> 0b0101 (Both changed: site 0 and 2)
	if c := ClassifyJump(0, 5, split); c != BoundaryCrossingJump {
		t.Errorf("Expected BoundaryCrossingJump, got %v", c)
	}
}

func TestConditionalUpdateRatioNullWhenLowEvents(t *testing.T) {
	acc := BridgeAccumulator{
		EventCountStrictEnvironment:      5, // < 10
		EventCountNoJump:                 20,
		SumFidelityDropStrictEnvironment: 0.5,
		SumFidelityDropNoJump:            0.2,
	}

	m := acc.FinalizeMetrics()
	if m.ConditionalUpdateRatio != nil {
		t.Errorf("Expected ratio to be nil, got %v", *m.ConditionalUpdateRatio)
	}
	if m.ConditionalUpdateRatioStatus != "unavailable_insufficient_events" {
		t.Errorf("Expected unavailable status, got %v", m.ConditionalUpdateRatioStatus)
	}

	// Now with >= 10
	acc.EventCountStrictEnvironment = 10
	m2 := acc.FinalizeMetrics()
	if m2.ConditionalUpdateRatio == nil {
		t.Errorf("Expected ratio to be calculated")
	} else {
		// ratio = (0.5/10) / (0.2/20) = 0.05 / 0.01 = 5.0
		if *m2.ConditionalUpdateRatio != 5.0 {
			t.Errorf("Expected ratio 5.0, got %v", *m2.ConditionalUpdateRatio)
		}
	}
}

func TestBoundaryCrossingJumpExcludedFromPrimaryRatio(t *testing.T) {
	acc := BridgeAccumulator{
		EventCountStrictEnvironment: 10,
		EventCountNoJump:            10,
		EventCountBoundaryCrossing:  10, // These should not affect the ratio

		SumFidelityDropStrictEnvironment: 1.0, // mean 0.1
		SumFidelityDropNoJump:            0.1, // mean 0.01
		SumFidelityDropBoundaryCrossing:  5.0, // Should be ignored in primary
	}

	m := acc.FinalizeMetrics()
	if m.ConditionalUpdateRatio == nil {
		t.Fatalf("Expected ratio to be calculated")
	}

	// ratio = 0.1 / 0.01 = 10.0
	if *m.ConditionalUpdateRatio != 10.0 {
		t.Errorf("Expected ratio 10.0, got %v", *m.ConditionalUpdateRatio)
	}

	if *m.MeanFidelityDropAtBoundaryCrossingJumps != 0.5 {
		t.Errorf("Expected boundary mean 0.5, got %v", *m.MeanFidelityDropAtBoundaryCrossingJumps)
	}
}
