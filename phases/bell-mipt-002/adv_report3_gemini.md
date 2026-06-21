### Executive verdict

`inconclusive_needs_source_review`

The reported artifacts show successful isolation of boundary-crossing jumps and excellent numerical stability (zero norm failures, safe `max_lambda_dt`). However, the staggering `conditional_update_ratio` of ~217,088 exposes a critical interpretive trap: comparing macroscopic discrete jumps against infinitesimal $\Delta t$ Schrödinger evolution naturally produces an unbounded ratio as $\Delta t \to 0$. Without source code to verify the strict exclusion of boundary jumps and a proper limitation warning regarding small-denominator $\Delta t$ amplification, this artifact cannot be accepted. EBP 2.1 mandates a source-code review before advancing debt.

### Strongest evidence for acceptance

1. **Boundary-Crossing Isolation:** The implementation correctly identified 331 boundary-crossing transitions and successfully separated them from the 75 strict environment jumps, preventing the most obvious source of fake correlation.
2. **Numerical Safety:** `max_lambda_dt` is ~0.0167 (well below the 0.05 warning threshold), and `conditional_norm_failures` is 0, validating the discrete-time sampling and projection mechanics.
3. **Baseline Preservation:** The master-equation equivariance L1 error remains intact at ~3.13e-11, proving the `BELL-MIPT-0001` invariants were preserved.

### Critical blockers

1. **Source Code Absence:** Artifacts alone cannot prove that `rates[dest][src]` is correctly oriented, or that `strict_environment_jump` logic doesn't leak boundary effects. Source-code review is an absolute requirement for EBP.
2. **Ratio Denominator Amplification:** A ratio of 217,088 is an artifact of dividing a discrete jump effect (~0.87) by an infinitesimal unitary integration step (~4e-6). Presenting this without a `dt_scaling_artifact` or `small_denominator_amplification` warning invites dangerous physics overclaims.
3. **Schema Hygiene:** `forbidden_language_audit.hits` being `null` instead of `[]` violates strict JSON schema contracts and implies the audit script may have crashed or been bypassed rather than returning empty.

### High-priority repairs

1. **Add $\Delta t$ Ratio Warning:** Append `small_denominator_amplification` and `finite_size_toy` to the warnings array.
2. **Update Limitations String:** Modify the outdated limitation to explicitly state: "This constructs only an environment-projected conditional-vector toy diagnostic, and ratio magnitudes are heavily dependent on integration step size."
3. **Fix Forbidden Hits Array:** Ensure `forbidden_language_audit.hits` serializes as `[]`.
4. **Provide Validation Logs:** Upload `go test -race ./...` and `go vet` logs to prove the implementation passes CI.

### Medium-priority repairs

1. **Add `finite_sample_noise` Warning:** The empirical equivariance L1 of 0.46 requires a warning that finite trajectories over a 64-dimensional space are purely descriptive noise, not a converged distribution.
2. **Markdown Completeness:** Ensure `report.md` surfaces the exact counts for boundary-crossing vs. strict environment jumps, so human reviewers aren't forced to dig through the JSON to spot the 331 vs 75 discrepancy.

### Low-priority polish

1. **Input Config Provenance:** Embed the input config hash in the final JSON to guarantee which parameters generated the run.
2. **JSON Capping:** Consider formatting massive, highly-sensitive ratios using scientific notation or bounding them in string representations if they exceed $10^4$.

---

### Specific answer table

