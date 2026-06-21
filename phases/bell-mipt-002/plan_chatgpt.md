# Implementation Plan: BELL-MIPT-0002A — Bell Trajectory + Conditional Subsystem State Audit

## 1. Ticket Summary

Ticket:

```text
BELL-MIPT-0002A
```

Title:

```text
Bell trajectory sampler + conditional subsystem state audit
```

Goal:

```text
Extend the existing Go Bell-rate toy so it can sample actual Bell configuration histories and compute conditional subsystem wave functions induced by the actual environment configuration.
```

Central question:

```text
Do Bell jumps in the environment induce measurable changes in the subsystem conditional state, beyond ordinary smooth Schrödinger evolution?
```

Allowed conclusion if successful:

```text
Bell trajectories were sampled, equivariance was checked empirically, and conditional subsystem-state changes were measured for the tested finite toy configuration.
```

Forbidden conclusions:

```text
No MIPT claim.
No holography claim.
No Bell-jumps-equal-measurements claim.
No Bell-MIPT bridge claim.
No Bohmian-mechanics validation claim.
No physics promotion.
```

This ticket remains a finite toy check. It extends the current Bell-rate/equivariance audit from distribution-level dynamics into sampled Bell configuration histories and conditional subsystem-state diagnostics.

---

## 2. Implementation Strategy

Implement `BELL-MIPT-0002A` as a direct extension of the existing command:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

Do not add subcommands.

Do not create a separate CLI.

Do not alter the existing `BELL-MIPT-0001` behavior when the new bridge section is omitted or disabled.

The new bridge logic should live under `internal/bellmipt` as separate files and package-level functions. The command should remain thin and should call the existing run path.

Recommended approach:

```text
cmd/bellmipt remains unchanged except for invoking the extended Run function.

internal/bellmipt keeps existing BELL-MIPT-0001 logic.

New bridge-specific code is added under internal/bellmipt:
- bridge_config.go
- trajectory.go
- conditional.go
- bridge_metrics.go
- bridge_report.go
```

This keeps the one-command user experience while isolating the new trajectory/conditional-state logic from the older master-equation audit.

---

## 3. File Layout Changes

Current assumed layout:

```text
cmd/bellmipt/main.go

internal/bellmipt/config.go
internal/bellmipt/basis.go
internal/bellmipt/fermion.go
internal/bellmipt/hamiltonian.go
internal/bellmipt/current.go
internal/bellmipt/rates.go
internal/bellmipt/evolve.go
internal/bellmipt/audit.go
internal/bellmipt/report.go
internal/bellmipt/markdown.go
internal/bellmipt/forbidden.go
internal/bellmipt/run.go
```

Add:

```text
internal/bellmipt/bridge_config.go
internal/bellmipt/trajectory.go
internal/bellmipt/conditional.go
internal/bellmipt/bridge_metrics.go
internal/bellmipt/bridge_report.go

internal/bellmipt/bridge_config_test.go
internal/bellmipt/trajectory_test.go
internal/bellmipt/conditional_test.go
internal/bellmipt/bridge_metrics_test.go
internal/bellmipt/bridge_report_test.go
```

Optional testdata:

```text
internal/bellmipt/testdata/bellmipt_bridge_small.json
```

Optional example config at repo root or examples folder:

```text
examples/bellmipt_bridge.json
```

Do not add:

```text
No web UI.
No database.
No agent runtime.
No MIPT package.
No trajectory visualization package.
```

---

## 4. Config Extension

Extend the existing `Config` struct with an optional bridge section:

```go
type Config struct {
    SchemaVersion string       `json:"schema_version"`
    Model         string       `json:"model"`
    Sites         int          `json:"sites"`
    Boundary      string       `json:"boundary"`
    Parameters    Parameters   `json:"parameters"`
    InitialState  InitialState `json:"initial_state"`
    Time          TimeConfig   `json:"time"`
    Audit         AuditConfig  `json:"audit"`
    Bridge        *BridgeConfig `json:"bridge,omitempty"`
}
```

New bridge config:

```go
type BridgeConfig struct {
    Enabled          bool  `json:"enabled"`
    SubsystemSites   []int `json:"subsystem_sites"`
    EnvironmentSites []int `json:"environment_sites"`
    Trajectories     int   `json:"trajectories"`
    Seed             int64 `json:"seed"`
    SampleEverySteps int   `json:"sample_every_steps"`
}
```

Example enabled config:

```json
{
  "bridge": {
    "enabled": true,
    "subsystem_sites": [0, 1, 2],
    "environment_sites": [3, 4, 5],
    "trajectories": 200,
    "seed": 777,
    "sample_every_steps": 1
  }
}
```

