package bellmipt

import "testing"

func TestInterpretConditionalUpdate(t *testing.T) {
	tests := []struct {
		name    string
		ratio   *float64
		status  string
		wantCat string
	}{
		{
			name:    "not assessed when disabled/failed",
			ratio:   func() *float64 { v := 5.0; return &v }(),
			status:  "bridge_audit_inconclusive",
			wantCat: "not_assessed",
		},
		{
			name:    "not assessed when ratio null",
			ratio:   nil,
			status:  "bridge_audit_completed",
			wantCat: "not_assessed",
		},
		{
			name:    "no clear correlation",
			ratio:   func() *float64 { v := 1.2; return &v }(),
			status:  "bridge_audit_completed",
			wantCat: "no_clear_correlation",
		},
		{
			name:    "weak correlation",
			ratio:   func() *float64 { v := 2.5; return &v }(),
			status:  "bridge_audit_completed",
			wantCat: "weak_correlation",
		},
		{
			name:    "candidate correlation",
			ratio:   func() *float64 { v := 3.5; return &v }(),
			status:  "bridge_audit_completed",
			wantCat: "candidate_correlation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := InterpretConditionalUpdate(tc.ratio, tc.status)
			if got.EnvironmentCorrelatedConditionalUpdate != tc.wantCat {
				t.Errorf("got category %v, want %v", got.EnvironmentCorrelatedConditionalUpdate, tc.wantCat)
			}
			if got.Reason != "Descriptive finite-toy diagnostic only. This does not establish a Bell-MIPT bridge." {
				t.Errorf("got incorrect reason string: %s", got.Reason)
			}
		})
	}
}
