# Final Adversarial Review Prompt: Repaired BELL-MIPT-0002A Plan

You are performing the **final adversarial review before implementation** for:

```text
BELL-MIPT-0002A:
Bell trajectory sampler + environment-projected conditional-vector audit
```

This is the repaired implementation plan after prior adversarial reviews. Your job is to determine whether the repaired plan is now safe, precise, and implementation-ready.

Do **not** merely summarize the plan. Attack it.

Look for remaining conceptual mistakes, hidden overclaims, mathematical ambiguities, software-design traps, numerical instability, report/schema weaknesses, and insufficient tests.

## 1. Ticket boundary

`BELL-MIPT-0001` has already been accepted for limited toy scope. It computes Bell probability currents and Bell positive-current jump rates for a finite Kitaev-chain-style fermionic lattice, evolves `ψ` and Bell master-equation `ρ`, and verifies numerical equivariance: `ρ(t)` tracks `|ψ(t)|²`.

`BELL-MIPT-0002A` must extend this only by:

```text
1. Sampling actual Bell configuration trajectories Q(t).
2. Splitting Q(t) into subsystem Q_A(t) and environment Q_B(t).
3. Constructing an environment-projected conditional vector:
      ψ_A(a,t | b) = Ψ(full_config(a,b), t)
   using the actual environment configuration b = Q_B(t).
4. Measuring fidelity-drop diagnostics between consecutive conditional vectors.
5. Comparing environment-jump transitions against non-environment-jump transitions.
6. Reporting empirical trajectory equivariance as a descriptive diagnostic.
7. Reporting max_lambda_dt as a discrete-time thinning reliability diagnostic.
```

Allowed conclusion, if implementation succeeds:

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

## 2. Repaired plan summary

The repaired plan uses a two-pass architecture.

### Pass 1

```text
Run accepted BELL-MIPT-0001 master-equation/equivariance audit.
Store ψ snapshots at full time-step boundaries.
Preserve all existing 0001 goal_status and metrics.
```

### Pass 2

Only if `bridge.enabled=true`:

```text
Use ψ[k] snapshots to sample Bell configuration trajectories.
Compute environment-projected conditional vectors from Q_B[k].
Compute fidelity-drop diagnostics.
Compute empirical trajectory equivariance at sampled steps.
Compute max_lambda_dt.
Emit bridge_audit_* status and warnings.
```

The top-level `goal_status` remains the original 0001 status. It must remain independent of the new bridge audit.

## 3. Repaired terminology

The repaired plan removes the old phrase:

```text
monitoring_like_signal
```

It now uses:

```text
environment_correlated_conditional_update
```

Allowed values:

```text
not_assessed
no_clear_update
weak_update_enhancement
candidate_update_enhancement
```

The bridge statuses are now:

```text
bridge_disabled
bridge_audit_completed
bridge_audit_failed
bridge_audit_inconclusive
```

The plan explicitly says:

```text
conditional_update_ratio never determines bridge_status.
```

## 4. Repaired time-step alignment

The repaired plan defines the exact time-step convention:

```text
At interval k -> k+1:

Given:
  ψ[k] = universal wavefunction at t_k
  Q[k] = actual configuration at t_k

Do:
  1. Compute Bell rates σ(dest <- src) from ψ[k].
  2. Use rates[dest][Q[k]] to sample Q[k+1].
  3. ψ[k+1] is already produced by the existing RK4 evolution.
  4. Conditional vector at sample k uses (ψ[k], Q_B[k]).
  5. Conditional vector at sample k+1 uses (ψ[k+1], Q_B[k+1]).
```

No jump sampling occurs at RK4 sub-stages in `0002A`. Jumps are sampled only at full time-step boundaries.

## 5. Repaired rate convention

The repaired plan locks the rate orientation:

```text
rates[dest][src] = σ(dest <- src)
```

For current configuration `q`:

```text
outgoing_rate_to_dest = rates[dest][q]
lambda(q) = Σ_dest rates[dest][q]
```

The implementation must reuse the same Bell current/rate definitions as `BELL-MIPT-0001`. A row/column orientation bug is a critical failure.

## 6. Repaired partition/Jordan-Wigner handling

The repaired plan says:

```text
The bridge audit constructs an environment-projected conditional vector in the canonical occupation basis:
  ψ_A(a,t | b) = Ψ(full_config(a,b), t)

No additional Jordan-Wigner sign is introduced during projection.
All phases/signs are inherited from the full wavefunction amplitude Ψ(q,t).
```

The plan must support non-contiguous partitions, such as:

```text
A = [0,2,5]
B = [1,3,4]
```

It reports both requested and canonical site lists:

```json
{
  "subsystem_sites_requested": [0, 2, 5],
  "environment_sites_requested": [1, 3, 4],
  "subsystem_sites_canonical": [0, 2, 5],
  "environment_sites_canonical": [1, 3, 4]
}
```

