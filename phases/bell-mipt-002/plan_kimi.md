# Implementation Plan: BELL-MIPT-0002A — Bell Trajectory + Conditional Subsystem State Audit

## Overview

Extend the existing Go toy program (`cmd/bellmipt`) to sample actual Bell configuration histories and compute conditional subsystem wave functions induced by environment configurations. This is a direct extension of `BELL-MIPT-0001`, preserving all existing behavior when the bridge feature is disabled.

---

## 1. File Layout Changes

### New Files

```
internal/bellmipt/bridge.go       // Bridge config, validation, and orchestration
internal/bellmipt/trajectory.go   // Bell trajectory sampling (RNG, jumps, thinning)
internal/bellmipt/conditional.go  // Conditional wave function construction and fidelity
internal/bellmipt/bridge_audit.go // Bridge metrics computation and classification
internal/bellmipt/bridge_test.go  // Tests for trajectory, conditional, and bridge logic
```

### Modified Files

```
internal/bellmipt/config.go       // Add BridgeConfig struct, validation, defaults
internal/bellmipt/run.go          // Add bridge branch in Run(), preserve old path
internal/bellmipt/report.go       // Add BridgeReport and BridgeMetrics structs
internal/bellmipt/audit.go        // Add bridge status logic, forbidden language audit
internal/bellmipt/markdown.go     // Add bridge section rendering
```

### Unchanged Files (verify compatibility)

```
cmd/bellmipt/main.go              // No subcommand changes
internal/bellmipt/basis.go        // Existing basis functions reused
internal/bellmipt/fermion.go      // Existing fermion operators reused
internal/bellmipt/hamiltonian.go  // Existing Hamiltonian reused
internal/bellmipt/current.go      // Existing Bell current reused
internal/bellmipt/rates.go        // Existing Bell rates reused
internal/bellmipt/evolve.go       // Existing time evolution reused
internal/bellmipt/forbidden.go    // Existing forbidden language audit reused
```

---

## 2. New Config Structs

### BridgeConfig

```go
// BridgeConfig controls the Bell trajectory sampler and conditional subsystem audit.
// When Enabled is false or the field is omitted, BELL-MIPT-0001 behavior is preserved.
type BridgeConfig struct {
    Enabled            bool   `json:"enabled"`
    SubsystemSites     []int  `json:"subsystem_sites"`
    EnvironmentSites   []int  `json:"environment_sites"`
    Trajectories       int    `json:"trajectories"`
    Seed               int64  `json:"seed"`
    SampleEverySteps   int    `json:"sample_every_steps"`
}
```

### Config Extension

```go
// Extend existing Config struct:
type Config struct {
    // ... existing BELL-MIPT-0001 fields ...
    Bridge *BridgeConfig `json:"bridge,omitempty"`
}
```

### Validation Rules (in config.go or bridge.go)

```go
func (c *Config) Validate() error {
    // ... existing validation ...
    if c.Bridge != nil && c.Bridge.Enabled {
        if err := c.Bridge.Validate(); err != nil {
            return fmt.Errorf("bridge config: %w", err)
        }
    }
    return nil
}

func (b *BridgeConfig) Validate() error {
    if b.Trajectories <= 0 {
        return fmt.Errorf("trajectories must be > 0, got %d", b.Trajectories)
    }
    if b.SampleEverySteps <= 0 {
        return fmt.Errorf("sample_every_steps must be > 0, got %d", b.SampleEverySteps)
    }
    // Partition validation: no overlap, no missing sites, all sites accounted for
    // Implementation: use map[int]bool to check overlap and completeness
    return nil
}
```

---

## 3. New Data Structures

### TrajectoryState

```go
// TrajectoryState holds the state of a single Bell trajectory at one time step.
type TrajectoryState struct {
    Q        int       // Current configuration index in full basis
    Psi      []complex128 // Current wavefunction (full Hilbert space)
    Time     float64   // Current time
    Step     int       // Current step number
}
```

### ConditionalState

```go
// ConditionalState holds the normalized conditional wave function for subsystem A.
type ConditionalState struct {
    Amps     []complex128 // Normalized amplitudes over subsystem configurations
    Norm     float64      // Raw norm before normalization
    Valid    bool         // False if norm below threshold
    ConfigA  int          // Subsystem configuration index (if tracked)
}
```

