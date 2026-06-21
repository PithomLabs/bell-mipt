Yes. The new files add **some net-new value**, but they do **not replace Claude as the base**. The best final plan is still:

```text id="jng4gh"
Base: Claude
Merge: Xiaomi’s max_lambda_dt / scaling / implementation-sequence details
Merge: Z’s rate-cache performance idea
Merge: Gemini’s simple status-state wording and report limitation clarity
Reject/repair: hard empirical-L1 thresholds, ratio=0 fallback, raw forbidden-word bans, math/rand/v2 requirement
```

## Net-new value by file

### `plan_gemini.md`

Gemini adds a clean, compact version of the bridge algorithm: discrete-time thinning, conditional wave-function extraction, fidelity drop, environment/subsystem classification, empirical equivariance, bridge status, and required limitations. It is useful as a **simple coding checklist**. 

Net-new worth keeping:

```text id="o7m0vk"
- clear bridge_toy_failed / bridge_toy_inconclusive / bridge_toy_passed state machine
- explicit threshold idea for conditional norm failures
- explicit required report limitations
- simple direct-extension architecture
```

Repair before merging:

```text id="lwzg9w"
- Do not set conditional_update_ratio to 0 when denominator is unavailable; use null.
- Do not use MaxEmpiricalEquivarianceL1 > 0.15 as a hard status gate in 0002A.
- Do not ban raw words like MIPT/holography/measurement globally; allow negated limitation text.
```

### `plan_xiaomi(1).md`

Xiaomi adds the strongest practical engineering details among the new three. It emphasizes preserving `BELL-MIPT-0001` behavior, keeping core files unchanged where possible, storing `ψ(t)` history, and adds a valuable runtime diagnostic: **track `max_lambda_dt`** to detect when discrete-time thinning may be missing multi-jump events. 

Net-new worth keeping:

```text id="e0n4vs"
- add max_lambda_dt to bridge metrics
- warn or mark inconclusive if lambda*dt becomes too large
- document psi-history memory scaling
- phase-by-phase implementation sequence
- explicit deterministic output tests
- concrete file inventory / estimated scope
```

Repair before merging:

```text id="i8bktn"
- “byte-for-byte” old report compatibility may conflict with adding bridge_disabled section.
- Prefer stable report schema over byte-for-byte identical output.
- Bridge-disabled behavior should preserve 0001 metrics and meaning, not necessarily identical JSON if schema version changes.
```

### `plan_z.md`

Z adds one important performance improvement: **precompute outgoing rates for all states at each time step once, then reuse them across trajectories**. That avoids recomputing a full Bell current/rates matrix separately per trajectory. 

Net-new worth keeping:

```text id="s8wz6j"
- optional per-time-step rate cache
- guard lambda < tiny floor before destination sampling
- explicit no parallelism in v0 for determinism
- compact bridge-run orchestration
```

Repair before merging:

```text id="x94z3p"
- Do not require math/rand/v2.
- Keep math/rand unless go.mod already mandates Go 1.22+ and repo prefers v2.
- Ratio < 0.8 should not automatically be called weak_signal; that may simply mean environment jumps are less disruptive.
- Signal thresholds must remain descriptive only.
```

## Final synthesis

The cohesive final plan should be:

```text id="3uj0ol"
Claude base + Xiaomi diagnostics + Z rate-cache option + Gemini status clarity.
```

Main final decisions:

| Design question        | Final decision                                      |
| ---------------------- | --------------------------------------------------- |
| Base plan              | Claude                                              |
| Package layout         | Direct extension inside `internal/bellmipt`         |
| CLI                    | No new subcommands                                  |
| Bridge config          | Optional `bridge` section                           |
| Default behavior       | Bridge disabled                                     |
| Wavefunction evolution | Preserve 0001 path, store `ψ(t)` snapshots          |
| Trajectory sampling    | Discrete-time thinning                              |
| Rate computation       | Prefer per-time-step rate cache if simple           |
| Conditional state      | Use projector/grouping by environment config        |
| Ratio unavailable      | JSON `null`, not `0`                                |
| Empirical equivariance | Descriptive only in 0002A                           |
| `max_lambda_dt`        | Add as bridge metric/warning                        |
| RNG                    | Use deterministic per-trajectory `math/rand`        |
| Signal classification  | Descriptive only, never status/promotion            |
| Forbidden audit        | Scan full assembled report including bridge section |