Default behavior:

```text
bridge omitted      → bridge disabled
bridge.enabled=false → bridge disabled
bridge.enabled=true  → run trajectory + conditional-state audit
```

The built-in default config should keep bridge disabled.

---

## 5. Bridge Config Validation

Add validation rules:

```text
If bridge is omitted:
  valid, disabled.

If bridge.enabled=false:
  valid, disabled.

If bridge.enabled=true:
  subsystem_sites must be non-empty.
  environment_sites must be non-empty.
  subsystem_sites and environment_sites must not overlap.
  subsystem_sites and environment_sites together must cover every site from 0 to sites-1.
  no site may be negative.
  no site may be >= sites.
  trajectories must be > 0.
  sample_every_steps must be >= 1.
```

Reject invalid configs before running the simulation.

Validation function:

```go
func ValidateBridgeConfig(cfg Config) error
```

Useful helper:

```go
func BridgeEnabled(cfg Config) bool
```

Partition metadata:

```go
type SitePartition struct {
    SubsystemSites   []int
    EnvironmentSites []int
    FullToSubsystem  map[int]int
    FullToEnvironment map[int]int
}
```

The partition should preserve the site order supplied in config, but tests should make the behavior explicit.

Recommendation:

```text
For v0, sort subsystem_sites and environment_sites ascending internally.
```

Reason:

```text
This makes subsystem/environment bit packing deterministic and easier to audit.
```

---

## 6. New Data Structures

### 6.1 Trajectory State

```go
type TrajectoryState struct {
    TrajectoryID int
    CurrentQ     uint64
    PreviousQ    uint64
    JumpCount    int
    EnvironmentJumpCount int
    SubsystemJumpCount   int
}
```

### 6.2 Trajectory Step Record

Avoid writing every step by default unless needed. Keep records in memory only for metrics.

```go
type TrajectoryStepSample struct {
    Step             int
    Time             float64
    Q                uint64
    PreviousQ        uint64
    AnyJump          bool
    EnvironmentJump  bool
    SubsystemJump    bool
    ConditionalNorm  float64
    Fidelity         *float64
    FidelityDrop     *float64
}
```

Use pointers or nullable wrappers for metrics that are unavailable at the first sampled step.

### 6.3 Conditional State

```go
type ConditionalState struct {
    EnvConfig uint64
    Vector    []complex128
    Norm      float64
    Normalized bool
}
```

The vector dimension is:

```text
2 ^ len(subsystem_sites)
```

### 6.4 Bridge Run Result

```go
type BridgeRunResult struct {
    Enabled       bool
    Status        string
    Metrics       BridgeMetrics
    Interpretation BridgeInterpretation
    Limitations   []string
    Warnings      []string
}
```

### 6.5 Bridge Metrics

```go
type BridgeMetrics struct {
    TrajectoryCount int `json:"trajectory_count"`

    MeanJumpCount            float64 `json:"mean_jump_count"`
    MeanEnvironmentJumpCount float64 `json:"mean_environment_jump_count"`
    MeanSubsystemJumpCount   float64 `json:"mean_subsystem_jump_count"`

    ConditionalNormFailures int `json:"conditional_norm_failures"`

    MeanFidelityDropAtEnvironmentJumps      *float64 `json:"mean_fidelity_drop_at_environment_jumps"`
    MeanFidelityDropWithoutEnvironmentJumps *float64 `json:"mean_fidelity_drop_without_environment_jumps"`
    MeanFidelityDropAtAnyJumps              *float64 `json:"mean_fidelity_drop_at_any_jumps"`
    MeanFidelityDropNoJump                  *float64 `json:"mean_fidelity_drop_no_jump"`

    ConditionalUpdateRatio *float64 `json:"conditional_update_ratio"`

    MaxEmpiricalEquivarianceL1   float64 `json:"max_empirical_equivariance_l1"`
    FinalEmpiricalEquivarianceL1 float64 `json:"final_empirical_equivariance_l1"`

    InitialEmpiricalEquivarianceL1 float64 `json:"initial_empirical_equivariance_l1"`

    EnvironmentJumpTransitions int `json:"environment_jump_transitions"`
    NonEnvironmentJumpTransitions int `json:"non_environment_jump_transitions"`
    AnyJumpTransitions int `json:"any_jump_transitions"`
    NoJumpTransitions int `json:"no_jump_transitions"`
}
```

Use `null` in JSON for unavailable means or ratios.

---

## 7. Bell Trajectory Sampling Algorithm

### 7.1 Initial Configuration Sampling

For each trajectory:

```text
1. Compute p(q) = |psi_0(q)|².
2. Sample Q_0 from p(q).
3. Store Q_0 as CurrentQ.
```

Function:

```go
func SampleDiscrete(prob []float64, rng *rand.Rand) (int, error)
```

Requirements:

```text
prob must be finite.
prob must be nonnegative within tolerance.
prob sum must be approximately 1.
sampling must be deterministic for fixed seed.
```

Initial empirical check:

```text
Across all trajectories, compute empirical distribution of Q_0.
Compare against |psi_0|² using L1 distance.
Report as initial_empirical_equivariance_l1.
```

This is descriptive, not a strict pass/fail condition unless sampling is broken.

---

### 7.2 Jump Sampling at Each Time Step

At each time step, for each trajectory with current actual configuration `q`:

```text
1. Compute Bell current J from current psi.
2. Compute Bell rates sigma(n <- m) from J and psi.
3. Read outgoing rates sigma(n <- q) for all n.
4. lambda = sum_n sigma(n <- q).
5. p_jump = 1 - exp(-lambda * dt).
6. Draw u ~ Uniform(0,1).
7. If u >= p_jump, no jump.
8. If u < p_jump, choose destination n with probability sigma(n <- q) / lambda.
```

Function:

```go
func StepBellConfiguration(
    q uint64,
    rates [][]float64,
    dt float64,
    rng *rand.Rand,
) (nextQ uint64, jumped bool, lambda float64, err error)
```

Important numerical guards:

```text
If lambda == 0:
  no jump.

If lambda < 0:
  error, rates invalid.

If p_jump is NaN or Inf:
  mark bridge inconclusive.

If p_jump > 1 due to numerical overflow:
  mark bridge inconclusive or clamp only with explicit warning.
```

For v0, discrete-time thinning is acceptable because the existing toy uses small fixed `dt`.

Do not implement Gillespie sampling in this ticket.

---

### 7.3 Coupling to Wavefunction Evolution

Use the same `psi(t)` trajectory as the master-equation audit.

Recommended implementation shape:

```text
Run the original BELL-MIPT-0001 master-equation audit first or as the primary evolution loop.

If bridge is enabled:
  during the same step loop, update all sampled actual configurations using the current psi and current rates.
```

Two acceptable designs:

#### Option A — Integrated loop

One simulation loop evolves:

```text
psi
rho master equation
trajectory configurations
bridge metrics
```

Benefit:

```text
Avoids recomputing psi history.
```

Risk:

```text
Touches existing BELL-MIPT-0001 loop more heavily.
```

#### Option B — Two-pass loop with stored psi snapshots

First pass:

```text
Run existing BELL-MIPT-0001 audit and store psi snapshots at every step or every sample step.
```

Second pass:

```text
Use psi snapshots to sample trajectories and conditional states.
```

Benefit:

```text
Preserves old behavior more safely.
```

Risk:

```text
Stores O(steps * dim) complex values.
```

Recommendation for v0:

```text
Use Option B for safety and backward compatibility.
```

Reason:

```text
BELL-MIPT-0001 is already accepted. BELL-MIPT-0002A should minimize risk of altering its behavior. For small toy dimensions and steps, storing psi snapshots is acceptable.
```

Memory estimate:

```text
sites=6, dim=64, steps=1000:
1001 * 64 * complex128 ≈ 1 MB.
```

Add guardrails if needed:

```text
If dim * steps is too large for snapshot mode, reject bridge run or mark inconclusive.
```

---

## 8. Conditional Subsystem Wave Function

### 8.1 Bit Splitting

Given full configuration `q`, split into subsystem and environment packed bitstrings.

Function:

```go
func SplitConfig(q uint64, partition SitePartition) (subQ uint64, envQ uint64)
```

Example:

```text
full sites: [0,1,2,3,4,5]
subsystem_sites: [0,1,2]
environment_sites: [3,4,5]

q bits: site 0,1,2 go into subQ bits 0,1,2
        site 3,4,5 go into envQ bits 0,1,2
```

Also need inverse packing:

```go
func CombineConfig(subQ uint64, envQ uint64, partition SitePartition) uint64
```

Tests should verify round-trip:

```text
CombineConfig(SplitConfig(q)) == q
```

---

### 8.2 Conditional State Construction

Given universal wavefunction `psi` over full basis and actual environment configuration `envQ`:

```text
For every subsystem configuration a:
  fullQ = CombineConfig(a, envQ)
  conditional[a] = psi[fullQ]
```

Then normalize:

```text
norm = sqrt(sum_a |conditional[a]|²)
if norm <= conditional_norm_floor:
  record conditional norm failure
else:
  conditional[a] /= norm
```