### BridgeMetrics (runtime accumulator)

```go
// BridgeMetrics accumulates statistics across all trajectories and samples.
type BridgeMetrics struct {
    TrajectoryCount                    int
    TotalJumps                         int64
    TotalEnvironmentJumps              int64
    TotalSubsystemJumps                int64
    ConditionalNormFailures            int64
    FidelityDropEnvironmentJumps       []float64
    FidelityDropNoEnvironmentJumps     []float64
    FidelityDropAnyJumps               []float64
    FidelityDropNoJumps                []float64
    EmpiricalDistribution              map[int]int // config -> count, per sample time
    EmpiricalEquivarianceL1            []float64   // per sample time
}
```

### BridgeReport (JSON output)

```go
// BridgeReport is the serializable bridge section of the final report.
type BridgeReport struct {
    Enabled        bool            `json:"enabled"`
    BridgeGoal     string          `json:"bridge_goal"`
    BridgeStatus   string          `json:"bridge_status"`
    Trajectories   int             `json:"trajectories"`
    SubsystemSites []int           `json:"subsystem_sites"`
    EnvironmentSites []int         `json:"environment_sites"`
    SampleEverySteps int           `json:"sample_every_steps"`
    Metrics        BridgeMetricsReport `json:"metrics"`
    Interpretation BridgeInterpretation `json:"interpretation"`
}

type BridgeMetricsReport struct {
    TrajectoryCount                           int     `json:"trajectory_count"`
    MeanJumpCount                             float64 `json:"mean_jump_count"`
    MeanEnvironmentJumpCount                  float64 `json:"mean_environment_jump_count"`
    MeanSubsystemJumpCount                    float64 `json:"mean_subsystem_jump_count"`
    ConditionalNormFailures                   int     `json:"conditional_norm_failures"`
    MeanFidelityDropAtEnvironmentJumps          float64 `json:"mean_fidelity_drop_at_environment_jumps"`
    MeanFidelityDropWithoutEnvironmentJumps   float64 `json:"mean_fidelity_drop_without_environment_jumps"`
    ConditionalUpdateRatio                    float64 `json:"conditional_update_ratio"`
    MaxEmpiricalEquivarianceL1                float64 `json:"max_empirical_equivariance_l1"`
    FinalEmpiricalEquivarianceL1              float64 `json:"final_empirical_equivariance_l1"`
}

type BridgeInterpretation struct {
    MonitoringLikeSignal string `json:"monitoring_like_signal"`
    Reason               string `json:"reason"`
}
```

---

## 4. Trajectory Sampling Algorithm

### File: `internal/bellmipt/trajectory.go`

