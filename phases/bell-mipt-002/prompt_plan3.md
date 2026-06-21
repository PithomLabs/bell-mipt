Yes — the plan is ready to give to a coding agent. I would add **one final instruction at the top**: implement tests first for the dangerous bugs before writing the full feature. The final adversarial reviews all converged on “ready with minor repairs,” and those repairs are now folded into the plan: strict boundary-jump isolation, `QIndex`/`QConfig` separation, near-zero probability handling, rate orientation tests, and no debt advancement unless `bridge_audit_completed`.    

Use this as the final coding-agent prompt.

# Final Coding Agent Prompt: Implement BELL-MIPT-0002A

You are implementing:

```text
BELL-MIPT-0002A:
Bell trajectory sampler + environment-projected conditional-vector audit
```

This extends the already accepted `BELL-MIPT-0001` Go toy.

## Current accepted baseline

`BELL-MIPT-0001` computes Bell probability currents and Bell positive-current jump rates for a finite Kitaev-chain-style fermionic lattice, evolves both:

```text
ψ(t)
ρ(t)
```

and verifies numerical equivariance:

```text
ρ(t) tracks |ψ(t)|²
```

The accepted conclusion of `0001` is only:

```text
The finite Bell-rate algebra and numerical equivariance audit passed for the tested toy configuration.
```

Do **not** disturb this baseline.

## Goal of 0002A

Add an optional bridge audit that:

```text
1. Samples actual Bell configuration trajectories Q(t).
2. Splits Q(t) into subsystem Q_A(t) and environment Q_B(t).
3. Constructs environment-projected conditional vectors:
     ψ_A(a,t | b) = Ψ(full_config(a,b), t)
   using b = Q_B(t).
4. Measures fidelity drops between consecutive conditional vectors.
5. Separates strict environment-only jumps from subsystem-only and boundary-crossing jumps.
6. Reports empirical trajectory equivariance as a descriptive diagnostic.
7. Reports max_lambda_dt as a discrete-time thinning reliability diagnostic.
```

Allowed conclusion if implementation succeeds:

```text
Bell trajectories were sampled, empirical trajectory equivariance was checked descriptively, and environment-projected conditional-vector changes were measured for the tested finite toy configuration.
```

Forbidden conclusions:

```text
No Bell-MIPT bridge claim.
No MIPT claim.
No measurement claim.
No Bell-jumps-equal-measurements claim.
No monitored-circuit claim.
No holography claim.
No black-hole claim.
No Bohmian-mechanics validation claim.
No physics promotion.
```

## Implementation rule

Use Go only.

Do not add external dependencies.

Do not add a new binary.

Do not add subcommands.

Keep the existing command shape:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

If the bridge config is omitted or disabled, the program must preserve `BELL-MIPT-0001` behavior and semantic metrics.

---

# 1. Architecture

Use a two-pass architecture.

## Pass 1: existing BELL-MIPT-0001 audit

Run the existing deterministic master-equation/equivariance audit as before.

It must still:

```text
build basis
build Hamiltonian
compute Bell currents/rates
evolve ψ and ρ
verify ρ(t) tracks |ψ(t)|²
write existing 0001 metrics
```

Store `ψ[k]` snapshots at full time-step boundaries.

Important:

```text
The bridge audit must use the same ψ snapshots produced by Pass 1.
The bridge audit must not rerun RK4 independently.
The bridge audit must not alter 0001 goal_status or 0001 metrics.
```

## Pass 2: optional bridge audit

Only if:

```json
"bridge": { "enabled": true }
```

then:

```text
sample Bell trajectories from stored ψ[k]
construct environment-projected conditional vectors
compute fidelity-drop metrics
compute empirical trajectory equivariance diagnostics
compute max_lambda_dt
emit bridge_audit_* status
```

If bridge is omitted or disabled:

```text
Do not emit bridge section.
Keep schema_version as bell_mipt_report_v0.
Preserve 0001 behavior and semantic metrics.
```

If bridge is enabled:

```text
schema_version = bell_mipt_report_v0_2a
emit bridge section
```

---

# 2. Files

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

Add tests:

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

---

# 3. Tests-first requirement

Before implementing the full feature, define tests for the dangerous bugs:

```text
TestTimeStepAlignmentUsesPsiKForIntervalK
TestRateOrientationDestinationSource
TestTotalOutgoingRateUsesSourceColumn
TestFakeRateTrajectoryKnownDestination
TestJumpFrequencyMatchesExpectedProbability
TestBridgeRatesMatch0001Rates

TestSplitCombineRoundTripAllConfigs
TestNonContiguousPartition
TestConditionalVectorDimension
TestConditionalProjectionPreservesPhase
TestConditionalProjectorGroupsByEnvironment

TestBoundaryCrossingJumpBothSidesChanged
TestBoundaryCrossingJumpExcludedFromPrimaryRatio
TestBoundaryCrossingJumpReportedSeparately

TestZeroRateNoJumpInconclusive
TestNearZeroCurrentConfigurationProbability
TestConditionalNormFailureThresholds
TestConditionalUpdateRatioNullWhenLowEvents
TestMaxLambdaDTWarningAndInconclusive

TestForbiddenAuditScansFullJSONAndMarkdown
TestBridgeDisabledPreserves0001SemanticMetrics

TestEmpiricalEquivarianceSampledStepsOnly
TestEmpiricalEquivarianceAgainstPsiSquared
```

Use deterministic fake RNG for correctness tests.

Use fixed-seed statistical tests only where distribution behavior is being tested.

Bridge-disabled regression must compare semantic fields, not byte-for-byte JSON.

---

# 4. Config extension

Add:

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

Extend config:

```go
Bridge *BridgeConfig `json:"bridge,omitempty"`
```

Validation:

```text
If bridge omitted or disabled:
  valid; skip bridge validation

If bridge enabled:
  subsystem_sites non-empty
  environment_sites non-empty
  no duplicate subsystem sites
  no duplicate environment sites
  no overlap
  no missing sites
  no negative sites
  no site >= sites
  trajectories > 0
  sample_every_steps >= 1
```

Use canonical sorted site lists for computation.

Report both requested and canonical lists:

```json
{
  "subsystem_sites_requested": [5, 0, 2],
  "environment_sites_requested": [1, 3, 4],
  "subsystem_sites_canonical": [0, 2, 5],
  "environment_sites_canonical": [1, 3, 4]
}
```

---

# 5. QIndex vs QConfig

The implementation must distinguish:

```go
QIndex  int    // index into basis slice / matrix column
QConfig uint64 // raw occupation bitmask, basis[QIndex]
```

Rules:

```text
Rate lookup uses QIndex.
Site splitting uses QConfig.
basis[QIndex] = QConfig.
rates[destIndex][srcIndex] = σ(dest <- src).
```

Never use a raw bitmask as a matrix index unless the code explicitly proves `basis[i] == uint64(i)` for that basis.

---

# 6. Time-step alignment

For each interval `k -> k+1`:

Given:

```text
ψ[k]       = universal wavefunction at t_k
QIndex[k] = actual configuration index at t_k
QConfig[k] = basis[QIndex[k]]
```

Do:

```text
1. Compute Bell rates from ψ[k].
2. Use rates[destIndex][QIndex[k]] to compute λ(Q[k], t_k).
3. Sample QIndex[k+1] using p_jump = 1 - exp(-λ dt).
4. ψ[k+1] is already available from Pass 1 RK4 snapshots.
5. Conditional vector at sample k uses (ψ[k], Q_B[k]).
6. Conditional vector at sample k+1 uses (ψ[k+1], Q_B[k+1]).
```

No jump sampling at RK4 sub-stages.

`sample_every_steps` refers to full time steps.

The first conditional vector is stored at `k=0`. Fidelity drops are computed from the first subsequent sampled transition onward.

---

# 7. Rate convention

Lock this convention:

```text
rates[dest][src] = σ(dest <- src)
```

For current source `q`:

```text
lambda(q) = Σ_dest rates[dest][q]
```

The trajectory sampler must reuse the exact Bell current/rate definitions from `BELL-MIPT-0001`.

Required tests:

```text
TestRateOrientationDestinationSource
TestTotalOutgoingRateUsesSourceColumn
TestFakeRateTrajectoryKnownDestination
TestBridgeRatesMatch0001Rates
```

---

# 8. Near-zero probability handling

Bell rates divide by `|ψ_q|²`.

Add:

```go
const RateProbabilityFloor = 1e-14
```

Prefer reusing the existing `BELL-MIPT-0001` probability floor if already defined.

If current actual configuration has:

```text
|ψ[QIndex]|² < RateProbabilityFloor
```

then:

```text
set outgoing λ = 0
do not sample a jump
keep QIndex unchanged
emit near_zero_current_configuration_probability warning
increment near_zero_current_configuration_probability_count
```

If this occurs frequently, the bridge audit should become inconclusive.

Suggested status rule:

```text
if near_zero_current_configuration_probability_count > 0:
  warning

if near_zero_current_configuration_probability_count / total_micro_steps > 0.01:
  bridge_audit_inconclusive
```

---

# 9. Discrete-time thinning

Use:

```text
p_jump = 1 - exp(-lambda * dt)
```

At most one jump per `dt`.

Track:

```text
max_lambda_dt = max(lambda * dt)
```

Thresholds:

```text
LambdaDTWarning = 0.05
LambdaDTInconclusive = 0.20
```

Rules:

```text
max_lambda_dt <= 0.05:
  acceptable toy regime

0.05 < max_lambda_dt <= 0.20:
  bridge_audit_completed with warning

max_lambda_dt > 0.20:
  bridge_audit_inconclusive
```

This is a numerical reliability diagnostic, not a physics criterion.

---

# 10. Site split

Add:

```go
type SiteSplit struct {
    SubsystemSitesRequested   []int
    EnvironmentSitesRequested []int
    SubsystemSitesCanonical   []int
    EnvironmentSitesCanonical []int
    SubsystemDim              int
    EnvironmentDim            int
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

for every full basis configuration `q`.

Must support non-contiguous partitions:

```text
A = [0,2,5]
B = [1,3,4]
```

---

# 11. Jordan-Wigner / fermion sign rule

The bridge audit constructs an environment-projected conditional vector in the canonical occupation basis:

```text
ψ_A(a,t | b) = Ψ(full_config(a,b), t)
```

No additional Jordan-Wigner sign is introduced during projection.

All phases and signs are inherited from the full wavefunction amplitude:

```text
Ψ(q,t)
```

Reason:

```text
Projection is a passive read of amplitudes from the already-evolved full occupation-basis vector.
No fermionic creation/annihilation operator is being applied during projection.
The Hamiltonian evolution already included Jordan-Wigner signs.
```

Required test:

```text
TestConditionalProjectionPreservesPhase
```

Add a permanent convention warning in bridge report:

```text
jordan_wigner_partition_convention
```

Message:

```text
Conditional vectors are projected in the canonical occupation basis. No additional Jordan-Wigner sign is applied during projection; all phases are inherited from the full wavefunction amplitudes.
```

This is a convention warning, not an error.

---

# 12. Conditional projector

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
func BuildConditionalProjector(split SiteSplit, basis []uint64) ConditionalProjector
```

For each basis index:

```text
qConfig = basis[fullIndex]
subQ, envQ = SplitConfig(qConfig, split)
ByEnvironment[envQ] append (fullIndex, subIndex)
```

---

# 13. Environment-projected conditional vector

Add:

```go
type ConditionalVector struct {
    EnvConfig  uint64
    Vector     []complex128
    Norm       float64
    Normalized bool
}
```

Function:

```go
func EnvironmentProjectedConditionalVector(
    psi []complex128,
    envQ uint64,
    projector ConditionalProjector,
) (ConditionalVector, error)
```

Use:

```go
const ConditionalNormFloor = 1e-12
```

Behavior:

```text
If norm < ConditionalNormFloor:
  return Normalized=false
  record conditional_norm_failure
  skip fidelity calculation for that sample/transition
Else:
  normalize vector
```

Norm-failure thresholds:

```text
NormFailureInconclusiveRate = 0.10
NormFailureFailedRate = 0.50
```

---

# 14. Fidelity drop

Function:

```go
func ConditionalFidelity(a, b []complex128) (float64, error)
```

Formula:

```text
F = |Σ_i conj(a_i) b_i|²
drop = 1 - F
```

Numerical rule:

```text
Clamp tiny drift to [0,1] if within 1e-12.
If outside tolerance, mark bridge_audit_inconclusive.
```

Fidelity is sufficient for `0002A`.

Do not add trace distance in this ticket.

---

# 15. Jump classification

Classify each sampled transition into mutually exclusive classes:

```text
no_jump:
  Q_A unchanged and Q_B unchanged

strict_environment_jump:
  Q_B changed and Q_A unchanged

strict_subsystem_jump:
  Q_A changed and Q_B unchanged

boundary_crossing_jump:
  Q_A changed and Q_B changed
```

