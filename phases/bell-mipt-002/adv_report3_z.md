To move from artifact-level review to a proper source-code review for BELL-MIPT-0002A, I would need the following files, prioritized by what each one would resolve from the high-priority repairs list. The previous review identified `accept_with_high_priority_repairs` with a final recommendation of "Require source-code review before acceptance," and these are the specific files that would close that gap.

## Prioritized file inventory

| Priority | File / artifact | Review purpose | Maps to repair |
|---|---|---|---|
| P0 | `bell_sampler.go` (or equivalent trajectory sampler source) | Verify thinning logic, at-most-one-jump-per-step, `max_lambda_dt` enforcement | Q31–33, numerical reliability |
| P0 | `conditional_vector.go` / `environment_projection.go` | Verify environment-projected conditional vector construction, norm renormalization, division-by-zero guards | Q34–35, limitations wording (Q38–39) |
| P0 | `jump_classifier.go` (strict env / strict subsystem / boundary crossing) | Confirm 331 boundary crossings are genuinely excluded from the primary ratio, and that the partition convention matches Jordan–Wigner | Q19–24, warnings |
| P0 | `metrics.go` (ratio, fidelity drop, equivariance) | Inspect the `conditional_update_ratio ≈ 217088` computation, denominator (`≈4e-6`) handling, and whether raw counts are recoverable | Q11–17, ratio warnings |
| P0 | `equivariance_audit.go` (analytical + empirical) | Reconcile the 12-order-of-magnitude gap between analytical L1 (`~3e-11`) and empirical L1 (`~0.35–0.46`); confirm empirical is multinomial-sampling noise, not a bug | Q25–30 |
| P1 | `report_schema.go` / JSON+MD emitter | Confirm `bell_mipt_report_v0_2a` schema preserves 0001 fields, emits all bridge fields, keeps `hits` as `[]` not `null` | Q41–44, schema contract |
| P1 | `warnings.go` / `limitations.go` | Check that warning set is exhaustive (`large_ratio_warning`, `ratio_denominator_small`, `finite_sample_noise`, `finite_size_toy`) and that limitations text is updated for 0002A | Q36–40, warnings/limitations contracts |
| P1 | `debt_ledger.go` (EBP 2.1 status emission) | Verify `needMap`/`needInvariant`/`needToyCheck` are not auto-advanced past `pending_source_review` | Q46–52, debt contract |
| P1 | `forbidden_language_audit.go` | Confirm the audit actually scans output, not just returns `[]`/`null` | forbidden_language_contract |
| P2 | `*_test.go` (unit tests for sampler, classifier, ratio) | Confirm tests exist for boundary-crossing exclusion, denominator-small regime, and equivariance invariants | Q53–57 |
| P2 | `go.mod`, `Makefile` / CI config, `task.md` command list | Confirm the exact `go test ./...`, `go test -race ./...`, `go vet ./...` invocations and that they ran clean | Q54–56 |
| P2 | `input.json` + config-hash generation code | Verify provenance hash is computed over the full input, not a subset | Q45 |
| P3 | `walkthrough.md` with embedded command outputs | Need actual stdout/stderr, not narrative | Q54–55 |
| P3 | Bridge-disabled run artifacts (parallel `report.json` under `bell_mipt_report_v0`) | Baseline comparison to confirm 0001 fields are untouched when bridge is off | Q5, Q59 |
| P3 | Second bridge-config run (different partition/seed) | Robustness check on the ratio and jump classification | Q60 |

## Review modules by purpose

**Sampler correctness (P0).** The single most important file is whatever implements the Bell-rate thinning sampler. I want to see: (a) the discrete-time step loop, (b) how `lambda_dt` is computed and capped, (c) that the at-most-one-jump-per-step assumption is enforced in code rather than assumed, and (d) how trajectories are counted toward `mean_jump_count = 2.39`. Without this, the `max_lambda_dt ≈ 0.0167` claim is unverifiable.

**Jump classification (P0).** This is where the 331 boundary-crossing transitions live. I need to see the exact predicate that distinguishes `strict_environment_jump` (75) from `boundary_crossing_jump` (331) from `strict_subsystem_jump` (72). If the predicate is off by one site or uses an inconsistent partition convention, the entire ratio interpretation collapses. This directly addresses Q19–24, the strongest open concern.

**Ratio and denominator (P0).** The `conditional_update_ratio ≈ 217088` with a denominator of `≈4e-6` is the most overclaim-prone number in the report. I want to see the exact division, whether there is any epsilon floor, whether boundary crossings leak into the numerator, and whether raw counts (75 vs 199522) are stored alongside the ratio. This is the difference between `candidate_correlation` and `candidate_correlation_unstable_ratio`.

**Schema and warnings emission (P1).** I want to see the struct definitions and the emitter to confirm `forbidden_language_audit.hits` is typed as `[]string` (not `*[]string`), that the 0001 fields are not conditionally omitted, and that the warning set is generated from a checklist rather than hardcoded.

**Test and validation evidence (P2).** The previous review flagged Q53–57 as high severity — task.md and walkthrough.md currently do not contain actual command outputs. I'd want the test files themselves plus the CI/Makefile targets, and ideally a pasted log of a clean `go test -race ./...` run.

## Minimal sufficient package for a binding verdict

If only a subset can be shared, the four files that would let me issue a binding verdict rather than `inconclusive_needs_source_review` are:

1. `bell_sampler.go` (thinning + jump loop)
2. `jump_classifier.go` (strict/boundary partitioning)
3. `metrics.go` (ratio + fidelity-drop + equivariance)
4. `*_test.go` covering the classifier and the denominator-small regime

Everything else (schema, warnings, debt, walkthrough logs) is P1–P3 polish that affects the artifact contract but not the core numerical honesty question. If you can share the repository path or paste those four files, the review can move from artifact-level to source-level and the `pending_source_review` qualifiers on the EBP ledger can be resolved.