## Cohesive merged plan

# BELL-MIPT-0002A Cohesive Implementation Plan

## Status

Plan only. No physics promotion.

This plan extends the accepted `BELL-MIPT-0001` Go toy. It should preserve the existing finite Bell-rate/master-equation equivariance audit and add an optional trajectory-level bridge audit behind a `bridge.enabled` config flag.

## Goal

Implement:

```text
BELL-MIPT-0002A:
Bell trajectory sampler + conditional subsystem state audit
```

The goal is to sample actual Bell configuration histories, split each actual configuration into subsystem and environment parts, construct the conditional subsystem wave function induced by the actual environment configuration, and measure whether environment jumps produce larger conditional-state changes than non-environment-jump evolution.

Allowed conclusion:

```text
Bell trajectories were sampled, empirical trajectory equivariance was checked descriptively, and conditional subsystem-state changes were measured for the tested finite toy configuration.
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

## Base plan

Use Claude’s plan as the base.

Merge in:

```text
1. Xiaomi’s max_lambda_dt diagnostic and implementation sequence.
2. Z’s optional per-time-step rate-cache optimization.
3. Gemini’s compact bridge status wording and explicit limitation list.
```

Reject or repair:

```text
1. Do not set unavailable ratios to 0; use null.
2. Do not make empirical trajectory L1 a hard pass/fail threshold in 0002A.
3. Do not ban raw words like MIPT, holography, measurement, or monitored globally.
4. Do not require math/rand/v2 unless the repo already mandates it.
5. Do not make conditional_update_ratio decide bridge_status.
```

## Pre-implementation source assumptions to verify

Before coding, verify the current `BELL-MIPT-0001` source:

```text
A1. Basis states are full occupation configurations, likely uint64 bitstrings.
A2. There is an existing way to compute Bell current and Bell rates from ψ(t).
A3. ψ(t) evolves deterministically and independently of the actual Bell configuration.
A4. The existing run loop can be minimally refactored to retain ψ snapshots.
A5. The existing report/forbidden-language audit can scan the fully assembled bridge report.
```

If any assumption is false, add small accessors or refactors. Do not rewrite the accepted 0001 machinery.

## File layout

Modify:

```text
internal/bellmipt/config.go
internal/bellmipt/run.go
internal/bellmipt/report.go
internal/bellmipt/markdown.go
internal/bellmipt/forbidden.go
```

Add:

```text
internal/bellmipt/bridge_config.go
internal/bellmipt/site_split.go
internal/bellmipt/trajectory.go
internal/bellmipt/conditional.go
internal/bellmipt/bridgemetrics.go
internal/bellmipt/equivariance_empirical.go
internal/bellmipt/bridge_report.go
```

Tests:

```text
internal/bellmipt/bridge_config_test.go
internal/bellmipt/site_split_test.go
internal/bellmipt/trajectory_test.go
internal/bellmipt/conditional_test.go
internal/bellmipt/bridgemetrics_test.go
internal/bellmipt/bridge_report_test.go
internal/bellmipt/bridge_integration_test.go
```

Add example config:

```text
bellmipt_bridge.json
```

Do not add:

```text
new binary
new subcommands
new external packages
web UI
database
agent runtime
MIPT package
```

## Config extension

Add optional bridge config:

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

Add to existing config:

```go
Bridge *BridgeConfig `json:"bridge,omitempty"`
```

Validation:

```text
If bridge omitted: valid, disabled.
If bridge.enabled=false: valid, disabled.
If bridge.enabled=true:
  subsystem_sites non-empty
  environment_sites non-empty
  no duplicates
  no overlaps
  no negative sites
  no site >= sites
  subsystem_sites ∪ environment_sites = {0,...,sites-1}
  trajectories > 0
  sample_every_steps >= 1