```go
package bellmipt

import (
    "math"
    "math/rand"
)

// TrajectorySampler manages the stochastic Bell trajectory sampling.
type TrajectorySampler struct {
    rng       *rand.Rand
    dt        float64
    threshold float64 // conditional norm threshold
}

// NewTrajectorySampler creates a sampler with deterministic RNG from seed.
func NewTrajectorySampler(seed int64, dt float64, threshold float64) *TrajectorySampler {
    return &TrajectorySampler{
        rng:       rand.New(rand.NewSource(seed)),
        dt:        dt,
        threshold: threshold,
    }
}

// SampleInitialConfiguration draws Q0 from |psi0|^2.
func (ts *TrajectorySampler) SampleInitialConfiguration(psi []complex128) int {
    // Compute probabilities
    probs := make([]float64, len(psi))
    for i, amp := range psi {
        probs[i] = real(amp)*real(amp) + imag(amp)*imag(amp)
    }
    // Cumulative distribution
    cumsum := make([]float64, len(probs))
    cumsum[0] = probs[0]
    for i := 1; i < len(probs); i++ {
        cumsum[i] = cumsum[i-1] + probs[i]
    }
    total := cumsum[len(cumsum)-1]
    if total == 0 {
        return 0 // fallback, should not happen with valid psi
    }
    r := ts.rng.Float64() * total
    // Binary search or linear scan for destination
    for i, c := range cumsum {
        if r <= c {
            return i
        }
    }
    return len(cumsum) - 1
}

// Step performs one discrete-time thinning step.
// Returns (newQ, jumpOccurred, destinationQ).
func (ts *TrajectorySampler) Step(q int, psi []complex128, rates []float64) (int, bool, int) {
    totalRate := 0.0
    for _, r := range rates {
        totalRate += r
    }
    if totalRate == 0 {
        return q, false, q
    }
    pJump := 1.0 - math.Exp(-totalRate*ts.dt)
    if ts.rng.Float64() >= pJump {
        return q, false, q
    }
    // Jump occurred: sample destination proportional to rates
    r := ts.rng.Float64() * totalRate
    cumsum := 0.0
    for n, rate := range rates {
        cumsum += rate
        if r <= cumsum {
            return n, true, n
        }
    }
    // Fallback to last state
    return len(rates) - 1, true, len(rates) - 1
}

// RunTrajectory executes a full trajectory from t=0 to t=T.
// Returns the trajectory history (slice of TrajectoryState at sample points).
func (ts *TrajectorySampler) RunTrajectory(
    psi0 []complex128,
    evolveFunc func([]complex128, float64) []complex128, // evolves psi by dt
    rateFunc func([]complex128, int) []float64,         // returns rates from given q
    sampleEvery int,
    totalSteps int,
) []TrajectoryState {
    // Implementation:
    // 1. Sample initial Q0 from |psi0|^2
    // 2. For each step:
    //    a. Evolve psi by dt (Schrödinger evolution)
    //    b. Compute rates from current q
    //    c. Call Step() to possibly jump
    //    d. If jump, update q
    //    e. If step % sampleEvery == 0, record TrajectoryState
    // 3. Return recorded states
    return nil
}
```

---

## 5. Conditional Wave Function Construction

### File: `internal/bellmipt/conditional.go`

```go
package bellmipt

import (
    "math"
    "math/cmplx"
)

// ConditionalBuilder constructs subsystem conditional states from full configurations.
type ConditionalBuilder struct {
    SubsystemSites   []int
    EnvironmentSites []int
    NumSubsystem     int // len(SubsystemSites)
    NumEnvironment   int // len(EnvironmentSites)
    SubsystemMask    int // bitmask for extracting subsystem bits
    EnvironmentMask  int // bitmask for extracting environment bits
    // Precomputed mapping: full_config -> (sub_config, env_config)
    SplitMap         []struct{ A, B int }
}

// NewConditionalBuilder creates a builder from site partitions.
// Assumes sites are 0..N-1 and partition is valid.
func NewConditionalBuilder(subSites, envSites []int, totalSites int) *ConditionalBuilder {
    // Precompute masks and split map for all 2^totalSites configurations
    return nil
}

// ExtractSubsystemConfig returns the subsystem configuration index for a full config.
func (cb *ConditionalBuilder) ExtractSubsystemConfig(fullConfig int) int {
    // Use precomputed SplitMap or bitwise extraction
    return 0
}

// ExtractEnvironmentConfig returns the environment configuration index for a full config.
func (cb *ConditionalBuilder) ExtractEnvironmentConfig(fullConfig int) int {
    return 0
}

// BuildConditionalState constructs psi_A(a) = Psi(a, b0, t) / norm, for fixed b0.
func (cb *ConditionalBuilder) BuildConditionalState(
    psi []complex128, // full wavefunction
    envConfig int,    // fixed environment configuration b0
) ConditionalState {
    dimA := 1 << cb.NumSubsystem
    amps := make([]complex128, dimA)
    var normSq float64
    for a := 0; a < dimA; a++ {
        // Reconstruct full config from (a, envConfig)
        fullConfig := cb.ReconstructFullConfig(a, envConfig)
        amp := psi[fullConfig]
        amps[a] = amp
        normSq += real(amp)*real(amp) + imag(amp)*imag(amp)
    }
    if normSq < 1e-15 {
        return ConditionalState{Valid: false, Norm: normSq}
    }
    norm := math.Sqrt(normSq)
    for i := range amps {
        amps[i] /= complex(norm, 0)
    }
    return ConditionalState{Amps: amps, Norm: norm, Valid: true}
}

// ReconstructFullConfig combines subsystem and environment configs into full config.
func (cb *ConditionalBuilder) ReconstructFullConfig(subConfig, envConfig int) int {
    // Bit manipulation: place subsystem bits and environment bits at correct positions
    return 0
}

// ComputeFidelity returns |<psiA | phiA>|^2.
func ComputeFidelity(a, b ConditionalState) float64 {
    if !a.Valid || !b.Valid {
        return 0
    }
    var inner complex128
    for i := range a.Amps {
        inner += cmplx.Conj(a.Amps[i]) * b.Amps[i]
    }
    return real(inner)*real(inner) + imag(inner)*imag(inner)
}
```