Required invariant:

```text
CombineConfig(SplitConfig(q)) == q
```

for all full configurations `q`.

## 7. Repaired discrete-time thinning safeguards

Trajectory sampling uses:

```text
p_jump = 1 - exp(-lambda * dt)
```

and samples at most one jump per `dt`.

The repaired plan adds:

```text
max_lambda_dt = max over all trajectory micro-steps of lambda * dt
```

Thresholds:

```text
max_lambda_dt <= 0.1:
  acceptable toy regime

0.1 < max_lambda_dt <= 0.5:
  bridge_audit_completed with warning

max_lambda_dt > 0.5:
  bridge_audit_inconclusive
```

This is a numerical reliability diagnostic, not a physics criterion.

## 8. Repaired ratio/event-count logic

The repaired plan only reports `conditional_update_ratio` when event counts are sufficient:

```text
environment_jump_transitions >= 10
non_environment_jump_transitions >= 10
denominator_mean_drop > 1e-14
```

Otherwise:

```json
{
  "conditional_update_ratio": null,
  "conditional_update_ratio_status": "unavailable_insufficient_events",
  "conditional_update_ratio_env_event_count": 3,
  "conditional_update_ratio_non_env_event_count": 7
}
```

The ratio is descriptive only and never promotes the result.

## 9. Repaired thresholds

The repaired constants are:

```text
ConditionalNormFloor = 1e-14
NormFailureInconclusiveRate = 0.10
NormFailureFailedRate = 0.50

LambdaDTWarning = 0.10
LambdaDTInconclusive = 0.50

MinEnvEventsForRatio = 10
MinNonEnvEventsForRatio = 10
RatioDenominatorFloor = 1e-14
```

## 10. Repaired report schema

When bridge is enabled, the repaired report uses:

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
    "subsystem_sites_requested": [0, 1, 2],
    "environment_sites_requested": [3, 4, 5],
    "subsystem_sites_canonical": [0, 1, 2],
    "environment_sites_canonical": [3, 4, 5],
    "trajectories": 200,
    "sample_every_steps": 1,
    "metrics": {
      "trajectory_count": 200,
      "mean_jump_count": 0.0,
      "mean_environment_jump_count": 0.0,
      "mean_subsystem_jump_count": 0.0,
      "conditional_norm_failures": 0,
      "mean_fidelity_drop_at_environment_jumps": null,
      "mean_fidelity_drop_without_environment_jumps": null,
      "mean_fidelity_drop_at_any_jumps": null,
      "mean_fidelity_drop_no_jump": null,
      "conditional_update_ratio": null,
      "conditional_update_ratio_status": "unavailable_insufficient_events",
      "conditional_update_ratio_env_event_count": 0,
      "conditional_update_ratio_non_env_event_count": 0,
      "initial_empirical_equivariance_l1": 0.0,
      "max_empirical_equivariance_l1": 0.0,
      "final_empirical_equivariance_l1": 0.0,
      "max_lambda_dt": 0.0,
      "environment_jump_transitions": 0,
      "non_environment_jump_transitions": 0,
      "any_jump_transitions": 0,
      "no_jump_transitions": 0,
      "fixed_environment_reference_mean_drop": null
    },
    "interpretation": {
      "environment_correlated_conditional_update": "not_assessed",
      "reason": "Descriptive finite-toy diagnostic only. This does not establish a Bell-MIPT bridge."
    },
    "warnings": []
  }
}
```

## 11. Required warnings

The repaired plan adds a structured warning type:

```go
type BridgeWarning struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

Warnings should be emitted for:

```text
large_lambda_dt_warning
severe_lambda_dt_inconclusive
low_environment_event_count
low_non_environment_event_count
conditional_norm_failures
finite_sample_noise
finite_size_toy
near_zero_current_configuration_probability
```

## 12. Required limitations