A boundary-crossing jump must be counted separately because it directly changes the subsystem and can fake a conditional-update correlation.

Metrics must include:

```json
{
  "strict_environment_jump_transitions": 0,
  "strict_subsystem_jump_transitions": 0,
  "boundary_crossing_jump_transitions": 0,
  "no_jump_transitions": 0,
  "any_jump_transitions": 0
}
```

Boundary-crossing jumps are:

```text
not included in primary numerator
not included in primary denominator
reported separately
```

Required tests:

```text
TestBoundaryCrossingJumpBothSidesChanged
TestBoundaryCrossingJumpExcludedFromPrimaryRatio
TestBoundaryCrossingJumpReportedSeparately
```

---

# 16. Primary conditional update ratio

Primary ratio:

```text
conditional_update_ratio =
  mean_fidelity_drop_at_strict_environment_jumps
  /
  mean_fidelity_drop_no_jump
```

Rationale:

```text
strict_environment_jump isolates transitions where only Q_B changed.
no_jump isolates smooth ψ evolution with no configuration change.
This avoids fake correlations from direct subsystem changes.
```

Do not use boundary-crossing jumps in the primary ratio.

Do not use strict subsystem jumps in the primary ratio.

Event-count rule:

```text
strict_environment_jump_transitions >= 10
no_jump_transitions >= 10
mean_fidelity_drop_no_jump > 1e-14
```

Otherwise:

```json
{
  "conditional_update_ratio": null,
  "conditional_update_ratio_status": "unavailable_insufficient_events",
  "conditional_update_ratio_env_event_count": 3,
  "conditional_update_ratio_no_jump_event_count": 7
}
```

The ratio never determines `bridge_status`.

---

# 17. Secondary metrics

Report these if tracked:

```text
mean_fidelity_drop_at_strict_environment_jumps
mean_fidelity_drop_at_strict_subsystem_jumps
mean_fidelity_drop_at_boundary_crossing_jumps
mean_fidelity_drop_no_jump
mean_fidelity_drop_at_any_jumps
```

Do not include `fixed_environment_reference_mean_drop` in `0002A`.

Defer fixed-environment or Schrödinger-only baseline work to `BELL-MIPT-0002B`.

---

# 18. Empirical trajectory equivariance

At sampled steps only:

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

This is descriptive only.

Do not gate `bridge_status` on empirical L1 unless:

```text
NaN/Inf appears
probabilities are invalid
counts are structurally inconsistent
```

Required tests:

```text
TestEmpiricalEquivarianceSampledStepsOnly
TestEmpiricalEquivarianceAgainstPsiSquared
```

The statistical agreement test must use a loose tolerance and fixed seed.

---

# 19. Bridge statuses

Statuses:

```text
bridge_disabled
bridge_audit_completed
bridge_audit_failed
bridge_audit_inconclusive
```

## bridge_disabled

```text
bridge omitted or bridge.enabled=false
```

## bridge_audit_failed

Use for implementation-level failures:

```text
invalid bridge config
invalid basis/index mapping
invalid probability distribution
invalid rate matrix
rate orientation inconsistency
structural split/combine error
forbidden-language audit failure
missing required report fields
```

## bridge_audit_inconclusive

Use when run completes but diagnostics are not reliable:

```text
max_lambda_dt > 0.20
norm failure rate > 10%
near-zero current configuration probability rate > 1%
no valid fidelity comparisons
strict_environment_jump_transitions == 0
no_jump_transitions == 0
NaN/Inf in bridge metrics
severe finite-sample pathology
```

If norm failure rate > 50%, prefer `bridge_audit_inconclusive` unless it reflects a coding error.

## bridge_audit_completed

Use only if:

```text
trajectories sampled successfully
conditional vectors computed with acceptable norm failure rate
fidelity metrics emitted or correctly null-gated
empirical trajectory equivariance diagnostic emitted
max_lambda_dt <= 0.20
no structural/numerical fatal error
forbidden-language audit passed
```

Important:

```text
conditional_update_ratio does not determine bridge_status.
```

---

# 20. Environment-correlated conditional update interpretation

Add:

```go
type BridgeInterpretation struct {
    EnvironmentCorrelatedConditionalUpdate string `json:"environment_correlated_conditional_update"`
    Reason string `json:"reason"`
}
```

Allowed values:

```text
not_assessed
no_clear_correlation
weak_correlation
candidate_correlation
```

Rules:

```text
if bridge disabled:
  not_assessed

if bridge inconclusive or failed:
  not_assessed

if ratio unavailable:
  not_assessed

if ratio < 1.5:
  no_clear_correlation

if 1.5 <= ratio < 3.0:
  weak_correlation

if ratio >= 3.0:
  candidate_correlation
```

Reason text must always include:

```text
Descriptive finite-toy diagnostic only. This does not establish a Bell-MIPT bridge.
```

Do not use:

```text
monitoring_like_signal
measurement-like
MIPT-like
phase transition
validated bridge
```

---

# 21. Bridge report schema

When bridge enabled:

```json
{
  "schema_version": "bell_mipt_report_v0_2a",
  "toy_analysis_only": true,
  "physics_claim": "none",
  "goal_status": "toy_goal_passed",
  "bridge": {
    "enabled": true,
    "bridge_goal": "sample_bell_trajectories_and_audit_environment_projected_conditional_vectors",
    "bridge_status": "bridge_audit_completed",

    "subsystem_sites_requested": [5, 0, 2],
    "environment_sites_requested": [1, 3, 4],
    "subsystem_sites_canonical": [0, 2, 5],
    "environment_sites_canonical": [1, 3, 4],

    "trajectories": 200,
    "sample_every_steps": 1,

    "metrics": {
      "trajectory_count": 200,

      "mean_jump_count": 0.0,
      "mean_strict_environment_jump_count": 0.0,
      "mean_strict_subsystem_jump_count": 0.0,
      "mean_boundary_crossing_jump_count": 0.0,

      "conditional_norm_failures": 0,

      "mean_fidelity_drop_at_strict_environment_jumps": null,
      "mean_fidelity_drop_at_strict_subsystem_jumps": null,
      "mean_fidelity_drop_at_boundary_crossing_jumps": null,
      "mean_fidelity_drop_no_jump": null,
      "mean_fidelity_drop_at_any_jumps": null,

      "conditional_update_ratio": null,
      "conditional_update_ratio_status": "unavailable_insufficient_events",
      "conditional_update_ratio_env_event_count": 0,
      "conditional_update_ratio_no_jump_event_count": 0,

      "initial_empirical_equivariance_l1": 0.0,
      "max_empirical_equivariance_l1": 0.0,
      "final_empirical_equivariance_l1": 0.0,

      "max_lambda_dt": 0.0,
      "near_zero_current_configuration_probability_count": 0,

      "strict_environment_jump_transitions": 0,
      "strict_subsystem_jump_transitions": 0,
      "boundary_crossing_jump_transitions": 0,
      "no_jump_transitions": 0,
      "any_jump_transitions": 0
    },

    "interpretation": {
      "environment_correlated_conditional_update": "not_assessed",
      "reason": "Descriptive finite-toy diagnostic only. This does not establish a Bell-MIPT bridge."
    },

    "warnings": []
  }
}
```

When bridge disabled:

```text
Do not emit bridge section.
Keep schema_version = bell_mipt_report_v0.
```

---

# 22. Structured warnings

Add:

```go
type BridgeWarning struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

Warning codes:

```text
large_lambda_dt_warning
severe_lambda_dt_inconclusive
low_strict_environment_event_count
low_no_jump_event_count
conditional_norm_failures
finite_sample_noise
finite_size_toy
near_zero_current_configuration_probability
jordan_wigner_partition_convention
sparse_sampling_warning
boundary_crossing_jumps_observed
```

---

# 23. Required limitations

Report must include:

```text
This samples Bell configuration histories in a finite toy model only.
This conditional-vector audit is not a monitored quantum trajectory simulation.
This does not implement MIPT.
This does not show Bell jumps are measurements.
This does not establish a Bell-MIPT bridge.
This does not support any holography or black-hole claim.
This is not a physics promotion.
Finite-size effects may dominate this diagnostic.
Finite-sample noise may dominate empirical trajectory equivariance.
Boundary-crossing jumps are reported separately because they directly change the subsystem.
```

---

# 24. Forbidden-language audit

The audit must scan fully assembled:

```text
report.json
report.md
bridge interpretation reason
warnings
limitations
debt status
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

Allow negated limitations:

```text
This does not show Bell jumps are measurements.
This does not establish a Bell-MIPT bridge.
No MIPT claim.
No holography claim.
No physics promotion.
not a monitored quantum trajectory simulation
```

Do not ban raw words globally.

---

# 25. Debt status

