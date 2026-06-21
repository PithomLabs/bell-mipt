# Implementation Plan: BELL-MIPT-0002A — Bell Trajectory + Conditional Subsystem State Audit

Status: PLAN ONLY. No production code included. Written for direct hand-off to a coding agent extending the existing `internal/bellmipt` package from `BELL-MIPT-0001`.

## Table of Contents

1. Scope restatement and non-goals
2. Pre-implementation assumptions to verify against actual `BELL-MIPT-0001` source
3. File layout changes
4. Config extension
5. New data structures
6. Algorithm: Bell trajectory sampling
7. Algorithm: conditional subsystem wave function
8. Bridge metrics (jump classification, fidelity, empirical equivariance)
9. Report schema changes
10. Goal/bridge status logic
11. EBP debt update
12. Proposed constants / defaults (single reference table)
13. Required tests
14. Validation commands
15. Known risks and open design decisions
16. Suggested build order
17. Recommendation: extend in place vs. new package

---

## 1. Scope restatement and non-goals

`BELL-MIPT-0002A` adds a trajectory-level audit on top of the existing ensemble-level Bell master-equation audit. It must answer, for a finite toy configuration only: *does conditioning on the actual environment configuration `Q_B(t)` produce conditional-state changes at the subsystem larger than the changes seen without an environment jump?*

This is **descriptive instrumentation**, not a measurement-theory result. Restating the ticket's hard constraints so they are visible in the plan itself, not just the source ticket:

```text
Out of scope for this ticket (do not implement):
  MIPT phase diagram, area-law/volume-law scaling, entanglement entropy,
  mutual information, purity, monitored quantum circuits, projective
  measurements, Lindblad dynamics, holography, black holes,
  Lean theorem proving, AI agents, multi-flavor Majorana chains,
  physics promotion of any kind.

Claims this ticket must never make, in code, comments, or reports:
  "Bell jumps are measurements."
  "Bell-MIPT bridge is established."
  "MIPT was observed."
  "Holography is explained."
  "Bohmian mechanics is validated."
```

Every section below is written to keep these boundaries mechanically enforced (status logic, forbidden-language audit, required limitations text) rather than relying on prose discipline alone.

---

## 2. Pre-implementation assumptions to verify against actual source

This plan is written without access to the actual `BELL-MIPT-0001` source. Before coding starts, the implementer must confirm or correct the following assumptions, because several of the new algorithms depend on them:

```text
A1. basis.go enumerates the full Hilbert space as integer indices q,
    and exposes (or can cheaply expose) a way to recover the per-site
    occupation pattern for a given q — either directly as a bitmask,
    or via an accessor like Occupations(q) []int. The plan below does
    NOT assume raw-bitmask indexing; it assumes an occupation accessor
    exists or can be added without disturbing existing logic.

A2. rates.go exposes (or can cheaply expose) a function that returns
    the full row of outgoing Bell rates sigma(n <- q, t) for a given
    origin configuration q and current psi(t), without requiring a
    rewrite of how BELL-MIPT-0001's ensemble audit already consumes
    rates internally.

A3. evolve.go computes psi(t) on a fixed dt grid, deterministically,
    independent of any particular Bell configuration. This is required
    for the trajectory sampler to reuse a single precomputed psi(t)
    timeline across all sampled trajectories instead of recomputing it
    per trajectory.

A4. The existing audit.go already holds or can expose this psi(t)
    timeline (it must, to do the existing equivariance check against
    |psi(t)|^2). If it currently discards intermediate psi(t) after
    use, a minimal non-behavioral refactor is needed to retain it.

A5. go.mod targets Go >= 1.22, so math/rand/v2 is available. If not,
    fall back to math/rand with explicit per-trajectory *rand.Rand
    instances (never the global source).
```

If A1–A4 turn out false, the affected sections (mainly §6 and §7) need the accessor functions added to the existing files as small, additive, non-behavior-changing exports — not reimplementations of basis/rate logic.

---

## 3. File layout changes

```text
internal/bellmipt/
  config.go          MODIFY: add Bridge field + wiring into existing Load/Validate
  report.go          MODIFY: add Bridge field to Report struct
  markdown.go        MODIFY: render bridge section when present
  forbidden.go       MODIFY: confirm/extend scan coverage over bridge section (see Risk R7)
  run.go             MODIFY: orchestrate bridge phase after existing 0001 pipeline
  basis.go           MODIFY (maybe): add minimal occupation accessor if not already present
  rates.go           MODIFY (maybe): add minimal "row of outgoing rates" accessor if not already present
  evolve.go          MODIFY (maybe): retain/expose psi(t) timeline if not already retained

  bridge_config.go        NEW: BridgeConfig struct + validation
  site_split.go            NEW: SiteSplit + ConditionalProjector (subsystem/environment partition)
  trajectory.go            NEW: trajectory sampler (initial-state + jump-step sampling)
  conditional.go           NEW: ConditionalState construction + fidelity
  bridgemetrics.go         NEW: jump classification + accumulators + ratio logic
  equivariance_empirical.go NEW: empirical Q(t) histogram vs |psi(t)|^2 diagnostic
  bridge_report.go         NEW: BridgeReport/BridgeMetrics/BridgeInterpretation + status logic

cmd/bellmipt/main.go   NO CHANGE EXPECTED (no new subcommands or flags; run.go absorbs the branch)

repo root:
  bellmipt_bridge.json  NEW: sample config fixture with bridge.enabled=true, used by the
                        validation command and by integration tests.
```

