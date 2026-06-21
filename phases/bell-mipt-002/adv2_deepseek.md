## Executive Verdict

**ready_for_implementation_with_minor_repairs**

The repaired plan is substantially improved: time‑step alignment is explicit, rate orientation is locked, partition/Jordan‑Wigner handling is clear, and the terminology avoids MIPT/measurement overclaim. The event‑count, norm‑failure, and `max_lambda_dt` thresholds provide mechanical safety. Only two minor issues remain that should be fixed before coding to avoid ambiguity and scope creep.

---

## Strongest Repaired Decisions

- **Explicit time‑step convention** – samples jumps using `ψ[k]` *before* the interval, avoids off‑by‑one confusion.
- **Locked rate orientation** `rates[dest][src]` – testable and unambiguous.
- **Clear partition/projection rule** – no extra Jordan‑Wigner sign, canonical sorting, and mandatory `Combine(Split(q)) == q` invariant.
- **Smart ratio/null logic** – suppresses `conditional_update_ratio` when event counts are insufficient, preventing spurious signals.
- **Softened terminology** – `environment_correlated_conditional_update` is far safer than `monitoring_like_signal`.
- **Debt‑status rule** – only advances if `bridge_audit_completed`, preventing overstatement.
- **Comprehensive warning types** – makes numerical limitations visible.

---

## Remaining Blockers

**None.** The plan is safe to implement with the minor edits below.

---

## High‑Priority Repairs Before Coding

1. **Clarify the denominator for `conditional_update_ratio`.**  
   The plan says `mean_fidelity_drop_without_environment_jumps`, but the report schema includes both that and `mean_fidelity_drop_no_jump`. The definition should be unambiguous:
   > `conditional_update_ratio = mean_fidelity_drop_at_environment_jumps / mean_fidelity_drop_without_environment_jumps`  
   where `without_environment_jumps` includes **both** no‑jump and subsystem‑only jumps. This is the most meaningful null because it isolates the presence of an environment change.  
   **Add explicit note** that denominator excludes transitions where the environment changed (even if the subsystem also changed).

2. **Remove or demote `fixed_environment_reference_mean_drop`.**  
   This is not requested in the original requirements and adds complexity without a clear role in the primary comparison. It can be added later if needed. For `0002A`, keep the scope minimal.

3. **Specify that bridge audit uses the *same* ψ snapshots as Pass 1.**  
   The plan says "reuse ψ snapshots" but should explicitly state that the bridge does **not** re‑run the RK4 evolution. This prevents accidental divergence between the master‑equation audit and the trajectory sampling.

4. **Add explicit rule for sampling at `t=0`.**  
   The plan says "sample Q₀ from |ψ₀|²" and "conditional vector at sample k uses (ψ[k], Q_B[k])". It should clarify whether the initial conditional vector is sampled and stored for later fidelity comparisons (it must be, to compute drops from the first step).  
   **Add** a sentence: "The first conditional vector is stored at k=0. Fidelity drops are computed from k=0→1 onward."

---

## Medium‑Priority Repairs (non‑blocking)

- **Test for `ψ_q` near zero** – the plan mentions a warning but doesn't specify a threshold. Add: if `|ψ_current_config|² < 1e‑12`, emit `near_zero_current_configuration_probability` warning.
- **Add a deterministic fake‑rates test** that forces exactly one environment jump in a small 2‑site system to verify the classification logic.
- **Clarify that `sample_every_steps` refers to full time steps, not RK4 sub‑steps** – this is already implied by the time‑step convention, but a note helps.

---

## Low‑Priority Polish

- **Order of site lists in report** – `requested` vs `canonical` is good; use `subsystem_sites_canonical` in the actual computation.
- **Schema version bump** – `v0_2a` is fine; consider adding `schema_version` as a top‑level field only if bridge enabled.
- **Bridge section omitted when disabled** – OK for backward compatibility; a short `"bridge": null` is also fine.

---

## Answer Table