Before implementation:

```json
{
  "needMap": "ready_for_repaired_conditional_vector_toy_attempt",
  "needInvariant": "0001_partially_paid; 0002A_empirical_trajectory_diagnostic_planned",
  "needToyCheck": "0002A_plan_repaired_and_reviewed_pending_implementation",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements_preserved",
  "needFaithfulnessReview": "0002A_pending_source_review_after_implementation",
  "promotion_status": "unpromoted_final_plan_review_only"
}
```

After implementation, only if:

```text
bridge_status == bridge_audit_completed
```

then:

```json
{
  "needMap": "partially_paid_environment_projected_conditional_vector_toy_only",
  "needInvariant": "partially_paid_equivariance_plus_descriptive_empirical_trajectory_check",
  "needToyCheck": "partially_paid_rate_algebra_and_conditional_vector_toy",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_review_required_for_0002A",
  "promotion_status": "unpromoted_toy_diagnostic_only"
}
```

If:

```text
bridge_audit_inconclusive
bridge_audit_failed
bridge_disabled
```

then do not advance debt status.

---

# 26. Validation commands

Run:

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge
cat /tmp/bellmipt-default/report.json
cat /tmp/bellmipt-bridge/report.json
cat /tmp/bellmipt-bridge/report.md
```

Manual acceptance checks:

```text
Default run preserves 0001 goal_status and metrics.
Bridge-enabled run emits schema_version bell_mipt_report_v0_2a.
Bridge-enabled run emits bridge_audit_* status.
No MIPT/holography/measurement/promotion overclaim appears.
Forbidden-language audit passes.
Debt advances only if bridge_audit_completed.
Boundary-crossing jumps are excluded from primary ratio.
Unavailable ratios serialize as null with status and event counts.
```

---

# 27. Final implementation sequence

```text
Phase 0: Pre-implementation
  Define tests first.
  Confirm QIndex/QConfig mapping.
  Confirm existing probability floor.

Phase 1: Config and validation
  Add BridgeConfig.
  Add bridge config validation.
  Add bridge-disabled regression.

Phase 2: Site split and projector
  Implement SiteSplit.
  Implement SplitConfig and CombineConfig.
  Implement ConditionalProjector.

Phase 3: Conditional vector
  Implement environment-projected conditional vector.
  Implement fidelity.
  Add norm failure logic.

Phase 4: Trajectory sampler
  Implement SampleDiscrete.
  Implement StepBellConfiguration.
  Implement max_lambda_dt.
  Implement near-zero probability guard.
  Implement deterministic RNG.

Phase 5: Bridge metrics
  Implement strict jump classification.
  Implement boundary-crossing exclusion.
  Implement primary ratio null-gating.
  Implement empirical trajectory equivariance.

Phase 6: Reports
  Add bridge report schema.
  Add warnings.
  Add limitations.
  Extend forbidden audit.

Phase 7: Validation
  Run all tests.
  Run default and bridge-enabled CLI.
  Review report artifacts.
```

---

# 28. Final acceptance boundary

Even if every test passes and bridge status is `bridge_audit_completed`, the only allowed conclusion is:

```text
Bell trajectories were sampled, empirical trajectory equivariance was checked descriptively, and environment-projected conditional-vector changes were measured for the tested finite toy configuration.
```

Not allowed:

```text
Bell jumps behave like measurements.
Bell-MIPT bridge is established.
MIPT was observed.
Holography was explained.
Bohmian mechanics was validated.
```

## Required implementation report from coding agent

After coding, report:

```text
1. Files added/modified.
2. Whether bridge-disabled default run preserves 0001 semantic metrics.
3. Default run goal_status and key 0001 metrics.
4. Bridge-enabled run bridge_status.
5. Key bridge metrics:
   - trajectory_count
   - mean_jump_count
   - mean_strict_environment_jump_count
   - mean_strict_subsystem_jump_count
   - mean_boundary_crossing_jump_count
   - conditional_norm_failures
   - conditional_update_ratio and status
   - max_lambda_dt
   - empirical equivariance initial/max/final L1
   - near_zero_current_configuration_probability_count
6. Warnings emitted.
7. Validation commands run.
8. Known limitations.
9. EBP debt update.
10. Explicit non-claim:
    No Bell-MIPT bridge claim, no MIPT claim, no measurement claim, no holography claim, no physics promotion.
```

That is the final prompt I would give the coding agent.

