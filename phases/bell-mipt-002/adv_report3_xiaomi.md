# Adversarial Review: BELL-MIPT-0002A Implementation Results and Report Artifacts

**Note:** This is an artifact-level review based on the reported metrics and schema provided in the prompt. Source code is not available for inspection. Where the review identifies issues that can only be resolved by inspecting source code, this is stated explicitly.

---

## Executive Verdict

```
accept_with_high_priority_repairs
```

---

## Strongest Evidence for Acceptance

1. **max_lambda_dt ≈ 0.0167 is safely below all thresholds.** The discrete-time thinning approximation is reliable. No multi-jump events are silently dropped.

2. **conditional_norm_failures = 0.** The environment-projected conditional vectors are numerically well-conditioned for all sampled environment configurations across all 200 trajectories. This validates the ConditionalNormFloor threshold for this system size.

3. **near_zero_current_configuration_probability_count = 0.** No trajectory encountered a degenerate current configuration with negligible probability. The rate computation guard was never triggered, meaning no division-by-zero risks materialized.

4. **0001 master-equation equivariance is preserved.** The max_equivariance_l1_error of ~3.13e-11 matches the earlier accepted BELL-MIPT-0001 result, confirming the two-pass architecture did not corrupt the original audit.

5. **Boundary-crossing jumps are separately counted.** The report distinguishes strict_environment_jump_transitions (75), strict_subsystem_jump_transitions (72), and boundary_crossing_jump_transitions (331). This is the correct accounting and shows the implementation tracks transition types as required.

6. **The report schema includes structured warnings with codes.** This enables programmatic filtering rather than relying on string parsing.

7. **The ratio event-count gating appears to work.** The ratio is reported (not null) with 75 environment-jump events and 199,522 no-jump events, both above the minimum threshold of 10.

---

## Critical Blockers

### Blocker 1: Extreme conditional_update_ratio without ratio-instability warning

The reported ratio of ~217,088 is computed as:

```
mean_fidelity_drop_at_strict_environment_jumps / mean_fidelity_drop_no_jump
= 0.8786 / 0.00000405
≈ 217,088
```

This ratio is **mathematically valid** but **diagnostically dangerous**. The denominator of ~4×10⁻⁶ represents fidelity drops from pure Schrödinger phase evolution during no-jump intervals over a single time step (dt = 0.001). Such tiny drops are expected: over one micro-step, the conditional vector barely rotates. The numerator of ~0.879 represents conditional vectors that flip to near-orthogonal states when the environment configuration changes. This magnitude difference is **entirely structural**: environment jumps change which branch of the wave function is selected, while no-jump intervals produce only smooth phase drift. A ratio of 217,088 does not indicate "measurement-like behavior" — it indicates that jumping to a different environment configuration and reading a different slice of the wave function produces a large vector change, which is trivially expected.

**The report must include a warning explaining this structural ratio amplification.** Without it, a reader encountering `conditional_update_ratio: 217088` and `candidate_correlation` in the same report will naturally interpret this as a strong signal, which it is not.

---

## High-Priority Repairs

### H1: Add `large_ratio_warning` to warnings

The warnings array must include a structured warning such as:

```json
{
  "code": "large_ratio_warning",
  "message": "conditional_update_ratio ≈ 217088 is extremely large. This reflects structural amplification: environment jumps select different wave-function branches (producing large vector changes) while no-jump intervals produce only smooth phase drift (producing tiny fidelity drops). The ratio magnitude is not evidence of measurement-like behavior."
}
```

This is not optional. The ratio is 5+ orders of magnitude above what a reader would consider a modest effect, and without explanation it will be misinterpreted.

### H2: Add `ratio_denominator_small` warning

The denominator mean_fidelity_drop_no_jump ≈ 4×10⁻⁶ is extremely small. The report should include:

