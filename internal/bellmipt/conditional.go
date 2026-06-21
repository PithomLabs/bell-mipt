package bellmipt

import (
	"fmt"
	"math"
	"math/cmplx"
)

const ConditionalNormFloor = 1e-12

type QAPair struct {
	FullIndex int
	SubIndex  int
}

type ConditionalProjector struct {
	Split         SiteSplit
	ByEnvironment [][]QAPair
}

// BuildConditionalProjector creates the conditional projector structure.
func BuildConditionalProjector(split SiteSplit, basis []uint64) ConditionalProjector {
	byEnv := make([][]QAPair, split.EnvironmentDim)
	for fullIdx, q := range basis {
		subQ, envQ := SplitConfig(q, split)
		byEnv[envQ] = append(byEnv[envQ], QAPair{
			FullIndex: fullIdx,
			SubIndex:  int(subQ),
		})
	}
	return ConditionalProjector{
		Split:         split,
		ByEnvironment: byEnv,
	}
}

type ConditionalVector struct {
	EnvConfig  uint64
	Vector     []complex128
	Norm       float64
	Normalized bool
}

// EnvironmentProjectedConditionalVector computes the projection of the full vector onto the environment subspace.
func EnvironmentProjectedConditionalVector(psi []complex128, envQ uint64, projector ConditionalProjector) (ConditionalVector, error) {
	if int(envQ) >= len(projector.ByEnvironment) {
		return ConditionalVector{}, fmt.Errorf("envQ %d out of bounds", envQ)
	}

	pairs := projector.ByEnvironment[envQ]
	vec := make([]complex128, projector.Split.SubsystemDim)

	var normSq float64
	for _, p := range pairs {
		val := psi[p.FullIndex]
		vec[p.SubIndex] = val
		normSq += real(val)*real(val) + imag(val)*imag(val)
	}

	norm := math.Sqrt(normSq)
	normalized := false
	if norm >= ConditionalNormFloor {
		normalized = true
		for i := range vec {
			vec[i] /= complex(norm, 0)
		}
	}

	return ConditionalVector{
		EnvConfig:  envQ,
		Vector:     vec,
		Norm:       norm,
		Normalized: normalized,
	}, nil
}

// ConditionalFidelity computes the fidelity F = |\sum conj(a_i) b_i|^2.
func ConditionalFidelity(a, b []complex128) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length")
	}

	var dot complex128
	for i := range a {
		dot += cmplx.Conj(a[i]) * b[i]
	}

	f := real(dot)*real(dot) + imag(dot)*imag(dot)
	if f > 1.0 {
		if f-1.0 <= 1e-12 {
			f = 1.0
		} else {
			return 0, fmt.Errorf("fidelity %f exceeds 1.0 by more than 1e-12", f)
		}
	} else if f < 0.0 {
		if f >= -1e-12 {
			f = 0.0
		} else {
			return 0, fmt.Errorf("fidelity %f is less than 0.0 by more than 1e-12", f)
		}
	}

	return f, nil
}