---

## 6. Bridge Metrics

### File: `internal/bellmipt/bridge_audit.go`

```go
package bellmipt

// BridgeAuditor computes bridge metrics from trajectory data.
type BridgeAuditor struct {
    metrics BridgeMetrics
    cb      *ConditionalBuilder
}

// NewBridgeAuditor creates an auditor with the given conditional builder.
func NewBridgeAuditor(cb *ConditionalBuilder) *BridgeAuditor {
    return &BridgeAuditor{cb: cb}
}

// RecordStep processes one sampled step pair (previous, current).
func (ba *BridgeAuditor) RecordStep(prev, curr TrajectoryState) {
    prevEnv := ba.cb.ExtractEnvironmentConfig(prev.Q)
    currEnv := ba.cb.ExtractEnvironmentConfig(curr.Q)
    prevSub := ba.cb.ExtractSubsystemConfig(prev.Q)
    currSub := ba.cb.ExtractSubsystemConfig(curr.Q)

    envJump := prevEnv != currEnv
    subJump := prevSub != currSub
    anyJump := prev.Q != curr.Q

    // Build conditional states
    prevCond := ba.cb.BuildConditionalState(prev.Psi, prevEnv)
    currCond := ba.cb.BuildConditionalState(curr.Psi, currEnv)

    if !prevCond.Valid || !currCond.Valid {
        ba.metrics.ConditionalNormFailures++
        return
    }

    fidelity := ComputeFidelity(prevCond, currCond)
    drop := 1.0 - fidelity
    if drop < 0 {
        drop = 0
    }
    if drop > 1 {
        drop = 1
    }

    if envJump {
        ba.metrics.FidelityDropEnvironmentJumps = append(ba.metrics.FidelityDropEnvironmentJumps, drop)
        ba.metrics.TotalEnvironmentJumps++
    } else {
        ba.metrics.FidelityDropNoEnvironmentJumps = append(ba.metrics.FidelityDropNoEnvironmentJumps, drop)
    }

    if subJump {
        ba.metrics.TotalSubsystemJumps++
    }

    if anyJump {
        ba.metrics.FidelityDropAnyJumps = append(ba.metrics.FidelityDropAnyJumps, drop)
        ba.metrics.TotalJumps++
    } else {
        ba.metrics.FidelityDropNoJumps = append(ba.metrics.FidelityDropNoJumps, drop)
    }
}

// Finalize computes final reportable metrics.
func (ba *BridgeAuditor) Finalize(trajectoryCount int) BridgeMetricsReport {
    m := ba.metrics
    report := BridgeMetricsReport{
        TrajectoryCount:         trajectoryCount,
        MeanJumpCount:           float64(m.TotalJumps) / float64(trajectoryCount),
        MeanEnvironmentJumpCount: float64(m.TotalEnvironmentJumps) / float64(trajectoryCount),
        MeanSubsystemJumpCount:   float64(m.TotalSubsystemJumps) / float64(trajectoryCount),
        ConditionalNormFailures:    int(m.ConditionalNormFailures),
    }

    report.MeanFidelityDropAtEnvironmentJumps = mean(m.FidelityDropEnvironmentJumps)
    report.MeanFidelityDropWithoutEnvironmentJumps = mean(m.FidelityDropNoEnvironmentJumps)

    if report.MeanFidelityDropWithoutEnvironmentJumps > 1e-12 {
        report.ConditionalUpdateRatio = report.MeanFidelityDropAtEnvironmentJumps / report.MeanFidelityDropWithoutEnvironmentJumps
    } else {
        report.ConditionalUpdateRatio = 0 // or mark as unavailable
    }

    // Empirical equivariance: compare empirical distribution at final time with |psi(T)|^2
    // (computed separately from accumulated EmpiricalDistribution)
    return report
}

func mean(vals []float64) float64 {
    if len(vals) == 0 {
        return 0
    }
    var sum float64
    for _, v := range vals {
        sum += v
    }
    return sum / float64(len(vals))
}
```