Function:

```go
func ConditionalSubsystemState(
    psi []complex128,
    envQ uint64,
    partition SitePartition,
) (ConditionalState, error)
```

Add constant:

```go
const ConditionalNormFloor = 1e-14
```

If below floor:

```text
Do not panic.
Return a state marked Normalized=false.
Increment conditional_norm_failures.
Skip fidelity calculation for that transition.
```

---

## 9. Conditional Fidelity and Fidelity Drop

Given two normalized conditional states:

[
F = |\langle \psi_A(t) | \psi_A(t+\Delta t) \rangle|^2
]

Function:

```go
func ConditionalFidelity(a, b []complex128) (float64, error)
```

Clamp only for tiny numerical drift:

```text
If F = 1 + 1e-13, clamp to 1 and track no failure.
If F < -1e-12 or F > 1+1e-12, return error or mark bridge inconclusive.
```

Fidelity drop:

```text
drop = 1 - F
```

It should satisfy:

```text
0 <= drop <= 1
```

within numerical tolerance.

---

## 10. Jump Classification

For each trajectory transition:

```text
previousQ -> currentQ
```

Compute:

```go
prevSubQ, prevEnvQ := SplitConfig(previousQ, partition)
nextSubQ, nextEnvQ := SplitConfig(currentQ, partition)
```

Classify:

```text
any_jump = previousQ != currentQ
environment_jump = prevEnvQ != nextEnvQ
subsystem_jump = prevSubQ != nextSubQ
no_jump = previousQ == currentQ
```

Important:

```text
A single full-configuration jump can affect subsystem only, environment only, or both.
```

So the categories are not mutually exclusive except `no_jump`.

Metrics should count:

```text
any jumps
environment jumps
subsystem jumps
no jumps
```

Fidelity-drop groupings:

```text
environment_jump:
  environment_jump == true

without_environment_jumps:
  environment_jump == false

any_jump:
  any_jump == true

no_jump:
  no_jump == true
```

---

## 11. Bridge Metrics Aggregation

Create a small accumulator:

```go
type BridgeMetricAccumulator struct {
    trajectoryCount int

    totalJumpCount int
    totalEnvironmentJumpCount int
    totalSubsystemJumpCount int

    conditionalNormFailures int

    dropEnvJump SumCount
    dropNoEnvJump SumCount
    dropAnyJump SumCount
    dropNoJump SumCount

    empiricalL1Values []float64
}
```

Helper:

```go
type SumCount struct {
    Sum   float64
    Count int
}
```

Finalization:

```go
func (a *BridgeMetricAccumulator) Finalize() BridgeMetrics
```

Ratio:

```text
conditional_update_ratio =
  mean_fidelity_drop_at_environment_jumps /
  mean_fidelity_drop_without_environment_jumps
```

If denominator is unavailable or too close to zero:

```text
conditional_update_ratio = null
```

Recommended denominator floor:

```go
const RatioDenominatorFloor = 1e-14
```

---

## 12. Empirical Trajectory Equivariance

At selected sample steps:

```text
1. Count trajectory configurations Q(t).
2. Convert counts to empirical probabilities.
3. Compare empirical distribution with |psi(t)|².
4. Compute L1 distance.
```

Function:

```go
func EmpiricalEquivarianceL1(
    trajectoryStates []TrajectoryState,
    psi []complex128,
    dim int,
) float64
```

This diagnostic is finite-sample noisy. It should not use the strict master-equation equivariance tolerance from `BELL-MIPT-0001`.

Suggested behavior:

```text
Report max_empirical_equivariance_l1.
Report final_empirical_equivariance_l1.
Use it descriptively.
Do not fail the bridge solely because finite trajectory count is noisy.
```

Bridge can be inconclusive if:

```text
empirical L1 is extremely large and trajectory count is too small to distinguish sampling noise from implementation error.
```

But v0 should avoid a strong statistical threshold unless a null model is added later.

Recommended status policy:

```text
Empirical trajectory equivariance is descriptive in 0002A.
A rigorous finite-sample statistical threshold belongs in BELL-MIPT-0002B or BELL-MIPT-0003.
```

---

## 13. Bridge Status Logic

Keep existing top-level `goal_status`.

It continues to mean:

```text
Did BELL-MIPT-0001 finite Bell-rate/master-equation equivariance audit pass?
```

Add separate bridge status:

```text
bridge_disabled
bridge_toy_passed
bridge_toy_failed
bridge_toy_inconclusive
```

Rules:

### bridge_disabled