| Question | Answer | Severity | Repair |
| --- | --- | --- | --- |
| 1. Preserve 0001 goal/metrics? | Yes, L1 equivariance matches the 0001 expectation. | none | None. |
| 2. `goal_status` tied to 0001? | Yes, remains `toy_goal_passed`. | none | None. |
| 3. Metrics match earlier run? | Yes, master equation L1s are identical to typical 0001 outputs. | none | None. |
| 4. Schema version 0_2a makes sense? | Yes, correctly identifies the extended struct. | none | None. |
| 5. Disabled output remains v0? | Cannot verify without the disabled artifact. | high | Require bridge-disabled artifact upload. |
| 6. `bridge_audit_completed` justified? | By metrics, yes. By EBP standards, needs source. | blocker | Require source code review. |
| 7. Require source-code review? | Absolutely. Artifacts can easily hide rate-matrix transposition. | blocker | Conduct source review. |
| 8. 331 boundary jumps weaken it? | No, finding them proves the classifier works, but context is needed. | low | Highlight in Markdown. |
| 9. Zero norm fails/low lambda enough? | Necessary but not sufficient without source logic verification. | medium | Source review. |
| 10. Empirical L1 ~0.46 inconclusive? | It's expected finite-sample noise for N=200, not a failure. | low | Add `finite_sample_noise` warning. |
| 11. Ratio ~217088 mathematically valid? | Yes, 0.87 / 0.000004 = 217,500. Math checks out. | none | None. |
| 12. Meaningful given ~4e-6 denominator? | Barely. It compares discrete jumps to infinitesimal dt scaling. | high | Add `small_denominator_amplification` warning. |
| 13. Warning for extreme amplification? | Mandatory. Without it, the number looks like massive physical correlation. | high | Add warning. |
| 14. Cap, log-scale, or fraction only? | Reporting raw is fine if paired with strict warnings. | medium | Add warning. |
| 15. Downgrade `candidate_correlation`? | "Candidate" is already tentative, but heavily caveat it. | none | Keep, but rely on warnings. |
| 16. Require null-model comparison? | Yes, `no_jump` is a weak null. Debt status reflects this gap. | none | Keep `needNullModel` unpaid. |
| 17. Suggest measurement-like behavior? | The massive ratio risks it if read without EBP constraints. | medium | Enhance limitation strings. |
| 18. Safe terminology under EBP? | Yes, "correlation" is mathematically neutral compared to "signal". | none | None. |
| 19. Boundary jumps separated? | Yes, distinct counts exist in JSON. | none | None. |
| 20. Excluded from primary ratio? | Yes, ratio matches `strict_env` / `no_jump`. | none | None. |
| 21. Exclusion clear enough? | In JSON yes, Markdown unknown. | low | Explicitly state in Markdown. |
| 22. Boundary count dominate interpretation? | It shows the toy is heavily mixing A and B. Normal for a 6-site toy. | none | None. |
| 23. Stronger warning for boundary jumps? | The warning `boundary_crossing_jumps_observed` is sufficient. | none | None. |
| 24. Interpret ratio cautiously? | Extremely cautiously. | high | Include dt scaling warning. |
| 25. Initial L1 ~0.438 reasonable? | Yes, multinomial sampling noise for $N=200, d=64$. | none | None. |
| 26. Final L1 ~0.346 reasonable? | Yes, falls within expected variance. | none | None. |
| 27. Adequately say descriptive only? | JSON interpretation reason handles this. | none | None. |
| 28. `finite_sample_noise` mandatory? | Yes, to prevent over-reading the 0.46 max L1. | high | Add warning. |
| 29. Report confidence intervals? | Overkill for a toy diagnostic. | none | None. |
| 30. Support sampler sanity? | L1 < 0.5 rules out catastrophic distribution failure. | none | None. |
| 31. `max_lambda_dt` safe? | Yes, ~0.016 is highly safe for Poisson thinning. | none | None. |
| 32. Justify at-most-one-jump? | Yes, $P(>1 \text{ jump}) \approx O(10^{-4})$, negligible here. | none | None. |
| 33. Zero probability count enough? | Yes, proves division by zero was avoided. | none | None. |
| 34. Zero norm failures enough? | Yes, conditional slices remained well-defined. | none | None. |
| 35. Report min/max conditional norms? | Overkill if failures=0. | none | None. |
| 36. Required 0002A limitations present? | Partially. Missing finite toy/noise specifics. | high | Update limitations. |
| 37. Missing limitations? | `finite_size_toy`, `finite_sample_noise`. | high | Add to JSON. |
| 38. "does not construct bridge" outdated? | Yes, it literally constructs a conditional vector toy bridge now. | high | Update text. |
| 39. Explicitly say "toy diagnostic"? | Yes. | high | Use exact wording. |
| 40. Warnings include finite_size by default? | Yes, always for 6 sites. | medium | Add to warnings. |
| 41. Forbidden hits `[]` instead of `null`? | Yes, `null` implies script failure in strict Go JSON. | blocker | Fix JSON serialization to `[]`. |
| 42. Include all necessary fields? | Yes, metrics block is highly detailed. | none | None. |
| 43. JSON and Markdown consistent? | Cannot verify without seeing full markdown. | medium | Verify in source review. |
| 44. Markdown omit fields? | Probably. Ratios without denominators in MD are dangerous. | high | Force Markdown to print denominators. |
| 45. Include input config hash? | Highly recommended for artifact provenance. | low | Add `config_hash`. |
| 46. `needMap` justified? | Yes, conditionally. | none | None. |
| 47. `needInvariant` justified? | Yes. | none | None. |
| 48. `needToyCheck` justified? | Yes. | none | None. |
| 49. Downgrade to `pending_source_review`? | Yes. Debt cannot be permanently paid via JSON payload alone. | blocker | Adjust ledger until source reviewed. |
| 50. `needNullModel: unpaid` correct? | Yes. No-jump is not a true null model for physics. | none | None. |
| 51. `needObstruction` visible? | Yes, string remains clear. | none | None. |
| 52. `source_code_review` sufficient? | It is mandatory before acceptance. | blocker | Require source files. |
| 53. `task.md` enough evidence? | Text claims are easily faked by LLMs. Logs required. | high | Require test logs. |
| 54. `walkthrough.md` evidence? | Same as above. | high | Require CLI output logs. |
| 55. Command outputs included? | Missing from prompt context. | high | Require them. |
| 56. Require `go test` logs? | Yes. | blocker | Upload logs. |
| 57. Source code reviewed? | Mandatory. | blocker | Upload source. |
| 58. Fresh clean checkout run? | Required for EBP CI hygiene. | medium | Execute in clean container. |
| 59. Bridge-disabled artifacts uploaded? | Required to prove regression safety. | high | Upload disabled `report.json`. |
| 60. Second bridge config run? | Useful to prove partition math is general. | medium | Run with different seed/partition. |

---

### Artifact contract verdict

```json
{
  "schema_contract": "breached_by_null_forbidden_array",
  "report_markdown_contract": "unverified_risk_of_hiding_denominators",
  "warnings_contract": "incomplete_missing_dt_scaling_and_finite_noise",
  "limitations_contract": "outdated_conditional_wave_function_string",
  "debt_status_contract": "accurately_requests_source_review",
  "forbidden_language_contract": "breached_by_null_value",
  "validation_evidence_contract": "incomplete_missing_test_logs"
}

```

### EBP 2.1 ledger

```json
{
  "needMap": "attempted_environment_projected_conditional_vector_toy_pending_source_review",
  "needInvariant": "attempted_empirical_trajectory_check_pending_source_review",
  "needToyCheck": "attempted_conditional_vector_toy_pending_source_review",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_review_required_for_0002A_acceptance",
  "promotion_status": "unpromoted_artifact_review_only"
}

```

### Final recommendation

Require source-code review before acceptance.