```

Sort subsystem and environment site lists ascending internally. Report the normalized order.

## Command behavior

Keep:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

When `bridge` is omitted or disabled:

```text
Run accepted BELL-MIPT-0001 path.
Do not sample trajectories.
Do not change goal_status semantics.
```

When `bridge.enabled=true`:

```text
Run BELL-MIPT-0001 base audit first.
Store ψ(t) snapshots.
Run bridge trajectory/conditional audit.
Add bridge section to report.
```

## Core architecture

Use two-pass mode.

Pass 1:

```text
Run existing master-equation/equivariance audit.
Store ψ snapshots at time steps k=0..steps.
Keep ρ master-equation audit unchanged.
```

Pass 2:

```text
Use ψ snapshots to sample actual Bell configuration trajectories.
Compute conditional subsystem states from Q_B(t).
Compute fidelity-drop metrics.
Compute empirical trajectory equivariance diagnostics.
```

This protects the accepted 0001 result.

## Optional rate-cache optimization

For each time step `k`, compute rates once from `ψ[k]`:

```text
J_k = BellCurrent(H, ψ[k])
Rates_k = BellRates(J_k, ψ[k])
```

Then reuse `Rates_k` for all trajectories at that same time step.

This is optional but preferred if easy because it avoids recomputing the full rates matrix once per trajectory.

Do not change Bell-rate math.

## Site splitting

Define:

```go
type SiteSplit struct {
    SubsystemSites   []int
    EnvironmentSites []int
    SubsystemDim     int
    EnvironmentDim   int
}
```

Functions:

```go
func NewSiteSplit(subsystem, environment []int, sites int) (SiteSplit, error)
func SplitConfig(q uint64, split SiteSplit) (subQ uint64, envQ uint64)
func CombineConfig(subQ uint64, envQ uint64, split SiteSplit) uint64
```

Required invariant:

```text
CombineConfig(SplitConfig(q)) == q
```

Subsystem compact index:

```text
subQ bit k = occupation of full site SubsystemSites[k]
```

Environment compact index:

```text
envQ bit k = occupation of full site EnvironmentSites[k]
```

## Conditional projector

Precompute environment groupings:

```go
type ConditionalProjector struct {
    Split SiteSplit
    ByEnvironment [][]QAPair
}

type QAPair struct {
    FullIndex int
    SubIndex  int
}
```

Build once:

```go
func BuildConditionalProjector(split SiteSplit, basis Basis) ConditionalProjector
```

For each full basis configuration, compute `(subQ, envQ)` and append `(fullIndex, subIndex)` to `ByEnvironment[envQ]`.

## Conditional subsystem state

Given `ψ(t)` and actual environment configuration `envQ`:

```text
ψ_A(a,t) = Ψ(a, envQ, t)
```

Implement:

```go
const ConditionalNormFloor = 1e-14

func ConditionalSubsystemState(
    psi []complex128,
    envQ uint64,
    projector ConditionalProjector,
) (ConditionalState, error)
```

State:

```go
type ConditionalState struct {
    EnvConfig  uint64
    Vector     []complex128
    Norm       float64
    Normalized bool
}
```

Behavior:

```text
If norm < ConditionalNormFloor:
  return Normalized=false
  record conditional_norm_failure
  skip fidelity for this state
Else:
  normalize vector