```text
bridge omitted or bridge.enabled=false.
```

### bridge_toy_passed

Only if:

```text
Trajectories sampled successfully.
Initial configurations sampled successfully.
No NaN/Inf in trajectory rates or conditional metrics.
Conditional states computed with acceptable norm failure count.
Fidelity drops computed when available.
Bridge metrics emitted.
Forbidden-language audit passes.
```

Do not require `conditional_update_ratio > 1`.

### bridge_toy_failed

Use for implementation-level failures:

```text
Invalid bridge config.
Invalid probability distribution.
Rates invalid.
Destination sampling invalid.
Conditional construction has deterministic indexing errors.
Report schema missing required bridge fields.
Forbidden-language audit fails.
```

### bridge_toy_inconclusive

Use when the run completes but does not support a stable diagnostic:

```text
Too few environment jumps to compare.
Too many conditional norm failures.
Numerical instability appears.
Empirical trajectory equivariance is too noisy to interpret.
Fidelity denominator unavailable.
```

Suggested thresholds for v0:

```text
If environment_jump_transitions == 0:
  bridge_toy_inconclusive.

If conditional_norm_failures > 0:
  bridge_toy_inconclusive unless explicitly tiny and explained.

If no valid fidelity comparisons exist:
  bridge_toy_inconclusive.
```

Keep the thresholds conservative and visible in the report.

---

## 14. Monitoring-Like Signal Interpretation

Add descriptive interpretation only:

```go
type BridgeInterpretation struct {
    MonitoringLikeSignal string `json:"monitoring_like_signal"`
    Reason               string `json:"reason"`
}
```

Allowed values:

```text
not_assessed
no_clear_signal
weak_signal
candidate_signal
```

Rules:

```text
bridge disabled:
  not_assessed

bridge inconclusive:
  not_assessed

bridge passed but ratio unavailable:
  no_clear_signal

bridge passed and ratio near 1:
  no_clear_signal

bridge passed and ratio modestly above 1:
  weak_signal

bridge passed and ratio clearly above 1 with enough environment jumps:
  candidate_signal
```

Suggested heuristic:

```text
candidate_signal only if:
  environment_jump_transitions >= 10
  non_environment_jump_transitions >= 10
  conditional_update_ratio >= 2.0
```

But the report must say:

```text
This is descriptive only and does not establish a Bell-MIPT bridge.
```

---

## 15. Report Schema Changes

Existing top-level fields must remain backward compatible:

```json
{
  "schema_version": "bell_mipt_report_v0",
  "toy_id": "BELL-MIPT-0001",
  "toy_analysis_only": true,
  "physics_claim": "none",
  "goal_status": "toy_goal_passed"
}
```

For this ticket, there are two acceptable schema choices:

### Option A — keep same report schema version

```text
schema_version: bell_mipt_report_v0
```

and add optional bridge section.

### Option B — bump report schema version

```text
schema_version: bell_mipt_report_v0_2a
```

Recommendation:

```text
Use schema_version: bell_mipt_report_v0_2a
```

Reason:

```text
The report gains new bridge fields. A schema version bump makes downstream audit easier.
```

However, preserve existing fields exactly.

Add:

```go
type BridgeReport struct {
    Enabled          bool                 `json:"enabled"`
    BridgeGoal      string               `json:"bridge_goal"`
    BridgeStatus    string               `json:"bridge_status"`
    Trajectories    int                  `json:"trajectories"`
    SubsystemSites  []int                `json:"subsystem_sites"`
    EnvironmentSites []int               `json:"environment_sites"`
    SampleEverySteps int                 `json:"sample_every_steps"`
    Metrics         BridgeMetrics        `json:"metrics"`
    Interpretation  BridgeInterpretation `json:"interpretation"`
    Warnings        []string             `json:"warnings,omitempty"`
}
```

Top-level report:

```go
type Report struct {
    SchemaVersion string `json:"schema_version"`
    ToyID string `json:"toy_id"`
    ToyAnalysisOnly bool `json:"toy_analysis_only"`
    PhysicsClaim string `json:"physics_claim"`

    Model string `json:"model"`
    Sites int `json:"sites"`
    HilbertDim int `json:"hilbert_dim"`

    Goal string `json:"goal"`
    GoalStatus string `json:"goal_status"`

    Checks Checks `json:"checks"`
    Metrics Metrics `json:"metrics"`

    Bridge BridgeReport `json:"bridge"`

    DebtStatus map[string]string `json:"debt_status"`
    Limitations []string `json:"limitations"`
    ForbiddenLanguageAudit ForbiddenLanguageAudit `json:"forbidden_language_audit"`
}
```