| Q | Answer | Severity | Repair |
|---|--------|----------|--------|
| 1 | No conceptual blockers remain. | none | – |
| 2 | Safer – explicitly focuses on projection from full state. | none | – |
| 3 | Acceptable; it describes correlation, not monitoring. | low | – |
| 4 | Risk is minimal; explicit limitations and `bell_jumps_are_not_measurements` cover it. | low | – |
| 5 | No – debt only advances on `bridge_audit_completed`. | none | – |
| 6 | Correct and implementable. | none | – |
| 7 | Proposed convention (jump uses ψ[k]) is correct; ψ[k+1] is pre‑computed. | none | – |
| 8 | Yes – RK4 sub‑stages are internal; full‑step sampling is sufficient for this toy. | low | – |
| 9 | Yes – explicit `rates[dest][src]` is unambiguous. | none | – |
|10 | Yes – mandatory tests catch orientation. | none | – |
|11 | Yes – occupation basis avoids fermionic sign; all phases in amplitudes. | none | – |
|12 | Yes – `Combine(Split(q)) == q` and non‑contiguous tests are sufficient. | none | – |
|13| Sorting internally is safe and deterministic; user order is cosmetic. | low | – |
|14| Yes – report both requested and canonical. | none | – |
|15| Not necessary; bitmask operations with site‑list mapping are fine for toy L≤6. | low | – |
|16| Reasonable; 0.1/0.5 thresholds are standard for discrete thinning. | low | – |
|17| Warning is fine; making it inconclusive at 0.1 would be too strict. | low | – |
|18| 10%/50% thresholds are sensible. | low | – |
|19| 1e‑14 is acceptable for double precision. | low | – |
|20| Warn and record `near_zero_current_configuration_probability`; no special handling needed. | medium | Add threshold. |
|21| Fidelity is sufficient for this diagnostic; trace distance could be added later. | low | – |
|22| Scope creep; remove from required metrics. | high | Remove or demote. |
|23| Yes – min 10 events each and denominator floor are sufficient. | none | – |
|24| Denominator should be `mean_fidelity_drop_without_environment_jumps` (includes no‑jump and subsystem‑only). | high | Explicitly state. |
|25| No – environment jump means `Q_B` changed; boundary changes still count as environment jumps. | none | – |
|26| Omit when disabled for backward compatibility. | low | – |
|27| Yes – version bump only when bridge enabled. | low | – |
|28| Safe names – clear and non‑promotional. | none | – |
|29| Safe – qualifies as “enhancement” not “signal” or “detection”. | low | – |
|30| Yes – warnings and limitations are comprehensive. | none | – |
|31| Yes – the mandatory test list covers dangerous bugs. | none | – |
|32| Missing test for deterministic `ψ_q` near zero with forced warning. | medium | Add threshold test. |
|33| `TestEmpiricalEquivarianceSampledStepsOnly` – may be redundant if `sample_every_steps` is tested. | low | Optional. |
|34| Use fake RNG for deterministic unit tests; stochastic tests can use fixed seed for integration. | low | – |
|35| Semantic comparison is more robust; byte‑for‑byte can break on non‑deterministic JSON ordering. | low | Use semantic. |

---

## Final Repaired Implementation Checklist

For the coding agent:

- [ ] **Site‑split bijection** implemented with `SplitConfig(q)` → `(aIdx, bIdx)` and `CombineConfig(aIdx, bIdx)` → `q`; add invariant test for all `q`.
- [ ] **Time‑step order** exactly as specified: use `ψ[k]` for rates to sample `Q[k+1]`; use `ψ[k]` and `Q_B[k]` for conditional vector at step `k`.
- [ ] **Rate orientation** `rates[dest][src]`; add a unit test with a fake rate matrix to verify correct selection.
- [ ] **Conditional vector construction**:
   - No extra Jordan‑Wigner signs.
   - Normalize iff norm² > 1e‑14; else record norm failure.
   - Norm‑failure counters per trajectory and total.
- [ ] **Fidelity drop classification**:
   - Define environment jump as `Q_B` changed from previous sampled step.
   - Compute `drop = 1 - |<prev_cond|curr_cond>|²`.
   - Aggregate into four categories: env jump, no env jump, any jump, no jump.
- [ ] **`conditional_update_ratio`**:
   - Only computed if `env_jump_transitions ≥ 10`, `non_env_jump_transitions ≥ 10`, and `mean_drop_non_env > 1e-14`.
   - Otherwise set to `null` with status `"unavailable_insufficient_events"`.
- [ ] **`max_lambda_dt`** computed as max over all trajectory micro‑steps; store and include warning/inconclusive logic.
- [ ] **Empirical equivariance** computed at sampled time steps only; store initial, max, final L1.
- [ ] **Warnings** emitted for `large_lambda_dt_warning`, `severe_lambda_dt_inconclusive`, `low_environment_event_count`, `low_non_environment_event_count`, `conditional_norm_failures`, `finite_sample_noise`, `finite_size_toy`, `near_zero_current_configuration_probability`.
- [ ] **Tests** all 15 mandatory tests pass; add threshold test for near‑zero probability.
- [ ] **Forbidden‑language audit** scans JSON and Markdown output.
- [ ] **Bridge‑disabled** regression preserves all `0001` semantic fields.
- [ ] **Debt‑status** update only if `bridge_audit_completed`.

---

## EBP 2.1 Ledger

```json
{
  "needMap": "ready_for_repaired_conditional_vector_toy_attempt",
  "needInvariant": "0001_partially_paid; 0002A_empirical_trajectory_diagnostic_planned",
  "needToyCheck": "0002A_plan_repaired_pending_implementation",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements_preserved",
  "needFaithfulnessReview": "0002A_pending_source_review_after_implementation",
  "promotion_status": "unpromoted_final_plan_review_only"
}
```

---

## Final Recommendation

**Proceed to implementation.** The plan is now specific, mechanically testable, and EBP‑safe. The two high‑priority clarifications (denominator definition and removal of `fixed_environment_reference_mean_drop`) should be applied before coding to keep scope tight, but they do not require another full plan revision.
