Reviewed `implementation_plan3.md`. Verdict:

```json
{
  "review_status": "accept_with_minor_repairs",
  "ready_for_coding_agent_after_patch": true,
  "critical_blockers": [],
  "minor_repairs_required": 6
}
```

The plan captures the main final requirements: two-pass architecture, bridge config, canonical/requested site lists, `QIndex`/`QConfig`, `ψ[k]` time-step alignment, `RateProbabilityFloor`, strict jump classes, boundary-crossing tests, empirical equivariance, bridge reporting, forbidden-language audit, and validation commands. 

## What is already good

The strongest parts are:

```text
1. Two-pass architecture is present.
2. Bridge-disabled path is explicitly protected.
3. QIndex/QConfig separation is included.
4. Time-step alignment using ψ[k] is included.
5. RateProbabilityFloor guard is included.
6. Boundary-crossing jump class and exclusion tests are included.
7. Empirical equivariance at sampled steps is included.
8. Forbidden-language scan over JSON/Markdown is included.
9. go test, race, vet, and CLI smoke validation are included.
```

This is close enough that I would not ask for another planning cycle.

## Minor repairs before giving it to the coding agent

### 1. Clarify primary ratio formula

The plan says `bridgemetrics.go` will “compute conditional update ratio,” but it does not explicitly restate the final repaired rule. Add this exact requirement:

```text
conditional_update_ratio =
  mean_fidelity_drop_at_strict_environment_jumps /
  mean_fidelity_drop_no_jump

Boundary-crossing jumps and strict subsystem jumps must be excluded from the primary ratio.
```

Also add:

```text
If strict_environment_jump_transitions < 10 or no_jump_transitions < 10, serialize conditional_update_ratio as null.
```

### 2. Fix `ConditionalFidelity` wording

The plan says `ConditionalFidelity()` computes:

```text
1 - |Σ conj(a_i)b_i|²
```

That is actually the **fidelity drop**, not fidelity. 

Patch to:

```text
ConditionalFidelity() computes:
  F = |Σ conj(a_i)b_i|²

The fidelity drop is computed separately:
  drop = 1 - F
```

Or rename the function if it returns the drop:

```go
ConditionalFidelityDrop()
```

I prefer keeping `ConditionalFidelity()` as fidelity and computing drop in metrics.

### 3. Add explicit `ConditionalNormFloor = 1e-12`

The plan mentions `RateProbabilityFloor = 1e-14`, but the conditional-vector section only mentions fidelity clamping, not the final conditional norm floor. Add:

```go
const ConditionalNormFloor = 1e-12
```

And:

```text
If conditional norm < ConditionalNormFloor:
  return Normalized=false
  record conditional_norm_failure
  skip fidelity for that transition
```

### 4. Add exact `max_lambda_dt` thresholds

The plan says it tracks `max_lambda_dt`, but it should explicitly include final thresholds:

```text
LambdaDTWarning = 0.05
LambdaDTInconclusive = 0.20

If max_lambda_dt > 0.05:
  emit large_lambda_dt_warning

If max_lambda_dt > 0.20:
  bridge_audit_inconclusive
```

### 5. Add required warning codes

The plan mentions warnings generally, but the final prompt should list the warning codes:

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

### 6. Replace local `file://` links with plain relative paths

The plan uses local `file:///home/...` links. They are fine for your machine, but for a coding agent prompt, plain repo-relative paths are cleaner and portable:

```text
internal/bellmipt/bridge_config.go
internal/bellmipt/trajectory.go
...
```

## Final patch block to add

Add this near the top before “Proposed Changes”:

```text
Final repaired rules:

1. Primary ratio:
   conditional_update_ratio =
     mean_fidelity_drop_at_strict_environment_jumps /
     mean_fidelity_drop_no_jump

2. Boundary-crossing jumps are reported separately and excluded from the primary ratio.

3. ConditionalFidelity returns F = |<a|b>|². Fidelity drop is computed separately as 1 - F.

4. ConditionalNormFloor = 1e-12.

5. LambdaDTWarning = 0.05 and LambdaDTInconclusive = 0.20.

6. Unavailable ratios serialize as null with event-count status fields.

7. The bridge audit must not emit fixed_environment_reference_mean_drop in 0002A.

8. The bridge audit advances EBP debt only if bridge_status == bridge_audit_completed.
```

## Final decision

```json
{
  "verdict": "ready_after_minor_patch",
  "do_not_replan": true,
  "next_step": "apply the patch block above, then hand to coding agent",
  "promotion_status": "unpromoted_implementation_plan_review_only"
}
```