---

## 7. Report Schema Changes

### Extend Report struct in `report.go`

```go
type Report struct {
    ToyAnalysisOnly    bool            `json:"toy_analysis_only"`
    PhysicsClaim       string          `json:"physics_claim"`
    GoalStatus         string          `json:"goal_status"`
    Bridge             *BridgeReport   `json:"bridge,omitempty"`
    DebtStatus         DebtStatus      `json:"debt_status"`
    Limitations        []string        `json:"limitations"`
    // ... existing fields ...
}
```

### Bridge Status Logic

```go
func DetermineBridgeStatus(metrics BridgeMetricsReport, enabled bool) string {
    if !enabled {
        return "bridge_disabled"
    }
    if metrics.ConditionalNormFailures > metrics.TrajectoryCount*10 { // heuristic threshold
        return "bridge_toy_inconclusive"
    }
    if metrics.TrajectoryCount == 0 {
        return "bridge_toy_failed"
    }
    if len(metrics.FidelityDropEnvironmentJumps) == 0 && len(metrics.FidelityDropNoEnvironmentJumps) == 0 {
        return "bridge_toy_inconclusive"
    }
    return "bridge_toy_passed"
}
```

---

## 8. Goal/Bridge Status Logic

### In `run.go`:

```go
func Run(cfg Config) (Report, error) {
    // 1. Run existing BELL-MIPT-0001 path (always)
    baseReport := runBaseAudit(cfg)

    // 2. If bridge enabled, run bridge path
    if cfg.Bridge != nil && cfg.Bridge.Enabled {
        bridgeReport, err := runBridgeAudit(cfg, baseReport)
        if err != nil {
            baseReport.Bridge = &BridgeReport{
                Enabled:      true,
                BridgeStatus: "bridge_toy_failed",
            }
            return baseReport, err
        }
        baseReport.Bridge = bridgeReport
        baseReport.DebtStatus = updateDebtForBridge()
    } else {
        baseReport.Bridge = &BridgeReport{
            Enabled:      false,
            BridgeStatus: "bridge_disabled",
        }
        baseReport.DebtStatus = retainOldDebt()
    }

    // 3. Forbidden language audit (always)
    if err := auditForbiddenLanguage(baseReport); err != nil {
        return baseReport, err
    }

    return baseReport, nil
}
```

---

## 9. Tests

### File: `internal/bellmipt/bridge_test.go`