```

Do not panic on norm failure.

## Fidelity

Implement:

```go
func ConditionalFidelity(a, b []complex128) (float64, error)
```

Formula:

```text
F = |Σ_i conj(a_i) b_i|²
drop = 1 - F
```

Numerical behavior:

```text
Clamp tiny floating drift to [0,1] if within 1e-12.
If outside tolerance, mark bridge inconclusive.
```

Unavailable fidelity due to invalid conditional states must be skipped, not converted to 0 or 1.

## Bell trajectory sampling

Initial configuration:

```text
Sample Q0 from |ψ0(q)|².
```

Function:

```go
func SampleDiscrete(prob []float64, rng *rand.Rand) (int, error)
```

Per-step update:

```go
func StepBellConfiguration(
    q uint64,
    rates [][]float64,
    dt float64,
    rng *rand.Rand,
) (nextQ uint64, jumped bool, lambda float64, err error)
```

Algorithm:

```text
lambda = Σ_n rates[n][q]
if lambda <= tiny floor: no jump
p_jump = 1 - exp(-lambda * dt)
if p_jump is NaN/Inf: bridge inconclusive
if p_jump > 1 due to pathology: bridge inconclusive
sample jump/no-jump
if jump:
  sample destination n proportional to rates[n][q]
```

Use discrete-time thinning only. Do not implement Gillespie in 0002A.

## RNG policy

Use deterministic per-trajectory RNG.

Preferred standard-library policy:

```go
seed_i := cfg.Bridge.Seed + int64(i)*0x9e3779b97f4a7c15
rng := rand.New(rand.NewSource(seed_i))
```

If the multiplication overflows, deterministic signed overflow is fine in Go for integer arithmetic? Actually Go integer overflow wraps only for unsigned; for signed it is defined for operations? Avoid relying on signed overflow. Use a simple helper based on uint64 mixing and cast back to int64.

Do not use global RNG.

Run trajectories sequentially in v0.

## Jump classification

For each transition:

```text
previousQ -> currentQ
```

Compute:

```text
prevSubQ, prevEnvQ = SplitConfig(previousQ)
nextSubQ, nextEnvQ = SplitConfig(currentQ)
```

Classify:

```text
environment_jump = prevEnvQ != nextEnvQ
subsystem_jump   = prevSubQ != nextSubQ
any_jump         = previousQ != currentQ
no_jump          = previousQ == currentQ
```

Environment and subsystem jumps are not mutually exclusive. Boundary-crossing jumps may change both.

Fidelity buckets:

```text
at_environment_jumps: environment_jump == true
without_environment_jumps: environment_jump == false
at_any_jumps: any_jump == true
no_jump: no_jump == true
```

## Bridge metrics

Add:

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

    InitialEmpiricalEquivarianceL1 float64 `json:"initial_empirical_equivariance_l1"`
    MaxEmpiricalEquivarianceL1     float64 `json:"max_empirical_equivariance_l1"`
    FinalEmpiricalEquivarianceL1   float64 `json:"final_empirical_equivariance_l1"`

    MaxLambdaDT float64 `json:"max_lambda_dt"`

    EnvironmentJumpTransitions    int `json:"environment_jump_transitions"`
    NonEnvironmentJumpTransitions int `json:"non_environment_jump_transitions"`
    AnyJumpTransitions            int `json:"any_jump_transitions"`
    NoJumpTransitions             int `json:"no_jump_transitions"`
}
```

`MaxLambdaDT` comes from Xiaomi’s useful diagnostic:

```text
max over all trajectory micro-steps of lambda * dt
```

Interpretation:

```text
If max_lambda_dt > 0.1:
  add warning that discrete-time thinning may miss multi-jump events.
  bridge_status may be inconclusive if severe.
```

Suggested severity:

```text
0.0 <= max_lambda_dt <= 0.1:
  acceptable toy regime
0.1 < max_lambda_dt <= 0.5:
  warning only
max_lambda_dt > 0.5:
  bridge_toy_inconclusive
```

This is not a physics threshold. It is a numerical-sampling warning.

## Empirical trajectory equivariance

At each sampled time step:

```text
empirical(q,t) = count trajectories with Q(t)=q / trajectories
born(q,t) = |ψ_q(t)|²
L1(t) = Σ_q |empirical(q,t) - born(q,t)|
```

Report:

```text
initial_empirical_equivariance_l1
max_empirical_equivariance_l1
final_empirical_equivariance_l1
```

Do not use strict 0001 tolerance. Do not fail the bridge solely because empirical L1 is large unless NaN/Inf/pathological values appear.

## Bridge status