No existing file is rewritten wholesale. Every "MODIFY" above is additive: a new field on an existing struct, a new branch in an existing function, or a new accessor function placed next to the logic it reads from.

---

## 4. Config extension

```go
// BridgeConfig is the optional, additive configuration block introduced
// by BELL-MIPT-0002A. Its zero value (Enabled == false) must reproduce
// BELL-MIPT-0001 behavior exactly.
type BridgeConfig struct {
    Enabled          bool  `json:"enabled"`
    SubsystemSites   []int `json:"subsystem_sites"`
    EnvironmentSites []int `json:"environment_sites"`
    Trajectories     int   `json:"trajectories"`
    Seed             int64 `json:"seed"`
    SampleEverySteps int   `json:"sample_every_steps"`
}
```

Top-level `Config` gets one new field:

```go
type Config struct {
    // ... existing fields, unchanged ...
    Bridge BridgeConfig `json:"bridge"`
}
```

Because `encoding/json` leaves an omitted JSON object as the Go zero value, a config file with no `"bridge"` key automatically yields `Bridge.Enabled == false` with all other fields zeroed — no special-case parsing required.

Validation (new function, called from the existing config validation entry point, gated on `Enabled`):

```go
// Validate checks BridgeConfig only when Enabled is true. numSites is the
// total chain length already known to the existing config/basis code.
func (b BridgeConfig) Validate(numSites int) error
```

Required checks, in this order (so error messages are unambiguous):

```text
1. If !Enabled: return nil immediately. Do not validate other fields.
2. SubsystemSites and EnvironmentSites must each be non-empty.
3. No duplicate indices within SubsystemSites.
4. No duplicate indices within EnvironmentSites.
5. No index appears in both lists (overlap check).
6. Union of both lists, as a set, must equal {0, ..., numSites-1} exactly
   (no missing site, no out-of-range site, no extra unknown site).
7. Trajectories must be > 0.
8. SampleEverySteps must be >= 1.
9. Seed has no constraint (any int64, including 0, is valid and must
   still produce deterministic output).
```

This satisfies the ticket's seven explicit config tests directly (see §13).

---

## 5. New data structures

### 5.1 Site partition

```go
// SiteSplit precomputes everything needed to decompose a full
// configuration index into a (subsystem, environment) pair of local
// indices, and back. It does not assume a particular basis-indexing
// scheme; it is built from whatever occupation accessor basis.go
// exposes (see Assumption A1).
type SiteSplit struct {
    SubsystemSites   []int // sorted
    EnvironmentSites []int // sorted
    SubsystemDim     int   // 1 << len(SubsystemSites)
    EnvironmentDim   int   // 1 << len(EnvironmentSites)
}

func NewSiteSplit(subsystem, environment []int, numSites int) (SiteSplit, error)

// Decompose maps a full basis index q to its local subsystem index a
// and local environment index b, using the existing basis occupation
// accessor under the hood.
func (s SiteSplit) Decompose(q int, occ OccupationAccessor) (a, b int)
```

`OccupationAccessor` is a one-method interface so this file does not need to know `basis.go`'s internal representation:

```go
type OccupationAccessor interface {
    // Occupied reports whether site index is occupied in configuration q.
    Occupied(q, site int) bool
}
```