```go
package bellmipt

import (
    "testing"
    "math"
    "math/cmplx"
)

// --- Config Tests ---

func TestBridgeDefaultsToDisabled(t *testing.T) {
    // Parse config without bridge section; verify Bridge == nil or Enabled == false
}

func TestBridgeDisabledPreservesOldBehavior(t *testing.T) {
    // Run with bridge disabled; verify goal_status reflects BELL-MIPT-0001
}

func TestBridgePartitionOverlapRejected(t *testing.T) {
    // cfg.Bridge.SubsystemSites = []int{0,1,2}
    // cfg.Bridge.EnvironmentSites = []int{2,3,4}
    // Expect Validate() error
}

func TestBridgePartitionMissingSiteRejected(t *testing.T) {
    // Sites 0,1,2,3,4,5; sub={0,1,2}, env={3,4} (missing 5)
    // Expect Validate() error
}

func TestBridgePartitionValidAccepted(t *testing.T) {
    // sub={0,1,2}, env={3,4,5} for 6 sites
    // Expect Validate() success
}

func TestBridgeTrajectoriesPositive(t *testing.T) {
    // trajectories=0 with enabled=true → error
}

func TestBridgeSampleEveryStepsPositive(t *testing.T) {
    // sample_every_steps=0 → error
}

// --- Trajectory Sampling Tests ---

func TestInitialConfigSampledFromPsiSquared(t *testing.T) {
    // Run many samples, verify empirical frequencies match |psi|^2 within tolerance
}

func TestJumpProbabilityUsesTotalRate(t *testing.T) {
    // Mock rates, verify jump probability approximates 1-exp(-lambda*dt)
}

func TestDestinationSamplingProportionalToRates(t *testing.T) {
    // Mock rates [1,2,3], run many jumps, verify empirical distribution
}

func TestNoJumpWhenTotalRateZero(t *testing.T) {
    // rates all zero → no jump ever
}

func TestDeterministicTrajectoryForFixedSeed(t *testing.T) {
    // Same seed, same psi0, same rates → identical trajectory
}

// --- Conditional Wave Function Tests ---

func TestSplitFullStateCorrectly(t *testing.T) {
    // 4 sites, sub={0,1}, env={2,3}
    // full config 0b1010 (10) → sub=0b10 (2), env=0b10 (2)
}

func TestConditionalStateDimension(t *testing.T) {
    // |A|=3 → dimension 8
}

func TestConditionalStateNormalizes(t *testing.T) {
    // Build with non-zero norm; verify sum |amps|^2 == 1
}

func TestConditionalNormFailureRecorded(t *testing.T) {
    // psi with zero support on given env config → Valid=false
}

func TestFidelityOfIdenticalStates(t *testing.T) {
    // Fidelity(state, state) == 1
}

func TestFidelityDropInRange(t *testing.T) {
    // drop = 1 - F; verify 0 <= drop <= 1+epsilon
}

// --- Bridge Metrics Tests ---

func TestEnvironmentJumpsCountedCorrectly(t *testing.T) {
    // Mock trajectory with known env jumps; verify count
}

func TestSubsystemJumpsCountedCorrectly(t *testing.T) {
    // Mock trajectory with known sub jumps; verify count
}

func TestFidelityDropsClassifiedCorrectly(t *testing.T) {
    // envJump=true → recorded in EnvironmentJumps slice
    // envJump=false → recorded in NoEnvironmentJumps slice
}

func TestConditionalUpdateRatioHandlesZeroDenominator(t *testing.T) {
    // No non-env jumps → ratio should be 0 or marked unavailable
}

func TestEmpiricalEquivarianceL1Computed(t *testing.T) {
    // Run multiple trajectories, verify L1 distance formula
}

func TestBridgeStatusPassed(t *testing.T) {
    // Normal run → bridge_toy_passed
}

func TestBridgeStatusInconclusive(t *testing.T) {
    // Too many norm failures → bridge_toy_inconclusive
}

// --- Report Tests ---

func TestOldReportFieldsBackwardCompatible(t *testing.T) {
    // Verify JSON serialization includes old fields
}

func TestBridgeDisabledAppearsWhenOmitted(t *testing.T) {
    // No bridge config → bridge_status="bridge_disabled"
}

func TestBridgeMetricsAppearWhenEnabled(t *testing.T) {
    // Bridge enabled → metrics section present
}

func TestForbiddenLanguageAuditPasses(t *testing.T) {
    // Verify no forbidden claims in report text
}
```

---

## 10. Validation Commands

```bash
# Unit tests
go test ./...

# Race detection
go test -race ./...

# Static analysis
go vet ./...

# Default run (bridge disabled, BELL-MIPT-0001 behavior)
go run ./cmd/bellmipt --out /tmp/bellmipt-default

# Bridge enabled run
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge
```

---

## 11. EBP Debt Updates

### If Bridge Enabled and Runs Successfully:

```json
{
    "needMap": "partially_paid_conditional_state_toy_only",
    "needInvariant": "partially_paid_equivariance_plus_empirical_trajectory_check",
    "needToyCheck": "partially_paid_rate_algebra_and_conditional_state_toy",
    "needNullModel": "unpaid",
    "needObstruction": "bell_jumps_are_not_measurements",
    "needFaithfulnessReview": "source_code_reviewed_for_0001_only"
}
```

### If Bridge Disabled:

```json
{
    "needMap": "unpaid",
    "needInvariant": "partially_paid_equivariance_only",
    "needToyCheck": "partially_paid_rate_algebra_only",
    "needNullModel": "unpaid",
    "needObstruction": "bell_jumps_are_not_measurements",
    "needFaithfulnessReview": "source_code_reviewed"
}
```

---

## 12. Known Risks

| Risk | Mitigation |
|------|-----------|
| Discrete-time thinning introduces bias at large dt | Use same small dt as BELL-MIPT-0001; document limitation |
| Finite trajectories make empirical equivariance noisy | Report as descriptive diagnostic only; no strict threshold |
| Conditional norm failures from zero-probability env configs | Count and report; mark inconclusive if too frequent |
| Bitmask extraction incorrect for non-contiguous sites | Extensive unit tests for split/reconstruct functions |
| RNG non-determinism across Go versions | Use math/rand with explicit seed; document version dependency |
| Performance with many trajectories | Profile if needed; trajectory loop is embarrassingly parallel but keep simple |

---

## 13. Non-Goals (Explicitly Excluded)

- MIPT phase diagram
- Area-law / volume-law scaling
- Entanglement entropy
- Mutual information
- Purity
- Monitored quantum circuits
- Projective measurements
- Lindblad dynamics
- Holography
- Black holes
- Lean theorem proving
- AI agents
- Multi-flavor Majorana chains
- Physics promotion
- Continuous-time Gillespie sampler (unless strictly necessary)
- Claim that Bell jumps are measurements
- Claim that Bell-MIPT bridge is established

---

## 14. Recommendation: Direct Extension vs. Separate Package

**Recommendation: Direct Extension**

Implement `BELL-MIPT-0002A` as a direct extension of the existing `internal/bellmipt` package, not as a separate sub-package.

**Rationale:**

1. **Tight coupling**: The bridge reuses existing basis, Hamiltonian, current, rates, and evolution code. Separating would require exporting many currently internal symbols or duplicating logic.

2. **Config cohesion**: The bridge config is a natural extension of the existing config. Splitting configs across packages adds friction.

3. **Report unity**: The final report must contain both base and bridge sections. A single package makes report construction straightforward.

4. **Test simplicity**: `go test ./...` runs all tests. No need for cross-package test orchestration.

5. **Command shape preservation**: `cmd/bellmipt/main.go` requires no changes. The `run.go` orchestrator simply branches based on config.

**Organization within `internal/bellmipt`:**

- `bridge.go` — Bridge config, validation, and high-level orchestration
- `trajectory.go` — Stochastic sampling (self-contained, testable)
- `conditional.go` — Wave function splitting (self-contained, testable)
- `bridge_audit.go` — Metrics and classification (self-contained, testable)
- `bridge_test.go` — All bridge-related tests

This keeps the new code modular and testable while maintaining the existing architecture.

---

## 15. Implementation Checklist for Coding Agent

- [ ] Create `bridge.go` with `BridgeConfig`, validation, and `RunBridge()` orchestrator
- [ ] Create `trajectory.go` with `TrajectorySampler`, `SampleInitialConfiguration`, `Step`, `RunTrajectory`
- [ ] Create `conditional.go` with `ConditionalBuilder`, `BuildConditionalState`, `ComputeFidelity`
- [ ] Create `bridge_audit.go` with `BridgeAuditor`, `RecordStep`, `Finalize`
- [ ] Modify `config.go` to add `BridgeConfig` field and `Validate()` extension
- [ ] Modify `run.go` to branch on `cfg.Bridge != nil && cfg.Bridge.Enabled`
- [ ] Modify `report.go` to add `BridgeReport`, `BridgeMetricsReport`, `BridgeInterpretation`
- [ ] Modify `audit.go` to add `DetermineBridgeStatus()` and forbidden language checks for bridge text
- [ ] Modify `markdown.go` to render bridge section
- [ ] Create `bridge_test.go` with all test cases listed in Section 9
- [ ] Create `bellmipt_bridge.json` example config for validation
- [ ] Run `go test ./...`, `go test -race ./...`, `go vet ./...`
- [ ] Run default and bridge validation commands
- [ ] Verify backward compatibility when bridge omitted
- [ ] Verify report contains required limitations and no forbidden claims
- [ ] Update debt status in report based on bridge outcome