Top-level `goal_status` remains unchanged and still refers only to BELL-MIPT-0001.

Add:

```text
bridge_disabled
bridge_toy_passed
bridge_toy_failed
bridge_toy_inconclusive
```

Rules:

```text
bridge_disabled:
  bridge omitted or bridge.enabled=false

bridge_toy_failed:
  invalid bridge config
  structural indexing error
  invalid probability distribution
  invalid rates
  report schema missing required bridge fields
  forbidden-language audit fails

bridge_toy_inconclusive:
  no environment jump transitions
  no valid fidelity comparisons
  conditional norm failures exceed threshold
  NaN/Inf appears in bridge metrics
  max_lambda_dt > severe threshold
  empirical equivariance diagnostic is pathological

bridge_toy_passed:
  trajectories sampled successfully
  conditional states computed with acceptable norm failure count
  fidelity metrics emitted where available
  empirical trajectory equivariance diagnostic emitted
  max_lambda_dt within acceptable or warning-only range
```

Do not require `conditional_update_ratio > 1`.

## Monitoring-like signal

Add:

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
bridge disabled or inconclusive:
  not_assessed

ratio unavailable:
  not_assessed or no_clear_signal with reason

ratio < 1.5:
  no_clear_signal

1.5 <= ratio < 3.0:
  weak_signal

ratio >= 3.0 and enough environment/non-environment samples:
  candidate_signal
```

Every reason must include:

```text
This is descriptive only and does not establish a Bell-MIPT bridge.
```

## Report schema

Use stable bridge section.

Preferred:

```go
type Report struct {
    // existing fields unchanged
    Bridge BridgeReport `json:"bridge"`
}
```

When bridge is disabled, include:

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
      "trajectory_count": 0,
      "mean_fidelity_drop_at_environment_jumps": null,
      "mean_fidelity_drop_without_environment_jumps": null,
      "conditional_update_ratio": null
    },
    "interpretation": {
      "monitoring_like_signal": "not_assessed",
      "reason": "Bridge audit disabled by config."
    },
    "warnings": []
  }
}
```

Use:

```text
unavailable scalar means/ratios -> null
slices -> []
```

Avoid `null` for slices.

## Markdown report

Add section:

```markdown
## Bridge Audit

Bridge enabled: true
Bridge status: bridge_toy_passed

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
| initial_empirical_equivariance_l1 | ... |
| max_empirical_equivariance_l1 | ... |
| final_empirical_equivariance_l1 | ... |
| max_lambda_dt | ... |

### Bridge Interpretation

Monitoring-like signal: not_assessed

Reason:
This is descriptive only and does not establish a Bell-MIPT bridge.
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

## Forbidden-language audit

Scan the fully assembled report, including:

```text
bridge metrics
bridge interpretation reason
bridge limitations
debt status
markdown report
```

Forbid positive overclaims:

```text
Bell jumps are measurements
Bell jumps equal measurements
Bell-MIPT bridge established
MIPT observed
holography explained
Bohmian mechanics validated
proves MIPT
proves holography
validated bridge
measurement-induced transition found
```

Allow negated limitation phrases:

```text
This does not show Bell jumps are measurements.
This does not establish a Bell-MIPT bridge.
No MIPT claim.
No holography claim.
No physics promotion.
not a monitored quantum trajectory simulation
```

Do not ban raw words globally.

## Debt status

If bridge disabled:

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

If bridge passed:

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

If bridge inconclusive:

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

If bridge failed:

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

## Tests

Config:

```text
TestBridgeOmittedDefaultsToDisabled
TestBridgeDisabledPreservesOldBehavior
TestBridgePartitionRejectsOverlap
TestBridgePartitionRejectsMissingSite
TestBridgePartitionRejectsOutOfRangeSite
TestBridgePartitionAcceptsValidSplit
TestBridgeRequiresPositiveTrajectories
TestBridgeRequiresPositiveSampleEverySteps
```

Site split and conditional state:

```text
TestSplitConfigManualPartition
TestCombineConfigRoundTrip
TestConditionalProjectorGroupsByEnvironment
TestConditionalStateDimension
TestConditionalStateNormalization
TestConditionalNormFailure
TestConditionalFidelityIdentical
TestConditionalFidelityOrthogonal
TestFidelityDropInUnitInterval
```

Trajectory:

```text
TestSampleDiscreteMatchesDistribution
TestSampleDiscreteDeterministic
TestStepBellConfigurationUsesTotalRate
TestDestinationSamplingProportionalToRates
TestNoJumpWhenTotalRateZero
TestMaxLambdaDTRecorded
TestDeterministicTrajectoryForFixedSeed
```

Metrics/status:

```text
TestEnvironmentJumpClassification
TestSubsystemJumpClassification
TestBothSidesChangedClassification
TestNoJumpClassification
TestFidelityDropBuckets
TestConditionalUpdateRatioNullOnZeroDenominator
TestEmpiricalEquivarianceL1Computed
TestBridgeStatusPassedWhenClean
TestBridgeStatusInconclusiveWhenNoEnvironmentJumps
TestBridgeStatusInconclusiveWhenMaxLambdaDTSevere
TestBridgeStatusIgnoresConditionalUpdateRatio
```

Report:

```text
TestBridgeDisabledReportShape
TestBridgeEnabledMetricsAppear
TestRequiredBridgeLimitationsPresent
TestForbiddenAuditScansBridgeReason
TestForbiddenAuditCatchesBridgeOverclaim
TestDebtStatusPassedInconclusiveFailedDisabled
TestDefault0001MetricsUnchangedWithBridgeDisabled
```

Validation:

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge
```