```json
{
  "code": "ratio_denominator_small",
  "message": "mean_fidelity_drop_no_jump ≈ 4.0e-6 is near the numerical precision floor for single-step conditional-vector evolution. The conditional_update_ratio is dominated by denominator smallness, not by numerator largeness."
}
```

### H3: Add `finite_sample_noise` warning

The empirical trajectory equivariance L1 values (0.438 initial, 0.346 final, 0.462 max) are large in absolute terms. For 200 trajectories over 64 states, the expected multinomial L1 noise is approximately:

```
E[L1] ≈ Σ_i sqrt(2 p_i (1-p_i) / N) ≈ sqrt(2/π) * sqrt((2^d - 1)/N)
```

For d=6 (64 states) and N=200: E[L1] ≈ 0.45, which matches the observed values. This means the empirical equivariance is **dominated by finite-sample noise** and provides essentially no diagnostic value beyond confirming the sampler does not produce a catastrophically wrong distribution. The warning is mandatory:

```json
{
  "code": "finite_sample_noise",
  "message": "Empirical trajectory equivariance L1 ≈ 0.35–0.46 is consistent with multinomial sampling noise for 200 trajectories over 64 states. This diagnostic provides no meaningful signal."
}
```

### H4: Add `finite_size_toy` warning

With 6 sites (Hilbert dimension 64), the system is small enough that finite-size effects dominate any physical interpretation. The conditional vectors live in a space of dimension at most 2³ = 8, which means a single environment jump can easily produce an O(1) fidelity drop simply by selecting a different 8-dimensional slice of the wave function. A warning is mandatory:

```json
{
  "code": "finite_size_toy",
  "message": "Hilbert dimension is 64 with subsystem dimension 8. At this scale, conditional-vector changes are dominated by finite-size effects and do not generalize to larger systems."
}
```

### H5: Downgrade `candidate_correlation` to `weak_correlation`

The term "candidate_correlation" implies the diagnostic detected something that warrants further investigation as a potential real effect. Given:

- The ratio is structurally amplified (not a signal)
- No null model exists to establish a baseline
- The system has only 64 basis states and 8-dimensional subsystem
- Boundary-crossing jumps (331) far outnumber strict environment jumps (75), and boundary-crossing jumps directly modify the subsystem configuration

There is no basis for "candidate" status. The implementation successfully ran and produced metrics, but the metrics do not indicate a correlation beyond what is structurally expected. Downgrade to `weak_correlation` or, more honestly, `no_clear_correlation`. The report can note that environment jumps produce larger fidelity drops than no-jump intervals, but this is expected from the construction, not a discovery.

### H6: Add warning for boundary-crossing dominance

The boundary_crossing_jump_transitions count of 331 is 4.4× the strict_environment_jump_transitions count of 75. Boundary-crossing jumps change both Q_A and Q_B simultaneously, meaning the conditional vector is reconstructed from a completely different environment branch **and** the subsystem state has changed. This makes fidelity drops from boundary-crossing jumps uninformative about the environment-only effect. A warning is needed:

```json
{
  "code": "boundary_crossing_jumps_dominate",
  "message": "Boundary-crossing jumps (331) far outnumber strict environment jumps (75). Boundary-crossing jumps change both subsystem and environment configurations, making their conditional-vector fidelity drops uninformative about the environment-only effect."
}
```

The existing `boundary_crossing_jumps_observed` warning is insufficient — it notes the existence but not the dominance.

### H7: Fix `forbidden_language_audit.hits` from `null` to `[]`

If the forbidden language audit found no hits, the field should be an empty JSON array `[]`, not `null`. Using `null` for "no results found" is an artifact-hygiene issue that can break downstream consumers that expect an array type. Additionally, verify that the audit actually scans the full report JSON and Markdown, not just individual fields.

---

## Medium-Priority Repairs

### M1: Report numerator and denominator values explicitly

The report should include the actual mean fidelity drop values alongside the ratio, not just the ratio. The current report likely includes them in separate metric fields, but the bridge section should make the computation transparent:

```json
"conditional_update_ratio_numerator": 0.87857,
"conditional_update_ratio_denominator": 0.00000405
```

This prevents the ratio from being read in isolation.

### M2: Add `mean_fidelity_drop_at_boundary_crossing_jumps` metric

The report tracks fidelity drops at strict environment jumps and at no-jump transitions, but does not appear to report fidelity drops at boundary-crossing jumps separately. Given that boundary-crossing jumps dominate the count, this metric is essential for interpretation. If a reader wants to understand whether "environment-correlated" updates are driven by boundary-crossing or strict environment jumps, they need both numbers.

### M3: Report `mean_fidelity_drop_at_any_jumps`

The report schema includes this field but it should be populated. Comparing "any jump" fidelity drops against "no jump" fidelity drops provides a simpler, less ambiguous diagnostic than the environment/subsystem/boundary decomposition.

### M4: Verify bridge-disabled output exists and matches 0001

The prompt states this is a concern (question 15: "Are the bridge-disabled semantics actually verified, or only claimed in walkthrough/task files?"). The review should require:
- An uploaded `report_disabled.json` artifact from a bridge-disabled run
- A semantic-field comparison against the accepted 0001 report

Without this artifact, backward compatibility is claimed but not demonstrated.

### M5: Include transition-count breakdown in Markdown

The Markdown report should include the full transition-count breakdown:

```markdown
### Transition Counts
- Strict environment jumps: 75
- Strict subsystem jumps: 72
- Boundary-crossing jumps: 331
- No-jump transitions: 199,522
- Any-jump transitions: 478
```

This makes the accounting auditable from the human-readable report alone.

### M6: Report whether conditional_update_ratio denominator is "no_jump" or "without_environment_jumps"

The ratio uses `mean_fidelity_drop_no_jump` as denominator, not `mean_fidelity_drop_without_environment_jumps`. The report should make this denominator choice explicit in a field like:

```json
"conditional_update_ratio_denominator_category": "no_jump"
```

This prevents confusion between two similar-but-different comparison baselines.

### M7: Clarify limitations text regarding conditional-wave-function bridge

The prompt raises concern that the limitations might still say "does not construct a conditional-wave-function bridge" when the implementation does construct environment-projected conditional vectors. The limitations should say:

```text
This constructs only an environment-projected conditional-vector toy diagnostic. This is not a conditional-wave-function bridge in the Bell/Bohmian sense.
```

This is more precise than either "does not construct" (false — it does construct something) or "constructs a bridge" (overclaim).

### M8: Validate that no-jump fidelity drops scale with dt

The tiny no-jump fidelity drop of ~4×10⁻⁶ per step is consistent with dt = 0.001 and a smooth Schrödinger evolution producing fidelity drops of order `(energy_spread × dt)²`. The implementation should verify this scaling: if dt were 10× larger, the no-jump drop should be ~100× larger (quadratic in dt for small drops). If this scaling does not hold, the no-jump fidelity computation may have a bug.

---

## Low-Priority Polish

### L1: Use `[]` consistently for empty arrays in JSON

All JSON fields that represent lists should use `[]` for empty lists, not `null`. This applies to `forbidden_language_audit.hits`, `warnings` if empty, etc.

### L2: Include schema_version comparison field

Add a field that explicitly states whether the bridge-disabled run would produce `bell_mipt_report_v0`:

```json
"bridge_disabled_schema_version": "bell_mipt_report_v0"
```

This makes backward-compatibility claims machine-verifiable.

### L3: Report empirical equivariance confidence intervals

Even though empirical equivariance is noise-dominated, reporting a rough multinomial confidence interval helps readers calibrate:

```json
"empirical_equivariance_expected_l1_noise": 0.45,
"empirical_equivariance_observed_l1": 0.44
```

This shows the observed value is within the noise envelope.

### L4: Report subsystem and environment dimensions explicitly

