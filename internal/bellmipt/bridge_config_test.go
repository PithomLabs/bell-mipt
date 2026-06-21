package bellmipt

import (
	"reflect"
	"testing"
)

func TestBridgeConfigValidation(t *testing.T) {
	tests := []struct {
		name       string
		config     BridgeConfig
		totalSites int
		wantErr    bool
	}{
		{
			name: "valid partition",
			config: BridgeConfig{
				Enabled:          true,
				SubsystemSites:   []int{0, 2},
				EnvironmentSites: []int{1},
				Trajectories:     10,
				SampleEverySteps: 1,
			},
			totalSites: 3,
			wantErr:    false,
		},
		{
			name: "disabled skips validation",
			config: BridgeConfig{
				Enabled: false,
			},
			totalSites: 3,
			wantErr:    false,
		},
		{
			name: "empty subsystem",
			config: BridgeConfig{
				Enabled:          true,
				SubsystemSites:   []int{},
				EnvironmentSites: []int{0, 1, 2},
				Trajectories:     10,
				SampleEverySteps: 1,
			},
			totalSites: 3,
			wantErr:    true,
		},
		{
			name: "empty environment",
			config: BridgeConfig{
				Enabled:          true,
				SubsystemSites:   []int{0, 1, 2},
				EnvironmentSites: []int{},
				Trajectories:     10,
				SampleEverySteps: 1,
			},
			totalSites: 3,
			wantErr:    true,
		},
		{
			name: "duplicate in subsystem",
			config: BridgeConfig{
				Enabled:          true,
				SubsystemSites:   []int{0, 0},
				EnvironmentSites: []int{1, 2},
				Trajectories:     10,
				SampleEverySteps: 1,
			},
			totalSites: 3,
			wantErr:    true,
		},
		{
			name: "overlap",
			config: BridgeConfig{
				Enabled:          true,
				SubsystemSites:   []int{0, 1},
				EnvironmentSites: []int{1, 2},
				Trajectories:     10,
				SampleEverySteps: 1,
			},
			totalSites: 3,
			wantErr:    true,
		},
		{
			name: "missing site",
			config: BridgeConfig{
				Enabled:          true,
				SubsystemSites:   []int{0},
				EnvironmentSites: []int{2},
				Trajectories:     10,
				SampleEverySteps: 1,
			},
			totalSites: 3,
			wantErr:    true,
		},
		{
			name: "out of bounds",
			config: BridgeConfig{
				Enabled:          true,
				SubsystemSites:   []int{0, 3},
				EnvironmentSites: []int{1, 2},
				Trajectories:     10,
				SampleEverySteps: 1,
			},
			totalSites: 3,
			wantErr:    true,
		},
		{
			name: "zero trajectories",
			config: BridgeConfig{
				Enabled:          true,
				SubsystemSites:   []int{0},
				EnvironmentSites: []int{1},
				Trajectories:     0,
				SampleEverySteps: 1,
			},
			totalSites: 2,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate(tc.totalSites)
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestCanonicalSites(t *testing.T) {
	bc := BridgeConfig{
		SubsystemSites:   []int{2, 0, 1},
		EnvironmentSites: []int{5, 3, 4},
	}

	sub := bc.CanonicalSubsystemSites()
	env := bc.CanonicalEnvironmentSites()

	if !reflect.DeepEqual(sub, []int{0, 1, 2}) {
		t.Errorf("CanonicalSubsystemSites() = %v, want [0, 1, 2]", sub)
	}
	if !reflect.DeepEqual(env, []int{3, 4, 5}) {
		t.Errorf("CanonicalEnvironmentSites() = %v, want [3, 4, 5]", env)
	}

	// Ensure original not modified
	if !reflect.DeepEqual(bc.SubsystemSites, []int{2, 0, 1}) {
		t.Errorf("Original SubsystemSites modified: %v", bc.SubsystemSites)
	}
}