When bridge disabled:

```json
{
  "bridge": {
    "enabled": false,
    "bridge_goal": "sample_bell_trajectories_and_audit_conditional_subsystem_state",
    "bridge_status": "bridge_disabled",
    "trajectories": 0,
    "subsystem_sites": [],
    "environment_sites": [],
    "sample_every_steps": 0,
    "metrics": {
      "trajectory_count": 0
    },
    "interpretation": {
      "monitoring_like_signal": "not_assessed",
      "reason": "Bridge audit disabled by config."
    }
  }
}
```

When bridge enabled:

```json
{
  "bridge": {
    "enabled": true,
    "bridge_goal": "sample_bell_trajectories_and_audit_conditional_subsystem_state",
    "bridge_status": "bridge_toy_passed",
    "trajectories": 200,
    "subsystem_sites": [0, 1, 2],
    "environment_sites": [3, 4, 5],
    "sample_every_steps": 1,
    "metrics": {
      "trajectory_count": 200,
      "mean_jump_count": 0.0,
      "mean_environment_jump_count": 0.0,
      "mean_subsystem_jump_count": 0.0,
      "conditional_norm_failures": 0,
      "mean_fidelity_drop_at_environment_jumps": 0.0,
      "mean_fidelity_drop_without_environment_jumps": 0.0,
      "mean_fidelity_drop_at_any_jumps": 0.0,
      "mean_fidelity_drop_no_jump": 0.0,
      "conditional_update_ratio": null,
      "max_empirical_equivariance_l1": 0.0,
      "final_empirical_equivariance_l1": 0.0,
      "initial_empirical_equivariance_l1": 0.0,
      "environment_jump_transitions": 0,
      "non_environment_jump_transitions": 0,
      "any_jump_transitions": 0,
      "no_jump_transitions": 0
    },
    "interpretation": {
      "monitoring_like_signal": "not_assessed",
      "reason": "Insufficient environment jumps for a stable comparison."
    }
  }
}
```

---

## 16. Markdown Report Changes

Extend `report.md` with a bridge section.

Required sections:

```markdown
## Bridge Audit

Bridge enabled: true  
Bridge status: bridge_toy_passed

Bridge goal:

sample_bell_trajectories_and_audit_conditional_subsystem_state

### Bridge Metrics

| Metric | Value |
|---|---:|
| trajectory_count | ... |
| mean_jump_count | ... |
| mean_environment_jump_count | ... |
| mean_subsystem_jump_count | ... |
| conditional_norm_failures | ... |
| mean_fidelity_drop_at_environment_jumps | ... |
| mean_fidelity_drop_without_environment_jumps | ... |
| conditional_update_ratio | ... |
| max_empirical_equivariance_l1 | ... |
| final_empirical_equivariance_l1 | ... |

### Bridge Interpretation

Monitoring-like signal: not_assessed

Reason:

This is descriptive only. It does not show Bell jumps are measurements and does not establish a Bell-MIPT bridge.
```

Required limitations:

```text
This samples Bell configuration histories in a finite toy model only.
This conditional-state audit is not a monitored quantum trajectory simulation.
This does not implement MIPT.
This does not show Bell jumps are measurements.
This does not establish a Bell-MIPT bridge.
This does not support any holography or black-hole claim.
This is not a physics promotion.
```

---

## 17. Debt Status Update

When bridge is disabled, retain old debt status:

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

When bridge is enabled and runs successfully:

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

If bridge enabled but inconclusive:

```json
{
  "needMap": "attempted_conditional_state_toy_inconclusive",
  "needInvariant": "partially_paid_equivariance_only_empirical_trajectory_inconclusive",
  "needToyCheck": "partially_paid_rate_algebra_conditional_state_inconclusive",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_reviewed_for_0001_only"
}
```

If bridge enabled but failed:

```json
{
  "needMap": "unpaid_bridge_toy_failed",
  "needInvariant": "partially_paid_equivariance_only",
  "needToyCheck": "partially_paid_rate_algebra_only_bridge_toy_failed",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_reviewed_for_0001_only"
}
```

---

## 18. Forbidden Language Audit Updates

Extend forbidden phrase scan to catch bridge overclaims.

Add forbidden phrases:

```text
Bell jumps are measurements
Bell jumps equal measurements
Bell-MIPT bridge established
MIPT observed
holography explained
Bohmian mechanics validated
proves MIPT
proves holography
physics promotion
validated bridge
measurement-induced transition found
```

Allowed phrases:

```text
monitoring-like signal
candidate_signal
weak_signal
conditional-state audit
finite toy model
descriptive only
not_assessed
no_clear_signal
```