```json
"subsystem_hilbert_dim": 8,
"environment_hilbert_dim": 8
```

This is derivable from site counts but makes the report self-contained.

### L5: Include input config hash for provenance

A SHA-256 hash of the input JSON enables artifact deduplication and provenance tracking.

---

## Specific Answer Table

### A. Baseline 0001 Preservation

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 1 | Does report preserve original 0001 goal and metrics? | Yes. `goal_status: toy_goal_passed` and `max_equivariance_l1_error: ~3.13e-11` match the accepted 0001 result. The two-pass architecture appears to preserve the original audit. | none | — |
| 2 | Is goal_status tied only to master-equation equivariance? | Yes, from the reported schema. `goal_status` is independent of `bridge_status`. | none | — |
| 3 | Do 0001 metrics match earlier accepted run? | Yes. 3.13e-11 matches the previously reported ~3.13e-11. | none | — |
| 4 | Does schema_version bell_mipt_report_v0_2a make sense for bridge-enabled run? | Yes. The bridge section is a nontrivial schema addition that warrants a version bump. | none | — |
| 5 | Should bridge-disabled remain v0 with no bridge section? | Yes. This is the correct backward-compatibility behavior. Verify with an actual bridge-disabled artifact. | **medium** | Upload bridge-disabled artifact for verification (see M4). |

### B. Bridge Audit Status

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 6 | Is bridge_audit_completed justified by metrics? | Conditionally yes. The implementation ran to completion without numerical failures (norm_failures=0, max_lambda_dt safe, no near-zero probabilities). "Completed" means the audit finished successfully, not that a physics result was found. However, the extreme ratio and missing warnings undermine the completeness of the audit. | **medium** | Add missing warnings before claiming completion. |
| 7 | Should bridge_audit_completed require source-code review? | No. The status reflects implementation-level completion (did the audit run and produce valid outputs). Source-code review is a separate concern captured in needFaithfulnessReview. | none | — |
| 8 | Should 331 boundary-crossing jumps weaken status? | No. Boundary-crossing jumps are valid jump events that the sampler correctly identified. The count is a metric, not a status qualifier. However, a warning about dominance is needed. | **medium** | Add boundary-crossing dominance warning (see H6). |
| 9 | Should no norm failures and low max_lambda_dt suffice for completion? | Yes, combined with successful trajectory sampling and metric emission. These are the correct implementation-level success criteria. | none | — |
| 10 | Should empirical L1 0.35–0.46 make audit inconclusive? | No. The empirical equivariance is explicitly a descriptive diagnostic, not a pass/fail criterion. The high L1 is expected from multinomial noise and does not indicate implementation failure. However, a warning about noise is mandatory. | **medium** | Add finite_sample_noise warning (see H3). |

### C. Conditional Update Ratio

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 11 | Is ratio ~217088 mathematically valid? | Yes. 0.879 / 4.05e-6 ≈ 217,037. The arithmetic is correct given the reported numerator and denominator. | none | — |
| 12 | Is ratio meaningful given small denominator? | No, not as a "signal." The denominator ~4e-6 represents single-step Schrödinger phase drift, which is structurally tiny. The ratio magnitude reflects denominator smallness, not numerator significance. | **high** | Add large_ratio_warning and ratio_denominator_small warning (H1, H2). |
| 13 | Should report include warning for extreme ratio amplification? | Yes. This is the most important missing element in the report. | **blocker** | Add large_ratio_warning (H1). |
| 14 | Should ratio be capped, log-scaled, or reported with components only? | Do not cap or log-scale — this would hide the actual computation. Instead, report the ratio as-is but with explicit numerator, denominator, and a warning explaining the amplification. Transparency is better than obfuscation. | none | Report numerator/denominator explicitly (M1). |
| 15 | Should candidate_correlation be downgraded? | Yes. Without a null model, and with structural amplification fully explaining the ratio, "candidate" overstates. Downgrade to `weak_correlation` or `no_clear_correlation`. | **high** | Downgrade to weak_correlation (H5). |
| 16 | Should candidate_correlation require null-model comparison? | Yes, for "candidate" status. Without a null, "no_clear_correlation" is the appropriate classification. | **high** | Downgrade (H5). |
| 17 | Does ratio accidentally suggest measurement behavior? | Yes, if read without the warnings. A ratio of 217,088 with "candidate_correlation" label strongly suggests measurement-like backaction. The warnings must prevent this reading. | **high** | Add structural-amplification warning (H1). |
| 18 | Is candidate_correlation safe under EBP 2.1? | No, given the current evidence. The label "candidate" implies the diagnostic detected something that could be a real effect. Without a null model and with structural amplification, this is not justified. | **high** | Downgrade (H5). |