The report must include:

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
```

## 13. Debt-status rule

Before implementation:

```json
{
  "needMap": "ready_for_repaired_conditional_vector_toy_attempt",
  "needInvariant": "0001_partially_paid; 0002A_empirical_trajectory_diagnostic_planned",
  "needToyCheck": "0002A_plan_repaired_pending_implementation",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements_preserved",
  "needFaithfulnessReview": "0002A_pending_source_review_after_implementation",
  "promotion_status": "unpromoted_plan_repair_only"
}
```

After implementation, only if:

```text
bridge_status == bridge_audit_completed
```

may the debt become:

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

If `bridge_audit_inconclusive` or `bridge_audit_failed`, do **not** advance debt status.

## 14. Mandatory tests in repaired plan

The repaired plan now requires these tests before implementation acceptance:

```text
TestTimeStepAlignmentUsesPsiKForIntervalK
TestRateOrientationDestinationSource
TestFakeRateTrajectoryKnownDestination
TestBridgeRatesMatch0001Rates
TestSplitCombineRoundTripAllConfigs
TestNonContiguousPartition
TestConditionalProjectorNoResign
TestBoundaryCrossingJumpBothSidesChanged
TestZeroRateNoJumpInconclusive
TestNearZeroCurrentConfigurationProbability
TestConditionalNormFailureThresholds
TestConditionalUpdateRatioNullWhenLowEvents
TestMaxLambdaDTWarningAndInconclusive
TestForbiddenAuditScansFullJSONAndMarkdown
TestBridgeDisabledPreserves0001SemanticMetrics
TestEmpiricalEquivarianceSampledStepsOnly
```

## 15. Your review tasks

Answer these directly.

### A. Remaining blockers

1. Are there any remaining conceptual blockers after the repairs?
2. Is the phrase `environment-projected conditional vector` safer than `conditional wave function`?
3. Is `environment_correlated_conditional_update` still too close to measurement/MIPT language?
4. Does the plan still risk implying Bell jumps are measurements?
5. Does the repaired debt-status rule still overstate what is paid?

### B. Time-step and rate correctness

6. Is the repaired time-step convention correct and implementable?
7. Should jump sampling happen before or after `ψ[k+1]` is computed, or does the proposed convention suffice?
8. Is it acceptable not to sample jumps at RK4 sub-stages?
9. Is `rates[dest][src]` unambiguous enough?
10. Are the mandatory rate-orientation tests sufficient?

### C. Conditional-vector construction

11. Does “no additional Jordan-Wigner sign during projection” correctly handle the finite occupation-basis toy?
12. Are non-contiguous partition tests sufficient to catch sign/indexing mistakes?
13. Is canonical sorting of site lists safe, or should user order be preserved?
14. Should the report include both requested and canonical orderings?
15. Does the plan need an explicit `basis_index_to_config` abstraction if basis is not raw bitmask?

### D. Numerical reliability

16. Are the `max_lambda_dt` thresholds reasonable?
17. Should `max_lambda_dt > 0.1` already make the audit inconclusive rather than only warning?
18. Are the norm-failure thresholds reasonable?
19. Is `ConditionalNormFloor = 1e-14` acceptable?
20. How should near-zero `|ψ_q|²` for actual current configuration be handled?

### E. Metrics and interpretation

21. Is fidelity drop enough for `0002A`, or must a trace-distance metric be added now?
22. Is `fixed_environment_reference_mean_drop` useful or scope creep?
23. Is the `conditional_update_ratio` event-count rule sufficient?
24. Should the ratio use `mean_fidelity_drop_without_environment_jumps` or only `mean_fidelity_drop_no_jump` as denominator?
25. Should `environment_jump_transitions` exclude boundary-crossing jumps where both subsystem and environment change?

### F. Report/schema/EBP safety

26. Should bridge section be omitted when disabled or always present?
27. Should schema version bump only when bridge enabled?
28. Are `bridge_audit_completed`, `bridge_audit_failed`, and `bridge_audit_inconclusive` safe names?
29. Are `candidate_update_enhancement` and `weak_update_enhancement` safe names?
30. Are the warnings and limitations sufficient to prevent overclaiming?

### G. Test adequacy

31. Are the mandatory tests sufficient to catch the dangerous bugs?
32. Which test is most likely still missing?
33. Which test is overkill or unnecessary?
34. Should stochastic tests use fake RNGs instead of statistical tolerances?
35. Should bridge-disabled regression compare semantic metrics or byte-for-byte reports?

## 16. Expected output format

Return your review in the following structure.

### Executive verdict

Choose one:

```text
ready_for_implementation
ready_for_implementation_with_minor_repairs
needs_one_more_plan_revision
reject_for_now
```

### Strongest repaired decisions

List the repaired choices that are now solid.

### Remaining blockers

List any true blockers.

### High-priority repairs before coding

List only repairs that must happen before implementation.

### Medium-priority repairs

List useful but non-blocking improvements.

### Low-priority polish

List naming/schema/test polish.

### Answer table

Answer all 35 questions in a compact table:

```text
Question | Answer | Severity | Repair
```

Severity options:

```text
blocker
high
medium
low
none
```

### Final repaired implementation checklist

Give a checklist that the coding agent must follow.

### EBP 2.1 ledger

Use:

```json
{
  "needMap": "...",
  "needInvariant": "...",
  "needToyCheck": "...",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "...",
  "promotion_status": "unpromoted_final_plan_review_only"
}
```

### Final recommendation

Choose one:

```text
Proceed to implementation.
Proceed after minor plan edits.
Request another revision before implementation.
Reject and redesign.
```

## 17. Review standard

Be strict. The repaired plan should only proceed if it is:

```text
implementation-ready
auditable
numerically honest
EBP-safe
not likely to manufacture a fake Bell-MIPT bridge result
```

Do not reward vague guardrails. Check whether the repaired plan has made the dangerous parts mechanically testable.

