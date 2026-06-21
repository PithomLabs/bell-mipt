# Adversarial Review Prompt: BELL-MIPT-0002A Implementation Results and Report Artifacts

You are performing an adversarial review of the **implemented BELL-MIPT-0002A report results**, not merely the plan.

The implementation adds a Bell trajectory sampler and environment-projected conditional-vector audit to the accepted `BELL-MIPT-0001` finite toy model.

Your task is to determine whether the uploaded implementation artifacts should be:

```text
accepted_for_limited_toy_scope
accepted_with_minor_repairs
accepted_with_high_priority_repairs
rejected_pending_repairs
inconclusive_needs_source_review
```

Do not rubber-stamp the result. Attack the numerical interpretation, artifact contract, report schema, warnings, debt update, limitations, and overclaim risk.

## Files to review

Review these artifacts:

```text
input.json
report.json
report.md
task.md
walkthrough.md
```

If source code is available, also inspect the implementation files and tests. If source code is not available, clearly say that this is an artifact-level review only, not a source-code review.

## Intended ticket

```text
BELL-MIPT-0002A:
Bell trajectory sampler + environment-projected conditional-vector audit
```

This extends the accepted `BELL-MIPT-0001` toy.

The allowed conclusion is only:

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

## Reported run summary to audit

The bridge-enabled run reports approximately:

```json
{
  "schema_version": "bell_mipt_report_v0_2a",
  "goal_status": "toy_goal_passed",
  "bridge_status": "bridge_audit_completed",
  "sites": 6,
  "hilbert_dim": 64,
  "trajectories": 200,
  "sample_every_steps": 1,
  "subsystem_sites": [0, 2, 5],
  "environment_sites": [1, 3, 4],
  "max_equivariance_l1_error": 3.1349902085116635e-11,
  "final_equivariance_l1_error": 1.783272157327917e-11,
  "mean_jump_count": 2.39,
  "mean_strict_environment_jump_count": 0.375,
  "mean_strict_subsystem_jump_count": 0.36,
  "mean_boundary_crossing_jump_count": 1.655,
  "conditional_norm_failures": 0,
  "mean_fidelity_drop_at_strict_environment_jumps": 0.8785681522629386,
  "mean_fidelity_drop_no_jump": 0.000004047050810602201,
  "conditional_update_ratio": 217088.49070076479,
  "conditional_update_ratio_status": "available",
  "conditional_update_ratio_env_event_count": 75,
  "conditional_update_ratio_no_jump_event_count": 199522,
  "initial_empirical_equivariance_l1": 0.4377269240776656,
  "max_empirical_equivariance_l1": 0.4615339524450977,
  "final_empirical_equivariance_l1": 0.34554195201102483,
  "max_lambda_dt": 0.01669145661361399,
  "near_zero_current_configuration_probability_count": 0,
  "strict_environment_jump_transitions": 75,
  "strict_subsystem_jump_transitions": 72,
  "boundary_crossing_jump_transitions": 331,
  "no_jump_transitions": 199522,
  "any_jump_transitions": 478,
  "interpretation": {
    "environment_correlated_conditional_update": "candidate_correlation",
    "reason": "Descriptive finite-toy diagnostic only. This does not establish a Bell-MIPT bridge."
  },
  "warnings": [
    "jordan_wigner_partition_convention",
    "boundary_crossing_jumps_observed"
  ]
}
```

## Key concerns to attack

The report result looks numerically clean in some respects, but it may still have artifact-contract weaknesses.

Pay special attention to these issues:

```text
1. Does bridge_audit_completed overstate the result?
2. Does candidate_correlation overstate the result?
3. Is conditional_update_ratio ≈ 217088 meaningful or mostly an artifact of using an extremely small no-jump denominator?
4. Should such a huge ratio trigger an additional warning about ratio instability or small denominator amplification?
5. Were boundary-crossing jumps properly excluded from the primary ratio?
6. Does the report expose enough evidence that strict_environment_jump_transitions, no_jump_transitions, and boundary_crossing_jump_transitions were counted correctly?
7. Does empirical trajectory equivariance L1 around 0.35–0.46 look acceptable for 200 trajectories over 64 states, or should the report explicitly warn about finite-sample noise?
8. Are warnings incomplete? Should finite_sample_noise, finite_size_toy, large_ratio_warning, and ratio_denominator_small be present?
9. Are limitations incomplete or outdated?
10. Does the limitations section still say “does not construct a conditional-wave-function bridge,” even though the implementation now constructs environment-projected conditional vectors?
11. Does debt_status advance too much given the artifact still needs source-code review?
12. Is needFaithfulnessReview correctly set to source_code_review_required_for_0002A?
13. Does forbidden_language_audit.hits being null instead of [] violate artifact hygiene?
14. Does the report schema preserve 0001 semantic fields while adding 0002A fields?
15. Are the bridge-disabled semantics actually verified, or only claimed in walkthrough/task files?
```

## Specific adversarial questions

Answer each question directly.

### A. Baseline 0001 preservation

1. Does the report preserve the original BELL-MIPT-0001 goal and metrics?
2. Does `goal_status: toy_goal_passed` remain tied only to the finite Bell-rate/master-equation equivariance audit?
3. Do the 0001 metrics match the earlier accepted run closely enough?
4. Does schema version `bell_mipt_report_v0_2a` in a bridge-enabled run make sense?
5. Should bridge-disabled output remain `bell_mipt_report_v0` with no bridge section?

### B. Bridge audit status

6. Is `bridge_audit_completed` justified by the metrics?
7. Should `bridge_audit_completed` require source-code review, or is artifact-level completion enough?
8. Should the presence of 331 boundary-crossing jumps weaken `bridge_audit_completed`?
9. Should no conditional norm failures and low `max_lambda_dt` be sufficient for completion?
10. Should empirical L1 around 0.35–0.46 make the audit inconclusive, or is it acceptable as a finite-sample diagnostic?

