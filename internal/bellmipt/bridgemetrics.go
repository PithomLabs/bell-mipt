package bellmipt

// JumpClass categorizes a discrete-time transition.
type JumpClass int

const (
	NoJump JumpClass = iota
	StrictEnvironmentJump
	StrictSubsystemJump
	BoundaryCrossingJump
)

// ClassifyJump determines the type of jump given the before/after configurations
// and the site partition.
func ClassifyJump(prevConfig, nextConfig uint64, split SiteSplit) JumpClass {
	if prevConfig == nextConfig {
		return NoJump
	}

	prevSub, prevEnv := SplitConfig(prevConfig, split)
	nextSub, nextEnv := SplitConfig(nextConfig, split)

	subChanged := prevSub != nextSub
	envChanged := prevEnv != nextEnv

	if subChanged && envChanged {
		return BoundaryCrossingJump
	} else if envChanged && !subChanged {
		return StrictEnvironmentJump
	} else if subChanged && !envChanged {
		return StrictSubsystemJump
	}

	return NoJump // Unreachable
}

// BridgeMetrics holds the accumulated statistics for the bridge audit.
type BridgeMetrics struct {
	TrajectoryCount                int     `json:"trajectory_count"`
	MeanJumpCount                  float64 `json:"mean_jump_count"`
	MeanStrictEnvironmentJumpCount float64 `json:"mean_strict_environment_jump_count"`
	MeanStrictSubsystemJumpCount   float64 `json:"mean_strict_subsystem_jump_count"`
	MeanBoundaryCrossingJumpCount  float64 `json:"mean_boundary_crossing_jump_count"`

	ConditionalNormFailures int `json:"conditional_norm_failures"`

	MeanFidelityDropAtStrictEnvironmentJumps *float64 `json:"mean_fidelity_drop_at_strict_environment_jumps"`
	MeanFidelityDropAtStrictSubsystemJumps   *float64 `json:"mean_fidelity_drop_at_strict_subsystem_jumps"`
	MeanFidelityDropAtBoundaryCrossingJumps  *float64 `json:"mean_fidelity_drop_at_boundary_crossing_jumps"`
	MeanFidelityDropNoJump                   *float64 `json:"mean_fidelity_drop_no_jump"`
	MeanFidelityDropAtAnyJumps               *float64 `json:"mean_fidelity_drop_at_any_jumps"`

	ConditionalUpdateRatio                 *float64 `json:"conditional_update_ratio"`
	ConditionalUpdateRatioStatus           string   `json:"conditional_update_ratio_status"`
	ConditionalUpdateRatioEnvEventCount    int      `json:"conditional_update_ratio_env_event_count"`
	ConditionalUpdateRatioNoJumpEventCount int      `json:"conditional_update_ratio_no_jump_event_count"`

	InitialEmpiricalEquivarianceL1 float64 `json:"initial_empirical_equivariance_l1"`
	MaxEmpiricalEquivarianceL1     float64 `json:"max_empirical_equivariance_l1"`
	FinalEmpiricalEquivarianceL1   float64 `json:"final_empirical_equivariance_l1"`

	MaxLambdaDT              float64 `json:"max_lambda_dt"`
	NearZeroProbabilityCount int     `json:"near_zero_current_configuration_probability_count"`

	StrictEnvironmentJumpTransitions int `json:"strict_environment_jump_transitions"`
	StrictSubsystemJumpTransitions   int `json:"strict_subsystem_jump_transitions"`
	BoundaryCrossingJumpTransitions  int `json:"boundary_crossing_jump_transitions"`
	NoJumpTransitions                int `json:"no_jump_transitions"`
	AnyJumpTransitions               int `json:"any_jump_transitions"`
}

// BridgeAccumulator accumulates per-trajectory bridge statistics.
type BridgeAccumulator struct {
	Trajectories int

	TotalJumps                  int
	TotalStrictEnvironmentJumps int
	TotalStrictSubsystemJumps   int
	TotalBoundaryCrossingJumps  int

	SumFidelityDropStrictEnvironment float64
	SumFidelityDropStrictSubsystem   float64
	SumFidelityDropBoundaryCrossing  float64
	SumFidelityDropNoJump            float64
	SumFidelityDropAnyJump           float64

	EventCountStrictEnvironment int
	EventCountStrictSubsystem   int
	EventCountBoundaryCrossing  int
	EventCountNoJump            int
	EventCountAnyJump           int

	ConditionalNormFailures  int
	MaxLambdaDT              float64
	NearZeroProbabilityCount int

	InitialL1 float64
	MaxL1     float64
	FinalL1   float64
}

