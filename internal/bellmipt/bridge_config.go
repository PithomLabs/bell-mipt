package bellmipt

import (
	"fmt"
	"sort"
)

// BridgeConfig configures the optional Bell trajectory bridge audit.
type BridgeConfig struct {
	Enabled          bool  `json:"enabled"`
	SubsystemSites   []int `json:"subsystem_sites"`
	EnvironmentSites []int `json:"environment_sites"`
	Trajectories     int   `json:"trajectories"`
	Seed             int64 `json:"seed"`
	SampleEverySteps int   `json:"sample_every_steps"`
}

// Validate checks the BridgeConfig constraints and ensures
// the subsystem and environment strictly partition the lattice
// without overlaps, missing sites, or out-of-bounds indices.
func (bc *BridgeConfig) Validate(totalSites int) error {
	if !bc.Enabled {
		return nil
	}

	if len(bc.SubsystemSites) == 0 {
		return &ValidationError{Field: "bridge.subsystem_sites", Message: "cannot be empty"}
	}
	if len(bc.EnvironmentSites) == 0 {
		return &ValidationError{Field: "bridge.environment_sites", Message: "cannot be empty"}
	}
	if bc.Trajectories <= 0 {
		return &ValidationError{Field: "bridge.trajectories", Message: "must be > 0"}
	}
	if bc.SampleEverySteps < 1 {
		return &ValidationError{Field: "bridge.sample_every_steps", Message: "must be >= 1"}
	}

	seen := make(map[int]bool)

	for _, site := range bc.SubsystemSites {
		if site < 0 || site >= totalSites {
			return &ValidationError{Field: "bridge.subsystem_sites", Message: fmt.Sprintf("site %d out of bounds for %d sites", site, totalSites)}
		}
		if seen[site] {
			return &ValidationError{Field: "bridge.subsystem_sites", Message: fmt.Sprintf("duplicate site %d", site)}
		}
		seen[site] = true
	}

	for _, site := range bc.EnvironmentSites {
		if site < 0 || site >= totalSites {
			return &ValidationError{Field: "bridge.environment_sites", Message: fmt.Sprintf("site %d out of bounds for %d sites", site, totalSites)}
		}
		if seen[site] {
			return &ValidationError{Field: "bridge.environment_sites", Message: fmt.Sprintf("site %d overlaps with subsystem or is duplicate", site)}
		}
		seen[site] = true
	}

	if len(seen) != totalSites {
		return &ValidationError{Field: "bridge", Message: "subsystem and environment must partition all sites exactly"}
	}

	return nil
}

// CanonicalSubsystemSites returns a sorted copy of SubsystemSites.
func (bc *BridgeConfig) CanonicalSubsystemSites() []int {
	sorted := make([]int, len(bc.SubsystemSites))
	copy(sorted, bc.SubsystemSites)
	sort.Ints(sorted)
	return sorted
}

// CanonicalEnvironmentSites returns a sorted copy of EnvironmentSites.
func (bc *BridgeConfig) CanonicalEnvironmentSites() []int {
	sorted := make([]int, len(bc.EnvironmentSites))
	copy(sorted, bc.EnvironmentSites)
	sort.Ints(sorted)
	return sorted
}