### D. Boundary-Crossing Jumps

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 19 | Are boundary-crossing jumps properly separated? | Yes. The report counts strict_environment (75), strict_subsystem (72), and boundary_crossing (331) separately. | none | — |
| 20 | Are boundary-crossing jumps excluded from primary ratio? | Yes. The ratio uses strict_environment_jumps (75 events) vs no_jump (199,522 events). | none | — |
| 21 | Is exclusion clear enough in report? | Partially. The ratio status field mentions event counts, but a human reader may not immediately see why boundary-crossing jumps (the majority) are excluded. The report should explicitly note this. | **medium** | Add note about boundary-crossing exclusion to ratio section. |
| 22 | Should large boundary-crossing count dominate interpretation? | Not dominate, but it should be prominently noted. The 331 boundary-crossing transitions represent the most common jump type, and their conditional-vector fidelity drops are uninformative about the environment-only effect because they change the subsystem too. | **medium** | Add boundary-crossing dominance warning (H6). |
| 23 | Should there be a stronger boundary-crossing warning? | Yes. The existing `boundary_crossing_jumps_observed` warning merely notes existence. A dominance warning is needed. | **high** | Add boundary_crossing_jumps_dominate warning (H6). |
| 24 | Should strict environment ratios be interpreted cautiously given boundary-crossing frequency? | Yes. Strict environment jumps are only 75 out of 478 total jumps (15.7%). The conditional_update_ratio is based on a small, potentially unrepresentative subset of jumps. | **medium** | Note in report that strict environment jumps are a minority of all jumps. |

### E. Empirical Trajectory Equivariance

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 25 | Is initial L1 ≈ 0.438 reasonable? | Yes. For 200 samples over 64 states with uniform distribution, expected L1 ≈ 0.45. For a non-uniform |ψ₀|² distribution, the value would be somewhat different but in the same range. The observed 0.438 is consistent with multinomial noise. | none | — |
| 26 | Is final L1 ≈ 0.346 reasonable? | Yes. Slightly lower than initial, possibly because the wave function concentrates on fewer states at later times, reducing the effective number of populated states and thus the noise. | none | — |
| 27 | Does report adequately describe equivariance as descriptive? | The report says empirical trajectory equivariance is a "descriptive diagnostic" in the limitations, but the metric values themselves (0.35–0.46) will be read by most users without this context. The warning about finite-sample noise is essential. | **medium** | Add finite_sample_noise warning (H3). |
| 28 | Should finite_sample_noise warning be mandatory? | Yes. | **high** | Add (H3). |
| 29 | Should confidence intervals be reported? | Not strictly necessary for acceptance, but helpful. An expected-L1-noise value provides context. | **low** | Add expected noise estimate (L3). |
| 30 | Does equivariance support sampler sanity? | Marginally. The observed L1 is consistent with multinomial noise, which means the sampler is not catastrophically wrong (e.g., always returning the same state). But it cannot detect subtle distributional biases. It is a basic sanity check, not a strong validation. | none | — |