If `basis.go` already exposes an equivalent method on its existing basis type, that type should satisfy this interface directly (Go's implicit interface satisfaction) — no wrapper needed, no duplicate logic.

### 5.2 Conditional projector (precomputed grouping)

```go
// ConditionalProjector groups full basis indices by their environment
// sub-configuration, so building a conditional state at runtime is
// O(SubsystemDim) instead of O(FullDim).
type ConditionalProjector struct {
    Split SiteSplit
    // ByEnvironment[b] lists every (fullIndex, subsystemIndex) pair
    // whose environment part equals b.
    ByEnvironment [][]QAPair
}

type QAPair struct {
    FullIndex int
    SubIndex  int
}

func BuildConditionalProjector(split SiteSplit, occ OccupationAccessor, fullDim int) ConditionalProjector
```

### 5.3 Conditional state

```go
// ConditionalState is the (post-normalization) subsystem-A wave
// function induced by conditioning on a fixed environment
// configuration at one point in time.
type ConditionalState struct {
    Amplitudes []complex128 // length == SubsystemDim
    Norm       float64      // pre-normalization norm
    Valid      bool         // false => conditional norm failure
}
```

### 5.4 Trajectory state

```go
// TrajectoryState is the evolving actual Bell configuration for one
// sampled trajectory.
type TrajectoryState struct {
    Config int // current full configuration index Q(t)
}
```

Per-trajectory running counters live in the accumulator (§8), not here, to keep this struct minimal and to make "replay one trajectory" trivial for tests.

### 5.5 Precomputed psi(t) timeline (reused, not new physics)

```go
// PsiTimeline is the already-deterministic universal wavefunction
// history produced once by the existing evolve.go pipeline. The bridge
// phase consumes this; it must not recompute psi(t) per trajectory.
type PsiTimeline struct {
    Times []float64
    Psi   [][]complex128 // Psi[k] has length == full Hilbert dimension
}
```

If `evolve.go`/`audit.go` does not currently retain this across the full run, add the minimal change needed to retain and expose it (Assumption A4). This is the single most important reuse point in the whole ticket: it is what keeps trajectory sampling cheap (`O(trajectories × steps)` stochastic updates against one shared deterministic timeline) instead of re-deriving psi(t) `trajectories` times.

---

## 6. Algorithm: Bell trajectory sampling

Discrete-time thinning on the existing dt grid, as the ticket specifies, no Gillespie sampler. Rationale for not building a continuous-time sampler: `BELL-MIPT-0001` already fixes a small dt for the master-equation audit; reusing that same grid means the trajectory sampler needs zero new time-discretization logic and zero new convergence analysis. A continuous-time sampler would only be justified if dt were too coarse for `p_jump = 1 - exp(-lambda*dt)` to be a good approximation — that is a tuning question for the existing dt, not a reason to add a second time-stepping scheme in this ticket.

### 6.1 RNG policy

```text
Use math/rand/v2 with rand.NewPCG, seeded per trajectory as:
    src := rand.NewPCG(uint64(cfg.Seed), uint64(trajectoryIndex))
    rng := rand.New(src)

Never use a shared/global RNG source across trajectories. This is
required both for "deterministic trajectory output for fixed seed"
(test requirement) and for go test -race: a per-trajectory local
*rand.Rand has no shared mutable state.

Default to sequential trajectory execution (no goroutines) for v1.
Toy-scale trajectory counts (e.g. 200) and toy-scale Hilbert dimensions
do not need concurrency, and concurrency would reintroduce a
determinism/race-safety risk for no demonstrated benefit. Revisit only
if profiling on a real toy config shows this is a bottleneck — and if
so, parallelize per-trajectory only (independent RNG, independent
result slot, sequential reduction), never share mutable accumulator
state across goroutines.
```

### 6.2 Initial configuration sampling

```text
function SampleInitialConfig(psi0 []complex128, rng) -> q0:
    1. Build cumulative distribution over q from |psi0(q)|^2 once
       (shared across all trajectories — this is deterministic data,
       not trajectory-specific).
    2. Draw u ~ Uniform(0,1) from rng.
    3. Binary-search u into the cumulative distribution to get q0.
```

The cumulative distribution table is built once per run, not once per trajectory.

### 6.3 Per-step jump sampling

For each trajectory, walked along the shared `PsiTimeline`:

```text
function StepTrajectory(q, psiAtT, dt, rng) -> qNext:
    1. Compute outgoing rates sigma(n <- q) for all n != q, using the
       existing rate machinery from rates.go and psiAtT. (Reuse, do not
       reimplement Bell-rate math.)
    2. lambda := sum over n of sigma(n <- q)
    3. If lambda <= 0: return q unchanged (explicitly required by the
       "no jump occurs when total outgoing rate is zero" test).
    4. p_jump := 1 - exp(-lambda * dt)
    5. Draw u ~ Uniform(0,1) from rng.
    6. If u >= p_jump: return q unchanged.
    7. Otherwise draw destination n with probability sigma(n<-q)/lambda
       via cumulative draw over the same rate row, and return n.
```

Full trajectory loop:

```text
function RunTrajectory(timeline, rng, sampleEverySteps) -> samples:
    q := SampleInitialConfig(timeline.Psi[0], rng)
    samples := []
    for k, t in enumerate(timeline.Times):
        if k % sampleEverySteps == 0:
            samples.append({stepIndex: k, time: t, config: q})
        if k is not the last step:
            dt := timeline.Times[k+1] - timeline.Times[k]
            q := StepTrajectory(q, timeline.Psi[k], dt, rng)
    return samples
```

This produces exactly the sampled-step granularity needed by §8; per-step "micro jumps" (every dt) are still individually visible inside this loop and are what feed `mean_jump_count` etc. (see §8.1 for the resolved granularity question).

---

## 7. Algorithm: conditional subsystem wave function

Given `psi(t)` (full vector) and the environment sub-configuration `b0` realized at that time:

```text
function Condition(projector, psi, b0, normThreshold) -> ConditionalState:
    amp := zero vector of length projector.Split.SubsystemDim
    normSq := 0
    for (fullIndex, subIndex) in projector.ByEnvironment[b0]:
        amp[subIndex] := psi[fullIndex]
        normSq += |psi[fullIndex]|^2
    norm := sqrt(normSq)
    if norm < normThreshold:
        return ConditionalState{Amplitudes: amp, Norm: norm, Valid: false}
    scale := 1 / norm
    for i in range(amp): amp[i] *= scale
    return ConditionalState{Amplitudes: amp, Norm: norm, Valid: true}
```

This directly satisfies:

```text
"conditional state vector has dimension 2^|A|"        -> len(amp) == SubsystemDim
"conditional state normalizes when norm > threshold"   -> Valid branch
"conditional norm failure recorded when norm too small" -> Valid=false branch
```

Fidelity between two conditional states of the same subsystem dimension:

```text
function Fidelity(prev, curr ConditionalState) -> (float64, ok bool):
    if !prev.Valid || !curr.Valid:
        return 0, false   // undefined; caller must NOT average this in
    overlap := sum_i conj(prev.Amplitudes[i]) * curr.Amplitudes[i]
    f := |overlap|^2
    f = clamp(f, 0, 1)    // guards against floating-point overshoot past 1
    return f, true
```

Returning `ok=false` rather than a sentinel float is deliberate: it forces every call site to make an explicit decision about norm-failure handling instead of silently folding a failure into an average (see Risk R3).

---

## 8. Bridge metrics

### 8.1 Jump classification — resolved design decision

The ticket's four categories (`environment_jump`, `subsystem_jump`, `any_jump`, `no_jump`) are **not mutually exclusive in general**, because a single Bell jump operator can in principle change occupation on both sides of the A/B partition at once (e.g. a hopping or pairing term whose two sites straddle the boundary). The plan resolves this explicitly rather than leaving it implicit:

```go
type StepClass struct {
    EnvironmentChanged bool // Q_B(t) != Q_B(t+dt)
    SubsystemChanged   bool // Q_A(t) != Q_A(t+dt)
}

func ClassifyTransition(split SiteSplit, occ OccupationAccessor, qPrev, qCurr int) StepClass
```

Bucket semantics used everywhere downstream:

```text
"at environment jump"        := EnvironmentChanged == true   (SubsystemChanged may also be true)
"without environment jump"   := EnvironmentChanged == false  (covers subsystem-only jumps AND no-jump steps)
"any jump"                   := EnvironmentChanged || SubsystemChanged
"no jump"                    := !EnvironmentChanged && !SubsystemChanged
```

This is what makes `mean_fidelity_drop_at_environment_jumps` vs `mean_fidelity_drop_without_environment_jumps` a clean, well-defined partition of all sampled transitions (every transition falls into exactly one of "at" / "without" with respect to the environment), while `any_jump` / `no_jump` remain available as a second, coarser diagnostic axis.

Second resolved ambiguity: classification and fidelity-drop bucketing happen at **sampled-step granularity** (i.e. comparing `Q` at consecutive recorded samples, spaced by `sample_every_steps`), per the ticket's literal wording ("at each sampled step ... classify each transition"). `mean_jump_count`, `mean_environment_jump_count`, `mean_subsystem_jump_count` are defined instead at **micro-step granularity** (every internal dt step of §6.3), since that is the native unit of "a Bell jump happened" and is independent of the sampling cadence used for the conditional-state/fidelity audit. These two granularities are documented as intentionally different views of the same trajectories, not an inconsistency.

### 8.2 Accumulator

```go
// BridgeAccumulator collects everything needed to populate
// BridgeMetrics after all trajectories have run. All sums use
// float64; counts use int64 to avoid overflow concerns at toy scale.
type BridgeAccumulator struct {
    TrajectoryCount int

    // Micro-step jump tallies (per trajectory, summed then divided by
    // TrajectoryCount to produce the "mean_*" metrics).
    TotalJumpCount            int64
    TotalEnvironmentJumpCount int64
    TotalSubsystemJumpCount   int64

    ConditionalNormFailures int64

    SumFidelityDropAtEnvJump   float64
    CountFidelityDropAtEnvJump int64

    SumFidelityDropNoEnvJump   float64
    CountFidelityDropNoEnvJump int64

    SumFidelityDropAnyJump   float64
    CountFidelityDropAnyJump int64

    SumFidelityDropNoJump   float64
    CountFidelityDropNoJump int64
}

func (acc *BridgeAccumulator) RecordMicroStep(class StepClass)
func (acc *BridgeAccumulator) RecordSampledTransition(class StepClass, drop float64, dropOK bool)
func (acc *BridgeAccumulator) Finalize() BridgeMetrics // also needs the equivariance tracker, see below
```

`RecordSampledTransition` must be a no-op with respect to all fidelity sums when `dropOK == false` (a norm failure on either endpoint), but must still increment `ConditionalNormFailures` exactly once per offending endpoint state, counted once even if that state is shared by two adjacent transitions (track failures by sampled-state index, not by transition, to avoid double counting).

### 8.3 Conditional update ratio

```text
function ConditionalUpdateRatio(acc) -> *float64:
    if acc.CountFidelityDropAtEnvJump == 0: return nil
    if acc.CountFidelityDropNoEnvJump == 0: return nil
    meanAt := acc.SumFidelityDropAtEnvJump / acc.CountFidelityDropAtEnvJump
    meanWithout := acc.SumFidelityDropNoEnvJump / acc.CountFidelityDropNoEnvJump
    if meanWithout < ratioDenominatorEpsilon: return nil   // avoid blow-up
    ratio := meanAt / meanWithout
    return &ratio
```

`nil` propagates to JSON `null`, matching the ticket's explicit "report null or mark ratio as unavailable" instruction.

### 8.4 Empirical trajectory equivariance

```go
// EquivarianceTracker accumulates, across trajectories, how many
// trajectories landed in each full configuration q at each recorded
// sample time, for later comparison against |psi(t)|^2.
type EquivarianceTracker struct {
    SampleStepIndices []int       // which timeline steps were sampled
    FullDim           int
    Counts            [][]int64   // Counts[sampleIdx][q]
}

func NewEquivarianceTracker(sampleStepIndices []int, fullDim int) *EquivarianceTracker
func (e *EquivarianceTracker) Record(sampleIdx int, q int)
func (e *EquivarianceTracker) L1(sampleIdx int, psiAtThatTime []complex128, trajectories int) float64
```

```text
L1(t) = sum_q | counts[q]/trajectories - |psi_q(t)|^2 |
```

`max_empirical_equivariance_l1` and `final_empirical_equivariance_l1` are simply the max and last value of this series across all sampled steps. This is explicitly a noisy, finite-sample diagnostic — see §10/§12 for how it feeds (or, more precisely, does *not* directly feed) `bridge_status`.

---

## 9. Report schema changes

### 9.1 Backward compatibility requirement, stated precisely

The ticket's test "bridge_disabled appears when bridge omitted" means the **report's** bridge section must always be present and populated with `bridge_status: "bridge_disabled"` even though the **input config's** bridge section was omitted. This is the opposite of `omitempty` at the top level:

```go
type Report struct {
    // ... all existing fields unchanged, same names/tags/order ...
    ToyAnalysisOnly bool   `json:"toy_analysis_only"`
    PhysicsClaim    string `json:"physics_claim"`
    GoalStatus      string `json:"goal_status"`
    // ...

    Bridge BridgeReport `json:"bridge"` // always present, never omitted
}
```

### 9.2 Bridge report types

```go
type BridgeReport struct {
    Enabled          bool                  `json:"enabled"`
    BridgeGoal       string                `json:"bridge_goal"`
    BridgeStatus     string                `json:"bridge_status"`
    Trajectories     int                   `json:"trajectories"`
    SubsystemSites   []int                 `json:"subsystem_sites,omitempty"`
    EnvironmentSites []int                 `json:"environment_sites,omitempty"`
    SampleEverySteps int                   `json:"sample_every_steps,omitempty"`
    Metrics          *BridgeMetrics        `json:"metrics,omitempty"`
    Interpretation   *BridgeInterpretation `json:"interpretation,omitempty"`
}

type BridgeMetrics struct {
    TrajectoryCount                         int      `json:"trajectory_count"`
    MeanJumpCount                           float64  `json:"mean_jump_count"`
    MeanEnvironmentJumpCount                float64  `json:"mean_environment_jump_count"`
    MeanSubsystemJumpCount                  float64  `json:"mean_subsystem_jump_count"`
    ConditionalNormFailures                 int64    `json:"conditional_norm_failures"`
    MeanFidelityDropAtEnvironmentJumps      *float64 `json:"mean_fidelity_drop_at_environment_jumps"`
    MeanFidelityDropWithoutEnvironmentJumps *float64 `json:"mean_fidelity_drop_without_environment_jumps"`
    MeanFidelityDropAtAnyJumps              *float64 `json:"mean_fidelity_drop_at_any_jumps"`
    MeanFidelityDropNoJump                  *float64 `json:"mean_fidelity_drop_no_jump"`
    ConditionalUpdateRatio                  *float64 `json:"conditional_update_ratio"`
    MaxEmpiricalEquivarianceL1              float64  `json:"max_empirical_equivariance_l1"`
    FinalEmpiricalEquivarianceL1            float64  `json:"final_empirical_equivariance_l1"`
}

type BridgeInterpretation struct {
    MonitoringLikeSignal string `json:"monitoring_like_signal"` // see §10
    Reason                string `json:"reason"`
}
```

Design notes:

```text
- BridgeGoal is a static descriptive string, always the same value
  ("sample_bell_trajectories_and_audit_conditional_subsystem_state"),
  present even when disabled — it describes what the section would do,
  not a per-run outcome.
- Trajectories (top-level) echoes the configured count; Metrics.TrajectoryCount
  is the actually-completed count used in every average. If a trajectory
  fails mid-run, that is an implementation-level failure (-> bridge_toy_failed,
  see §10), not something silently absorbed by shrinking TrajectoryCount.
  Fail loud, do not soft-discard data quality problems.
- Metrics and Interpretation are pointers so a disabled bridge serializes
  metrics/interpretation as JSON null rather than a misleading zero-valued
  block that could be misread as "zero jumps were observed."
- *float64 fields use no struct-tag omitempty, so a nil pointer marshals
  to JSON null and the key is never dropped, matching the spec's example
  schema where these keys are always present.
```

### 9.3 Markdown rendering

`markdown.go` gets one new section, rendered unconditionally, that prints `Enabled`/`BridgeStatus` always, and the metrics table only when `Metrics != nil`. The existing limitations block gains the seven required limitation lines verbatim (§ "Required Limitations" in the ticket) whenever `Bridge.Enabled == true`; when disabled, the existing 0001 limitations text is untouched.

---

## 10. Goal/bridge status logic

`goal_status` (existing, from `BELL-MIPT-0001`'s master-equation/equivariance audit) is **untouched** by this ticket — no new branch, no new input. This is stated explicitly here because it is the kind of coupling a careless extension could introduce by accident.

`bridge_status` is computed independently:

```text
function DetermineBridgeStatus(cfg, run) -> string:
    if !cfg.Bridge.Enabled:
        return "bridge_disabled"
    if run.HasImplementationFailure:
        return "bridge_toy_failed"
    if run.ConditionalNormFailureRate > maxConditionalNormFailureRate:
        return "bridge_toy_inconclusive"
    if run.EnvironmentJumpSampleCount < minEnvironmentJumpSamples:
        return "bridge_toy_inconclusive"
    if run.MaxEmpiricalEquivarianceL1 > noisyEquivarianceThreshold(dim, trajectories):
        return "bridge_toy_inconclusive"
    return "bridge_toy_passed"
```

`HasImplementationFailure` covers: a panic recovered during the bridge phase, NaN/Inf appearing in any `psi` or conditional amplitude, a negative total outgoing rate, a dimension mismatch between `SubsystemDim` and an actual conditional amplitude vector, or a trajectory that did not complete its sample sequence.

**Explicit rule, called out because the ticket calls it out**: `conditional_update_ratio` plays **no role** in this function. A ratio greater than 1 is a descriptive signal, fed only into `Interpretation`, never into `bridge_status`. This keeps "the audit ran cleanly" (status) fully decoupled from "the audit found something interesting" (interpretation).

`Interpretation.MonitoringLikeSignal` is computed separately and only consulted by humans:

```text
function ClassifySignal(ratio *float64) -> (signal, reason):
    if ratio == nil:
        return "not_assessed",
               "conditional_update_ratio unavailable (too few environment-jump
                samples, or non-environment-jump fidelity drop too small to
                divide by safely)."
    if *ratio < signalLowThreshold:
        return "no_clear_signal", "<reason text>"
    if *ratio < signalCandidateThreshold:
        return "weak_signal", "<reason text>"
    return "candidate_signal",
           "Descriptive only: elevated ratio in this finite toy run. Not a
            statistical test, not evidence of a Bell-MIPT bridge, not a
            measurement claim. See needNullModel (still unpaid)."
```

Every branch's `reason` text must itself avoid the forbidden phrases — write the reason strings once, in `bridge_report.go`, and have `forbidden.go`'s scan cover them like any other report text (see Risk R7).

---

## 11. EBP debt update

Resolved rule for when the "paid-down" debt status applies: whenever the bridge mechanism **ran** without an implementation-level failure — i.e. `bridge_status` is `bridge_toy_passed` **or** `bridge_toy_inconclusive` — because `needMap`/`needToyCheck`/`needInvariant` are about whether the toy mechanism exists and executes, not about whether its output was statistically clean. Only `bridge_toy_failed` (or `bridge_disabled`) leaves debt status at the pre-0002A values.

If bridge ran (passed or inconclusive):

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

If bridge disabled or failed, retain the pre-0002A debt block unchanged.

This block is emitted from the same place `bridge_status` is computed, as a pure function of `bridge_status`, so the two can never drift apart.

---

## 12. Proposed constants / defaults (single reference table)

All of these are judgment calls the ticket leaves open. Centralizing them here — and as named constants in `bridge_report.go`, not scattered magic numbers — makes them easy to recalibrate in a follow-up review without hunting through files.

```text
conditionalNormThreshold        = 1e-6   (below this, conditional norm failure)
maxConditionalNormFailureRate   = 0.01   (1% of sampled conditional states)
minEnvironmentJumpSamples       = 30     (conventional small-sample floor)
ratioDenominatorEpsilon         = 1e-9   (guards conditional_update_ratio blow-up)
signalLowThreshold              = 1.5    (ratio below this => no_clear_signal)
signalCandidateThreshold        = 3.0    (ratio below this => weak_signal, else candidate_signal)
noisyEquivarianceThreshold(D,N) = 3 * sqrt(D / N)
                                  (heuristic multinomial-noise scale; NOT a
                                  rigorous test, used only to gate
                                  bridge_toy_inconclusive, never to promote
                                  a result to "passed")
```

None of these are physics constants; all are toy-instrumentation tuning knobs and should be reviewed the same way the TC-1A/TC-1B reviews flagged underspecified predicates — as explicit, named, and revisitable, not buried.

---

## 13. Required tests

Organized by file, one Go test function per ticket-listed behavior.

### `bridge_config_test.go`

```text
TestBridgeConfig_OmittedDefaultsToDisabled
TestBridgeConfig_DisabledPreservesOldBehavior        (golden-file comparison of full 0001 report)
TestBridgeConfig_ValidateRejectsOverlap
TestBridgeConfig_ValidateRejectsMissingSite
TestBridgeConfig_ValidateAcceptsValidSplit
TestBridgeConfig_ValidateRejectsNonPositiveTrajectories
TestBridgeConfig_ValidateRejectsNonPositiveSampleEverySteps
```

### `trajectory_test.go`

```text
TestSampleInitialConfig_MatchesPsiSquaredApproximately   (chi-square or L1 bound over many draws)
TestStepTrajectory_JumpProbabilityUsesTotalOutgoingRate
TestStepTrajectory_DestinationProportionalToRates
TestStepTrajectory_NoJumpWhenTotalRateZero
TestRunTrajectory_DeterministicForFixedSeed              (same seed+index => identical sample sequence, run twice)
```

### `site_split_test.go` / `conditional_test.go`

```text
TestSiteSplit_DecomposeMatchesManualPartition
TestConditionalState_DimensionEqualsTwoToSubsystemSize
TestConditionalState_NormalizesAboveThreshold
TestConditionalState_NormFailureRecordedBelowThreshold
TestFidelity_IdenticalStatesEqualsOne
TestFidelity_WithinZeroOneTolerance                      (including a near-1 floating point overshoot case)
TestFidelity_UndefinedWhenEitherStateInvalid
```

### `bridgemetrics_test.go`

```text
TestClassifyTransition_EnvironmentJumpDetected
TestClassifyTransition_SubsystemJumpDetected
TestClassifyTransition_BothChangedSimultaneously          (boundary-crossing jump case, see §8.1)
TestClassifyTransition_NoJump
TestBridgeAccumulator_BucketsFidelityDropsCorrectly
TestConditionalUpdateRatio_NilOnZeroDenominator
TestConditionalUpdateRatio_NilOnTooFewEnvironmentJumpSamples
TestEquivarianceTracker_L1ComputedAgainstPsiSquared
TestDetermineBridgeStatus_PassedWhenCleanRun
TestDetermineBridgeStatus_FailedOnImplementationFailure
TestDetermineBridgeStatus_InconclusiveOnTooFewEnvironmentJumps
TestDetermineBridgeStatus_InconclusiveOnNoisyEquivariance
TestDetermineBridgeStatus_IgnoresConditionalUpdateRatio    (regression test for the explicit "ratio>1 is not proof" rule)
```

### `report_test.go` / `markdown_test.go`

```text
TestReport_OldFieldsBackwardCompatible
TestReport_BridgeDisabledAppearsWhenBridgeOmittedFromConfig
TestReport_BridgeMetricsAppearWhenEnabled
TestReport_LimitationsIncludeRequiredBridgeDisclaimers
TestForbiddenAudit_PassesWithBridgeSectionPresent
TestForbiddenAudit_CatchesInjectedForbiddenPhraseInBridgeReason  (negative test: deliberately insert a forbidden phrase into Interpretation.Reason and confirm the audit catches it)
```

---

## 14. Validation commands

Exactly as specified by the ticket, run in this order:

```bash
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge
```

Additional non-mandatory checks worth running before sign-off, since they catch issues the above commands won't:

```bash
gofmt -l .                     # formatting drift
diff <(jq .bridge /tmp/bellmipt-default/report.json) <(echo '{"enabled":false,...}')
                                # sanity-check the disabled-bridge report shape by hand
```

`bellmipt_bridge.json` must exist at repo root, with `bridge.enabled: true` and a valid, fully-partitioned site split, sized for fast CI (small chain length, modest trajectory count — e.g. the 200-trajectory example from the ticket's proposed config is a reasonable default).

---

## 15. Known risks and open design decisions

```text
R1. Basis indexing assumption (Assumption A1). If basis.go uses a
    restricted-particle-number combinatorial index rather than a raw
    site bitmask, Decompose() must still work correctly — it is
    designed around an occupation accessor specifically to avoid
    assuming bitmask layout. Verify this against real source before
    coding §5.1.

R2. Jump-classification granularity (§8.1) is genuinely ambiguous in
    the ticket text and has been resolved here by an explicit design
    choice (sampled-step granularity for fidelity bucketing, micro-step
    granularity for jump counts). Flag this resolution for review
    rather than treating it as obviously correct.

R3. Conditional norm failures must never be silently folded into
    fidelity averages as 0 or 1 — Fidelity() returns ok=false and every
    caller must branch on it. A missed branch here would quietly bias
    conditional_update_ratio without any test catching it unless
    TestFidelity_UndefinedWhenEitherStateInvalid is honored at every
    call site, not just in isolation.

R4. conditionalNormThreshold, maxConditionalNormFailureRate,
    minEnvironmentJumpSamples, and noisyEquivarianceThreshold are all
    heuristic (see §12). They are defensible starting points, not
    derived results. A future ticket calibrating these against
    needNullModel work should be expected.

R5. RNG/determinism/concurrency: sequential trajectory execution is the
    recommended default specifically to keep go test -race trivially
    clean and determinism trivially provable. Do not parallelize
    trajectories in this ticket; if a future performance need arises,
    parallelize with fully independent per-trajectory RNG and
    independent result slots, never a shared accumulator written from
    multiple goroutines.

R6. Boundary-crossing jump operators (a single Bell jump changing
    occupation on both sides of the partition at once) are possible
    depdepending on which terms in the existing Hamiltonian generate
    Bell currents across site boundaries. §8.1's non-exclusive boolean
    classification handles this correctly; a naive switch/exhaustive-
    enum classification would not. Confirm with a deliberate
    TestClassifyTransition_BothChangedSimultaneously test using a
    constructed pair of configurations, not just configurations that
    happen to arise during a real run.

R7. forbidden.go must scan the fully assembled report (including the
    new Bridge section and Interpretation.Reason strings), not a
    hardcoded subset of legacy fields. If forbidden.go currently only
    walks specific known fields rather than the whole serialized
    report, this is a required (small) modification, not optional —
    otherwise the new Interpretation.Reason free-text field becomes an
    unaudited channel for accidental physics-promotion language.

R8. Performance/memory: EquivarianceTracker stores
    O(numSampleSteps x FullDim) int64 counters. Fine at toy scale (e.g.
    FullDim <= 2^8, a few hundred sample steps); flag as a scaling
    limit if a future ticket increases chain length substantially.

R9. trajectory_count vs configured trajectories divergence: per §9.2,
    any trajectory that fails to complete is treated as an
    implementation failure (-> bridge_toy_failed), not silently
    excluded from the denominator. This is a deliberate "fail loud"
    choice and should be called out in code review if anyone proposes
    softening it.
```

---

## 16. Suggested build order

```text
1. site_split.go + tests                  (pure data-structure logic, no dependency on rates/evolve)
2. bridge_config.go + tests               (config parsing/validation, unlocks early integration testing)
3. conditional.go + tests                 (depends only on §5.1/§5.2 plus a stub psi vector)
4. trajectory.go + tests                  (depends on rates.go accessor from Assumption A2)
5. bridgemetrics.go + tests               (depends on §6 output shape and §7 ConditionalState)
6. equivariance_empirical.go + tests
7. bridge_report.go + status logic + tests
8. Wire into run.go, report.go, markdown.go, forbidden.go
9. bellmipt_bridge.json fixture + end-to-end validation commands (§14)
```

This order front-loads the parts with no dependency on verifying Assumptions A1–A4, so work can start immediately while the basis/rates/evolve accessor questions (§2) are confirmed against real source in parallel.

---

## 17. Recommendation: extend in place vs. separate package

**Recommendation: implement as new files inside the existing `internal/bellmipt` package, not as a new sub-package.**

Reasoning:

```text
- The ticket is explicit: extend, do not rewrite, and do not add
  subcommands. A new sub-package (e.g. internal/bellmipt/bridge) would
  force several currently-unexported types and functions in basis.go,
  rates.go, and evolve.go to become exported purely to cross a package
  boundary that exists for no functional reason — inflating the public
  surface of a toy-scope package for a single internal consumer.

- The bridge phase is not an independent concern; it is a second
  consumer of the exact same Hilbert-space, rate, and evolution
  machinery the 0001 audit already uses on the same data, in the same
  run, writing into the same report. Splitting it into a separate
  package buys module boundary discipline that this project does not
  need yet, at the cost of import-direction complexity (bridge package
  importing bellmipt, or vice versa, plus whatever accessor types must
  be threaded through).

- Staying in one package keeps `go vet ./...` and `go test -race ./...`
  scoped exactly as the existing validation commands expect, with no
  new package-boundary edge cases to reason about.

Revisit this decision only if a later ticket needs the trajectory/
conditional-state machinery reused against a different toy model
independent of this specific Kitaev-chain setup. At that point,
extracting bridge_*.go and site_split.go into a package with a small,
deliberately exported interface (PsiTimeline, OccupationAccessor,
RateRowFunc) would be the right move — but that is future work, not
part of BELL-MIPT-0002A.
```

---

*End of plan. No MIPT claim, no holography claim, no Bell-jumps-equal-measurements claim, no physics promotion is made anywhere in this document — every status/metric/interpretation field above is designed specifically so the implementation cannot accidentally make one either.*