## Implementation sequence

```text
1. Verify source assumptions from BELL-MIPT-0001.
2. Add BridgeConfig and validation.
3. Add SiteSplit and round-trip tests.
4. Add ConditionalProjector and conditional state tests.
5. Add fidelity tests.
6. Add trajectory sampling and deterministic RNG tests.
7. Add rate-cache or direct rate access.
8. Add bridge metrics accumulator.
9. Add empirical equivariance tracker.
10. Add bridge status and interpretation logic.
11. Add report schema and markdown section.
12. Extend forbidden-language audit to assembled bridge report.
13. Add bridge-enabled config fixture.
14. Run full validation.
```

## Final acceptance boundary

Even if all tests pass, the only allowed conclusion is:

```text
Bell trajectories were sampled, empirical trajectory equivariance was checked descriptively, and conditional subsystem-state changes were measured for the tested finite toy configuration.
```

Not allowed:

```text
Bell jumps behave like measurements.
Bell-MIPT bridge is established.
MIPT was observed.
Holography was explained.
Bohmian mechanics was validated.
```

## Final recommendation

Use this merged version as the new **final implementation-plan base**.

```json id="lmp3zj"
{
  "base": "plan_claude.md",
  "merged_net_new": [
    "plan_xiaomi max_lambda_dt diagnostic",
    "plan_xiaomi implementation phasing and memory-scaling notes",
    "plan_z optional per-time-step rate cache",
    "plan_gemini compact bridge status wording and limitation list"
  ],
  "rejected_or_repaired": [
    "hard empirical L1 threshold",
    "ratio unavailable encoded as zero",
    "raw forbidden-word bans",
    "math/rand/v2 requirement",
    "conditional_update_ratio controlling bridge_status"
  ],
  "promotion_status": "unpromoted_implementation_plan_only"
}
```

EBP status:

```json id="h13mns"
{
  "needMap": "ready_for_partial_conditional_state_toy_attempt",
  "needInvariant": "0001_partially_paid; 0002A_empirical_trajectory_diagnostic_planned",
  "needToyCheck": "0002A_ready_for_implementation_prompt",
  "needNullModel": "unpaid_later_ticket",
  "needObstruction": "bell_jumps_are_not_measurements_preserved",
  "needFaithfulnessReview": "0002A_pending_after_implementation",
  "promotion_status": "unpromoted_plan_synthesis_only"
}
```