// RecordTransition records a single step transition and updates accumulators.
func (a *BridgeAccumulator) RecordTransition(
	class JumpClass,
	fidelityDrop float64,
	normFailure bool,
) {
	if normFailure {
		a.ConditionalNormFailures++
		// Skip fidelity drop accumulation
		return
	}

	switch class {
	case NoJump:
		a.EventCountNoJump++
		a.SumFidelityDropNoJump += fidelityDrop
	case StrictEnvironmentJump:
		a.EventCountStrictEnvironment++
		a.SumFidelityDropStrictEnvironment += fidelityDrop
		a.EventCountAnyJump++
		a.SumFidelityDropAnyJump += fidelityDrop
		a.TotalJumps++
		a.TotalStrictEnvironmentJumps++
	case StrictSubsystemJump:
		a.EventCountStrictSubsystem++
		a.SumFidelityDropStrictSubsystem += fidelityDrop
		a.EventCountAnyJump++
		a.SumFidelityDropAnyJump += fidelityDrop
		a.TotalJumps++
		a.TotalStrictSubsystemJumps++
	case BoundaryCrossingJump:
		a.EventCountBoundaryCrossing++
		a.SumFidelityDropBoundaryCrossing += fidelityDrop
		a.EventCountAnyJump++
		a.SumFidelityDropAnyJump += fidelityDrop
		a.TotalJumps++
		a.TotalBoundaryCrossingJumps++
	}
}

func meanPtr(sum float64, count int) *float64 {
	if count == 0 {
		return nil
	}
	v := sum / float64(count)
	return &v
}

// FinalizeMetrics computes the final metrics struct from the accumulator.
func (a *BridgeAccumulator) FinalizeMetrics() BridgeMetrics {
	m := BridgeMetrics{
		TrajectoryCount: a.Trajectories,

		ConditionalNormFailures:  a.ConditionalNormFailures,
		MaxLambdaDT:              a.MaxLambdaDT,
		NearZeroProbabilityCount: a.NearZeroProbabilityCount,

		InitialEmpiricalEquivarianceL1: a.InitialL1,
		MaxEmpiricalEquivarianceL1:     a.MaxL1,
		FinalEmpiricalEquivarianceL1:   a.FinalL1,

		StrictEnvironmentJumpTransitions: a.EventCountStrictEnvironment,
		StrictSubsystemJumpTransitions:   a.EventCountStrictSubsystem,
		BoundaryCrossingJumpTransitions:  a.EventCountBoundaryCrossing,
		NoJumpTransitions:                a.EventCountNoJump,
		AnyJumpTransitions:               a.EventCountAnyJump,

		ConditionalUpdateRatioEnvEventCount:    a.EventCountStrictEnvironment,
		ConditionalUpdateRatioNoJumpEventCount: a.EventCountNoJump,
	}

	if a.Trajectories > 0 {
		trajFloat := float64(a.Trajectories)
		m.MeanJumpCount = float64(a.TotalJumps) / trajFloat
		m.MeanStrictEnvironmentJumpCount = float64(a.TotalStrictEnvironmentJumps) / trajFloat
		m.MeanStrictSubsystemJumpCount = float64(a.TotalStrictSubsystemJumps) / trajFloat
		m.MeanBoundaryCrossingJumpCount = float64(a.TotalBoundaryCrossingJumps) / trajFloat
	}

	m.MeanFidelityDropAtStrictEnvironmentJumps = meanPtr(a.SumFidelityDropStrictEnvironment, a.EventCountStrictEnvironment)
	m.MeanFidelityDropAtStrictSubsystemJumps = meanPtr(a.SumFidelityDropStrictSubsystem, a.EventCountStrictSubsystem)
	m.MeanFidelityDropAtBoundaryCrossingJumps = meanPtr(a.SumFidelityDropBoundaryCrossing, a.EventCountBoundaryCrossing)
	m.MeanFidelityDropNoJump = meanPtr(a.SumFidelityDropNoJump, a.EventCountNoJump)
	m.MeanFidelityDropAtAnyJumps = meanPtr(a.SumFidelityDropAnyJump, a.EventCountAnyJump)

	// Primary ratio
	if a.EventCountStrictEnvironment >= 10 && a.EventCountNoJump >= 10 && m.MeanFidelityDropNoJump != nil && *m.MeanFidelityDropNoJump > 1e-14 {
		ratio := *m.MeanFidelityDropAtStrictEnvironmentJumps / *m.MeanFidelityDropNoJump
		m.ConditionalUpdateRatio = &ratio
		m.ConditionalUpdateRatioStatus = "available"
	} else {
		m.ConditionalUpdateRatio = nil
		m.ConditionalUpdateRatioStatus = "unavailable_insufficient_events"
	}

	return m
}