### F. Numerical Reliability

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 31 | Is max_lambda_dt ≈ 0.0167 safely below threshold? | Yes. Well below the 0.1 warning threshold and far below the 0.5 inconclusive threshold. | none | — |
| 32 | Does low max_lambda_dt justify at-most-one-jump? | Yes. At λdt ≈ 0.017, the multi-jump probability is ~0.014%, negligible. | none | — |
| 33 | Is near_zero count = 0 enough to clear division risk? | Yes, for this run. It confirms no trajectory encountered a configuration with |ψ_q|² below the floor. | none | — |
| 34 | Are zero norm failures enough to trust fidelity? | Yes, for this run. All conditional vectors had norm above the floor, meaning the normalized vectors are numerically reliable. | none | — |
| 35 | Should report include min/max conditional norms? | Helpful but not required. Would provide additional numerical confidence. | **low** | Add min/max norm diagnostics. |

### G. Report/Schema/Limitations

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 36 | Are required 0002A limitations present? | Cannot fully verify without seeing the actual report.md content. The reported JSON does not include a limitations field. | **medium** | Verify all 9 required limitation sentences are present in both JSON and Markdown. |
| 37 | Which limitations are missing? | Cannot determine without seeing the actual limitations list. At minimum, the "finite-size effects may dominate" and "finite-sample noise may dominate" limitations must be present. | **medium** | Verify. |
| 38 | Is "does not construct a conditional-wave-function bridge" outdated? | Yes. The implementation constructs environment-projected conditional vectors. The limitation should say "constructs only an environment-projected conditional-vector toy diagnostic, not a conditional-wave-function bridge in the Bell/Bohmian sense." | **high** | Update limitation text (M7). |
| 39 | Should limitations explicitly say "environment-projected conditional-vector toy diagnostic"? | Yes. | **high** | Update (M7). |
| 40 | Should warnings include finite_size_toy and finite_sample_noise by default? | Yes, for any bridge-enabled run on a system with dim ≤ 256 and trajectories ≤ 1000. | **high** | Add both warnings (H3, H4). |
| 41 | Should forbidden_language_audit.hits be [] not null? | Yes. | **low** | Use [] for empty arrays (H7). |
| 42 | Does report include all bridge fields? | The reported JSON includes the key fields. Cannot verify completeness without seeing the full schema. | **low** | — |
| 43 | Are JSON and Markdown consistent? | Cannot verify without seeing report.md. | **medium** | Verify consistency. |
| 44 | Does Markdown omit important fields? | Cannot verify. The transition counts (strict/subsystem/boundary/no-jump) should be in Markdown. | **medium** | Ensure transition counts in Markdown (M5). |
| 45 | Should report include config hash or provenance? | Helpful but not required for acceptance. | **low** | Add (L5). |

### H. EBP Debt Status

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 46 | Is needMap partially_paid justified? | Yes, with the caveat that source code review has not occurred. The environment-projected conditional vector computation was implemented and ran without errors. | **medium** | Add "pending source review" qualifier. |
| 47 | Is needInvariant partially_paid justified? | Yes, similarly qualified. | **medium** | Add qualifier. |
| 48 | Is needToyCheck partially_paid justified? | Yes. The rate algebra was already partially validated in 0001, and the conditional vector toy ran successfully. | **medium** | Add qualifier. |
| 49 | Should any be downgraded to attempted_*_pending_source_review? | Yes, all three should include the pending-source-review qualifier until source code is actually reviewed. The artifact review alone cannot verify correctness of the implementation. | **high** | Downgrade all to include pending source review qualifier. |
| 50 | Is needNullModel: unpaid correct? | Yes. No null model was implemented or compared against. This is the most significant remaining gap. | none | — |
| 51 | Is needObstruction preserved? | Yes. `bell_jumps_are_not_measurements` remains. | none | — |
| 52 | Is needFaithfulnessReview sufficient? | Yes, as stated: `source_code_review_required_for_0002A`. This correctly indicates that artifact review alone is insufficient. | none | — |