### C. Conditional update ratio

11. Is the reported ratio of roughly `217088` mathematically valid?
12. Is the ratio meaningful given the denominator `mean_fidelity_drop_no_jump ≈ 4e-6`?
13. Should the report include a warning for extreme ratio amplification?
14. Should the ratio be capped, log-scaled, or reported with numerator/denominator counts only?
15. Should `candidate_correlation` be downgraded to `weak_correlation` or `no_clear_correlation` until null models exist?
16. Should `candidate_correlation` require a null-model comparison beyond no-jump transitions?
17. Does the ratio accidentally suggest measurement-like behavior despite the disclaimer?
18. Is `candidate_correlation` safe terminology under EBP 2.1?

### D. Boundary-crossing jumps

19. Are boundary-crossing jumps properly separated from strict environment jumps?
20. Are boundary-crossing jumps excluded from the primary ratio?
21. Does the report make this exclusion clear enough?
22. Should the large boundary-crossing count, 331 transitions, dominate interpretation?
23. Should there be a stronger warning that boundary-crossing jumps directly change the subsystem?
24. Should any ratio involving strict environment jumps be interpreted cautiously when boundary crossings are more frequent than strict environment jumps?

### E. Empirical trajectory equivariance

25. Is initial empirical L1 ≈ 0.438 reasonable for 200 samples over 64 states?
26. Is final empirical L1 ≈ 0.346 reasonable?
27. Does the report adequately say empirical trajectory equivariance is descriptive only?
28. Should finite_sample_noise warning be mandatory?
29. Should confidence intervals or rough multinomial expectations be reported?
30. Does empirical equivariance support sampler sanity, or is it too noisy to say much?

### F. Numerical reliability

31. Is `max_lambda_dt ≈ 0.0167` safely below the warning threshold?
32. Does low `max_lambda_dt` justify at-most-one-jump-per-step discrete-time thinning?
33. Is `near_zero_current_configuration_probability_count = 0` enough to clear the node/division risk?
34. Are conditional norm failures being zero enough to trust fidelity computations?
35. Should the report include min/max conditional norms or denominator diagnostics?

### G. Report/schema/limitations

36. Are required 0002A limitations present?
37. Which limitations are missing?
38. Is the line “This does not construct a conditional-wave-function bridge” outdated or misleading?
39. Should the limitations explicitly say “This constructs only an environment-projected conditional-vector toy diagnostic”?
40. Should warnings include finite_size_toy and finite_sample_noise by default?
41. Should forbidden_language_audit.hits be `[]` instead of `null`?
42. Does the report include all necessary bridge fields?
43. Are JSON and Markdown consistent?
44. Does Markdown omit fields that are important for audit, such as strict transition counts?
45. Should the report include the full input config hash or artifact provenance?

### H. EBP debt status

46. Is `needMap: partially_paid_environment_projected_conditional_vector_toy_only` justified?
47. Is `needInvariant: partially_paid_equivariance_plus_descriptive_empirical_trajectory_check` justified?
48. Is `needToyCheck: partially_paid_rate_algebra_and_conditional_vector_toy` justified?
49. Should any of those be downgraded to `attempted_*_pending_source_review`?
50. Does `needNullModel: unpaid` correctly capture the remaining gap?
51. Does `needObstruction: bell_jumps_are_not_measurements` remain sufficiently visible?
52. Is `needFaithfulnessReview: source_code_review_required_for_0002A` sufficient, or should artifact review require source review before any debt advancement?

### I. Validation evidence

53. Does task.md provide enough evidence that tests actually ran and passed?
54. Does walkthrough.md provide enough evidence of validation commands?
55. Are actual command outputs included?
56. Should this review require `go test ./...`, `go test -race ./...`, and `go vet ./...` logs?
57. Should source code be reviewed before acceptance?
58. Should a fresh clean checkout run be required?
59. Should bridge-disabled run artifacts be uploaded for comparison?
60. Should a second bridge config with different partition/seed be run?

## Expected output format

Return your review in this structure.

### Executive verdict

Choose one:

```text
accept_for_limited_toy_scope
accept_with_minor_repairs
accept_with_high_priority_repairs
reject_pending_repairs
inconclusive_needs_source_review
```

### Strongest evidence for acceptance

List the strongest evidence from the report artifacts.

### Critical blockers

List blockers, if any.

### High-priority repairs

List repairs required before acceptance.

### Medium-priority repairs

List useful but non-blocking repairs.

### Low-priority polish

List artifact hygiene improvements.

### Specific answer table

Answer all 60 questions in a compact table:

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

### Artifact contract verdict

Evaluate:

```json
{
  "schema_contract": "...",
  "report_markdown_contract": "...",
  "warnings_contract": "...",
  "limitations_contract": "...",
  "debt_status_contract": "...",
  "forbidden_language_contract": "...",
  "validation_evidence_contract": "..."
}
```

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
  "promotion_status": "unpromoted_artifact_review_only"
}
```

### Final recommendation

Choose one:

```text
Accept current artifacts for limited toy scope.
Accept after minor artifact repairs.
Require source-code review before acceptance.
Reject and repair implementation.
```

## Review standard

Be strict.

The implementation should only be accepted if the artifacts are:

```text
auditable
numerically honest
EBP-safe
clear about non-claims
clear about boundary-crossing jumps
clear that candidate_correlation is descriptive only
not likely to manufacture a fake Bell-MIPT bridge result
```

Do not reward a large conditional_update_ratio unless the report explains why it is safe to interpret only as a finite-toy diagnostic.