The report may use `candidate_signal`, but only with explicit non-promotion language.

---

## 19. Tests

### 19.1 Config Tests

```text
TestBridgeOmittedDefaultsToDisabled
TestBridgeDisabledPreservesOldBehavior
TestBridgePartitionValidationRejectsOverlap
TestBridgePartitionValidationRejectsMissingSite
TestBridgePartitionValidationRejectsOutOfRangeSite
TestBridgePartitionValidationAcceptsValidSplit
TestBridgeRequiresPositiveTrajectoryCount
TestBridgeRequiresPositiveSampleEverySteps
```

### 19.2 Trajectory Sampling Tests

```text
TestSampleDiscreteApproximatelyMatchesBornDistribution
TestSampleDiscreteDeterministicForFixedSeed
TestJumpProbabilityUsesTotalOutgoingRate
TestDestinationSamplingProportionalToOutgoingRates
TestNoJumpWhenTotalOutgoingRateZero
TestStepBellConfigurationRejectsInvalidRates
TestDeterministicTrajectoryOutputForFixedSeed
```

Use simple artificial rate matrices where the expected behavior is easy to verify.

### 19.3 Conditional State Tests

```text
TestSplitConfigSubsystemEnvironment
TestCombineConfigRoundTrip
TestConditionalStateDimension
TestConditionalStateNormalization
TestConditionalNormFailureRecorded
TestConditionalStateUsesActualEnvironmentConfiguration
TestConditionalFidelityIdenticalStatesIsOne
TestConditionalFidelityOrthogonalStatesIsZero
TestFidelityDropInUnitInterval
```

### 19.4 Bridge Metrics Tests

```text
TestEnvironmentJumpsCountedCorrectly
TestSubsystemJumpsCountedCorrectly
TestAnyJumpAndNoJumpClassifiedCorrectly
TestFidelityDropsClassifiedByEnvironmentJump
TestConditionalUpdateRatioHandlesZeroDenominator
TestEmpiricalTrajectoryEquivarianceL1Computed
TestBridgeStatusPassedWhenMetricsValid
TestBridgeStatusInconclusiveWhenNoEnvironmentJumps
TestBridgeStatusInconclusiveWhenConditionalNormFailuresTooHigh
```

### 19.5 Report Tests

```text
TestOldReportFieldsRemainBackwardCompatible
TestBridgeDisabledAppearsWhenBridgeOmitted
TestBridgeMetricsAppearWhenBridgeEnabled
TestBridgeLimitationsIncludeRequiredNonClaims
TestForbiddenLanguageAuditPassesForBridgeReport
TestForbiddenLanguageAuditCatchesBridgeOverclaims
TestDebtStatusUpdatedWhenBridgeEnabledPassed
TestDebtStatusRetainedWhenBridgeDisabled
```

### 19.6 Regression Tests

```text
TestBellMIPT0001DefaultRunUnchangedWithBridgeDisabled
TestFixedBridgeConfigProducesDeterministicReport
```

The deterministic report test should be careful about timestamps. Prefer no timestamps, or normalize them out before comparison.

---

## 20. Validation Commands

Required:

```bash
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge
```

Additional useful checks:

```bash
cat /tmp/bellmipt-default/report.json
cat /tmp/bellmipt-bridge/report.json
cat /tmp/bellmipt-bridge/report.md
```

Manual acceptance checks:

```text
Default run has bridge_status: bridge_disabled.
Default run preserves BELL-MIPT-0001 behavior.
Bridge run emits bridge metrics.
Bridge run does not emit MIPT/holography/measurement/promotion claims.
Forbidden-language audit passes.
Debt status is updated only when bridge is enabled.
```

---

## 21. Expected Final Implementation Report

The coding agent should report:

```text
1. Files added and modified.
2. Whether BELL-MIPT-0001 behavior remained unchanged when bridge was disabled.
3. Bridge-enabled run status.
4. Key bridge metrics.
5. Test commands run.
6. Known limitations.
7. EBP debt update.
8. Explicit non-claim statement.
```

Required final non-claim statement:

```text
No MIPT claim, no holography claim, no Bell-jumps-equal-measurements claim, no physics promotion.
```

---

## 22. Known Risks

### Risk 1: Finite trajectory noise

Empirical trajectory equivariance will be noisy for small trajectory counts.

Mitigation:

```text
Treat empirical trajectory equivariance as descriptive in 0002A.
Do not use strict BELL-MIPT-0001 master-equation tolerance.
Leave statistical null model for a later ticket.
```

### Risk 2: Too few environment jumps