### I. Validation Evidence

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 53 | Does task.md provide test evidence? | Cannot verify without seeing task.md content. | **medium** | Require inclusion of test output logs. |
| 54 | Does walkthrough.md provide validation evidence? | Cannot verify without seeing walkthrough.md content. | **medium** | Require inclusion of validation command outputs. |
| 55 | Are actual command outputs included? | Cannot verify. | **medium** | Require. |
| 55 | Are actual command outputs included? | Cannot verify. | **medium** | Require. |
| 56 | Should review require go test/vet/race logs? | Yes. Source-code-level validation cannot be inferred from artifact-level outputs alone. | **high** | Require test logs. |
| 57 | Should source code be reviewed? | Yes, before full acceptance. The artifact review can confirm the report schema and numerical values, but cannot verify that the code implements what it claims. | **high** | Require source-code review. |
| 58 | Should a fresh clean checkout run be required? | Yes, to confirm reproducibility. | **medium** | Require. |
| 59 | Should bridge-disabled artifacts be uploaded? | Yes, to verify backward compatibility empirically. | **medium** | Upload bridge-disabled report (M4). |
| 60 | Should a second bridge config be run? | Helpful but not required for acceptance. A different partition (e.g., A=[0,1], B=[2,3,4,5]) and seed would test generality. | **low** | Optional. |

---

## Artifact Contract Verdict

```json
{
  "schema_contract": "acceptable_with_repairs",
  "report_markdown_contract": "cannot_fully_verify_missing_report_md_content",
  "warnings_contract": "incomplete_missing_4_required_warnings",
  "limitations_contract": "needs_update_conditional_wave_function_text",
  "debt_status_contract": "needs_source_review_qualifier",
  "forbidden_language_contract": "acceptable_hits_null_needs_fix",
  "validation_evidence_contract": "insufficient_no_logs_provided"
}
```

---

## EBP 2.1 Ledger

```json
{
  "needMap": "attempted_environment_projected_conditional_vector_toy_pending_source_review",
  "needInvariant": "0001_partially_paid; 0002A_empirical_trajectory_diagnostic_completed_pending_source_review",
  "needToyCheck": "0002A_artifact_reviewed_pending_source_review",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_review_required_for_0002A",
  "promotion_status": "unpromoted_artifact_review_only"
}
```

---

## Final Recommendation

```
Accept after minor artifact repairs.
```

The implementation appears to be technically correct at the artifact level: the two-pass architecture preserves 0001 results, the trajectory sampler runs without numerical failures, the conditional vectors are well-conditioned, and the report schema is properly structured. However, four repairs must be made to the report artifacts before acceptance:

1. **Add `large_ratio_warning` and `ratio_denominator_small` warnings.** The conditional_update_ratio of ~217,088 is the single most dangerous number in the report. Without explicit explanation that this reflects structural amplification (environment jumps select different wave-function branches, producing O(1) fidelity drops, while no-jump intervals produce only O(dt²) phase drift), it will be misinterpreted as evidence of measurement-like behavior. This is the highest-priority repair.

2. **Downgrade `candidate_correlation` to `weak_correlation` or `no_clear_correlation`.** Without a null model, and with the ratio fully explained by structural amplification, "candidate" is not justified. The terminology must not invite physics promotion.

3. **Add `finite_sample_noise`, `finite_size_toy`, and `boundary_crossing_jumps_dominate` warnings.** These three warnings are mandatory for honest reporting. The empirical equivariance is noise-dominated, the system is too small for physical generalization, and boundary-crossing jumps outnumber the strict environment jumps that drive the ratio.

4. **Update limitations to reflect that environment-projected conditional vectors were constructed.** The limitations must not say "does not construct" when it does — they must say "constructs only an environment-projected conditional-vector toy diagnostic, not a conditional-wave-function bridge."

After these four repairs to the report artifacts, the result should be accepted for limited toy scope with the understanding that source-code review is still required before any debt status can fully advance.
