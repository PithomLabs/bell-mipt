package bellmipt

import (
	"fmt"
	"sort"
)

type SiteSplit struct {
	SubsystemSitesRequested   []int
	EnvironmentSitesRequested []int
	SubsystemSitesCanonical   []int
	EnvironmentSitesCanonical []int
	SubsystemDim              int
	EnvironmentDim            int
}

// NewSiteSplit creates a SiteSplit. Validates that subsystem and environment disjointly cover [0, sites-1].
func NewSiteSplit(subsystem, environment []int, sites int) (SiteSplit, error) {
	if len(subsystem)+len(environment) != sites {
		return SiteSplit{}, fmt.Errorf("number of subsystem and environment sites must equal total sites")
	}

	seen := make(map[int]bool)
	for _, s := range subsystem {
		if s < 0 || s >= sites {
			return SiteSplit{}, fmt.Errorf("site %d out of bounds", s)
		}
		if seen[s] {
			return SiteSplit{}, fmt.Errorf("duplicate site %d", s)
		}
		seen[s] = true
	}
	for _, s := range environment {
		if s < 0 || s >= sites {
			return SiteSplit{}, fmt.Errorf("site %d out of bounds", s)
		}
		if seen[s] {
			return SiteSplit{}, fmt.Errorf("site %d overlaps or is duplicated", s)
		}
		seen[s] = true
	}

	subCanonical := make([]int, len(subsystem))
	copy(subCanonical, subsystem)
	sort.Ints(subCanonical)

	envCanonical := make([]int, len(environment))
	copy(envCanonical, environment)
	sort.Ints(envCanonical)

	return SiteSplit{
		SubsystemSitesRequested:   subsystem,
		EnvironmentSitesRequested: environment,
		SubsystemSitesCanonical:   subCanonical,
		EnvironmentSitesCanonical: envCanonical,
		SubsystemDim:              1 << len(subsystem),
		EnvironmentDim:            1 << len(environment),
	}, nil
}

// SplitConfig extracts the bits of q according to split.
// The subsystem bits of q are extracted and packed contiguously into subQ based on the canonical order.
// Environment bits into envQ.
func SplitConfig(q uint64, split SiteSplit) (subQ uint64, envQ uint64) {
	for i, site := range split.SubsystemSitesCanonical {
		bit := (q >> site) & 1
		subQ |= (bit << i)
	}
	for i, site := range split.EnvironmentSitesCanonical {
		bit := (q >> site) & 1
		envQ |= (bit << i)
	}
	return subQ, envQ
}

// CombineConfig merges subQ and envQ.
func CombineConfig(subQ uint64, envQ uint64, split SiteSplit) uint64 {
	var q uint64
	for i, site := range split.SubsystemSitesCanonical {
		bit := (subQ >> i) & 1
		q |= (bit << site)
	}
	for i, site := range split.EnvironmentSitesCanonical {
		bit := (envQ >> i) & 1
		q |= (bit << site)
	}
	return q
}
