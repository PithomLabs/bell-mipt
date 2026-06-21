## BELL-MIPT-0002A implementation result report

```json
{
  "ticket": "BELL-MIPT-0002A",
  "implementation_status": "completed_reported_validated",
  "artifact_review_status": "accept_with_minor_repairs",
  "physics_claim": "none",
  "promotion_status": "unpromoted_toy_diagnostic_only"
}
```

The uploaded artifacts show that the bridge-enabled run used the intended non-contiguous partition: subsystem sites `[0, 2, 5]`, environment sites `[1, 3, 4]`, 200 trajectories, seed `42`, and `sample_every_steps: 1`. 

## Baseline 0001 result preserved

The original finite Bell-rate/equivariance audit still passes:

| Item                        |            Result |
| --------------------------- | ----------------: |
| `goal_status`               | `toy_goal_passed` |
| Hilbert dimension           |                64 |
| Max Hermitian error         |                 0 |
| Max norm error              |        `1.81e-13` |
| Max ρ-sum error             |        `7.77e-16` |
| Max equivariance L1 error   |        `3.13e-11` |
| Final equivariance L1 error |        `1.78e-11` |
| Probability floor hits      |                 0 |

All core checks are true: Hamiltonian Hermitian, norm preserved, ρ sum preserved, current antisymmetric, rates nonnegative, equivariance within tolerance, no NaN/Inf, and forbidden-language audit passed. 

## Bridge audit result

The bridge audit completed:

```json
{
  "bridge_status": "bridge_audit_completed",
  "environment_correlated_conditional_update": "candidate_correlation"
}
```

Key bridge metrics:

| Metric                                            |   Value |
| ------------------------------------------------- | ------: |
| Trajectories                                      |     200 |
| Mean jump count                                   |    2.39 |
| Mean strict environment jump count                |   0.375 |
| Mean strict subsystem jump count                  |    0.36 |
| Mean boundary-crossing jump count                 |   1.655 |
| Strict environment jump transitions               |      75 |
| Strict subsystem jump transitions                 |      72 |
| Boundary-crossing transitions                     |     331 |
| No-jump transitions                               | 199,522 |
| Conditional norm failures                         |       0 |
| Max `lambda*dt`                                   | 0.01669 |
| Near-zero current configuration probability count |       0 |
| Initial empirical equivariance L1                 |  0.4377 |
| Max empirical equivariance L1                     |  0.4615 |
| Final empirical equivariance L1                   |  0.3455 |

The `max_lambda_dt` value is below the final warning threshold `0.05`, so the discrete-time thinning regime looks numerically acceptable for this toy run. Conditional norm failures and near-zero-current-configuration events are both zero. 

## Conditional-update diagnostic

The run reports:

| Metric                                         |        Value |
| ---------------------------------------------- | -----------: |
| Mean fidelity drop at strict environment jumps |   `0.878568` |
| Mean fidelity drop at strict subsystem jumps   | `0.00000415` |
| Mean fidelity drop at boundary-crossing jumps  |   `0.887823` |
| Mean fidelity drop at no-jump transitions      | `0.00000405` |
| Conditional update ratio                       |  `217088.49` |
| Ratio status                                   |  `available` |
| Env event count                                |           75 |
| No-jump event count                            |      199,522 |

This is a strong finite-toy diagnostic correlation, but it is **not** a Bell–MIPT bridge result. The report’s interpretation correctly says: “Descriptive finite-toy diagnostic only. This does not establish a Bell-MIPT bridge.” 

## Warnings emitted

Two bridge warnings were emitted:

```text
jordan_wigner_partition_convention
boundary_crossing_jumps_observed
```

The first states that conditional vectors are projected in the canonical occupation basis with no extra Jordan-Wigner sign added during projection. The second records that boundary-crossing jumps occurred. 

## Implementation checklist status

The task checklist marks all phases complete:

```text
Phase 0: Pre-implementation
Phase 1: Config and validation
Phase 2: Site split and projector
Phase 3: Conditional vector
Phase 4: Trajectory sampler
Phase 5: Bridge metrics
Phase 6: Reports
Phase 7: Validation
```

It specifically reports completion of bridge config validation, `SiteSplit`, `SplitConfig`, `CombineConfig`, `ConditionalProjector`, environment-projected conditional vectors, fidelity, `ConditionalNormFloor = 1e-12`, `SampleDiscrete`, `StepBellConfiguration`, `max_lambda_dt`, near-zero probability guard, deterministic RNG, strict jump classification, boundary-crossing exclusion, ratio null-gating, empirical trajectory equivariance, bridge report schema, warnings, limitations, and forbidden audit extension. 

The walkthrough also reports that the two-pass architecture was implemented so trajectory tracking stays separate from primary state evolution, preserving the `BELL-MIPT-0001` metrics when the bridge is disabled. 

## Minor repairs before final acceptance

I would not call this rejected. I would call it:

```text
accept_with_minor_artifact_repairs
```

Repairs:

1. **Update limitations text.** The JSON/Markdown limitations still mostly show the older `0001` limitations and are missing several final-plan limitations, including finite-size effects, finite-sample noise, boundary-crossing handling, and “conditional-vector audit is not a monitored quantum trajectory simulation.” 

2. **Replace this limitation line:**

```text
This does not construct a conditional-wave-function bridge.
```

with something closer to:

```text
This constructs an environment-projected conditional-vector toy diagnostic only; it does not establish a Bell-MIPT bridge.
```

3. **Serialize forbidden audit hits as `[]` instead of `null`.** This is a low-priority polish item, but it has appeared before. 

4. **Add missing warnings if final artifact contract requires them:** `finite_sample_noise` and `finite_size_toy` would be appropriate because empirical trajectory L1 is noisy at 200 trajectories over 64 states.

## EBP ledger

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

## Final interpretation

The implementation appears to have completed the intended `BELL-MIPT-0002A` toy diagnostic. The base Bell-rate/equivariance audit still passes, the bridge audit completed, boundary-crossing jumps were separated, conditional norm failures were zero, `max_lambda_dt` stayed safely below threshold, and the conditional-update ratio was available.

This supports only the narrow statement:

```text
For this finite 6-site toy configuration, sampled Bell trajectories produced measurable environment-correlated changes in environment-projected conditional vectors.
```

It does **not** establish MIPT, measurement behavior, holography, black-hole relevance, or a Bell–MIPT bridge.