Depending on parameters and time horizon, there may be few or no environment jumps.

Mitigation:

```text
Mark bridge_toy_inconclusive if there are too few environment jumps.
Do not manufacture a signal from insufficient data.
```

### Risk 3: Conditional norm failures

For some actual environment configurations, the conditional slice may have tiny norm.

Mitigation:

```text
Record conditional_norm_failures.
Skip fidelity for failed conditional states.
Mark bridge inconclusive if failures are nontrivial.
```

### Risk 4: Confusing Bell jumps with measurements

The conditional-state update may look monitoring-like, but this does not mean Bell jumps are measurements.

Mitigation:

```text
Report limitation explicitly.
Keep needObstruction: bell_jumps_are_not_measurements.
Use descriptive monitoring_like_signal only.
```

### Risk 5: Altering accepted 0001 behavior

Integrating trajectory sampling into the existing loop could accidentally change the accepted master-equation audit.

Mitigation:

```text
Prefer a two-pass bridge implementation using stored psi snapshots.
Add regression test that bridge-disabled default output remains stable.
```

### Risk 6: Dense matrix scalability

The existing finite toy uses dense Hilbert-space arrays.

Mitigation:

```text
Keep small-site toy scope.
Do not optimize prematurely.
Add guardrails for sites/steps/trajectories if memory would explode.
```

---

## 23. Non-Goals

Do not implement:

```text
MIPT phase diagram
area-law / volume-law scaling
entanglement entropy
mutual information
purity
monitored quantum circuits
projective measurements
Lindblad dynamics
holography
black holes
Lean theorem proving
AI agents
multi-flavor Majorana chains
physics promotion
random trajectory visualization
continuous-time Gillespie sampler
```

Do not claim:

```text
Bell jumps are measurements.
Bell-MIPT bridge is established.
MIPT was observed.
Holography is explained.
Bohmian mechanics is validated.
```

---

## 24. EBP 2.1 Claim Ledger

### Claim 1

`BELL-MIPT-0002A` samples Bell configuration trajectories from Bell positive-current jump rates.

Status:

```text
implementation_target
```

Debt:

```text
needToyCheck: partially payable by trajectory sampler tests
needInvariant: only descriptively supported by empirical trajectory equivariance
needFaithfulnessReview: unpaid for new bridge logic until reviewed
```

### Claim 2

The actual environment configuration induces a conditional subsystem wave function.

Status:

```text
toy_construction_target
```

Debt:

```text
needMap: partially payable as conditional_state_toy_only
needObstruction: bell_jumps_are_not_measurements remains active
needNullModel: unpaid
```

### Claim 3

Environment jumps may correlate with larger conditional-state fidelity drops.

Status:

```text
descriptive_metric_only
```

Debt:

```text
needNullModel: unpaid
needInvariant: unpaid beyond descriptive trajectory diagnostics
needFaithfulnessReview: unpaid for interpretation
```

### Claim 4

A monitoring-like signal, if observed, supports further investigation.

Status:

```text
not_promoted
```

Debt:

```text
needMap: not fully paid
needNullModel: unpaid
needObstruction: active
needFaithfulnessReview: unpaid
```

Boundary:

```text
A monitoring-like conditional-state diagnostic is not a measurement claim and not a Bell-MIPT bridge claim.
```

---

## 25. Recommendation

Implement `BELL-MIPT-0002A` as a direct extension of the current `cmd/bellmipt` command, but isolate the new logic in separate files under the existing `internal/bellmipt` package.

Recommended final structure:

```text
cmd/bellmipt/main.go
  remains thin

internal/bellmipt/run.go
  coordinates existing 0001 audit and optional 0002A bridge audit

internal/bellmipt/bridge_config.go
  validates bridge config and site partition

internal/bellmipt/trajectory.go
  samples initial configurations and Bell jumps

internal/bellmipt/conditional.go
  splits configurations and computes conditional subsystem states

internal/bellmipt/bridge_metrics.go
  accumulates fidelity drops, jump counts, and empirical equivariance

internal/bellmipt/bridge_report.go
  adds bridge report sections and debt updates
```

Do not create a separate command.

Do not create a separate top-level package.

Reason:

```text
BELL-MIPT-0002A depends directly on the existing Hamiltonian, Bell current, Bell rates, wavefunction evolution, report, and forbidden-language audit. Keeping it under internal/bellmipt maximizes reuse while preserving one-command simplicity.
```

Final implementation principle:

```text
One command.
Optional bridge section.
Old behavior preserved when disabled.
Trajectory and conditional-state audit enabled only by config.
No promotion language.
No MIPT claim.
No measurement claim.
```

