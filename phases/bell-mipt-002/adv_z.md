## Executive Verdict

**use_as_base_with_repairs**  
The plan is structurally sound, clearly separates old and new functionality, and includes important diagnostics like `max_lambda_dt`. However, several critical ambiguities and overclaim risks must be resolved before implementation. Once the high‑priority repairs are addressed, the plan can proceed.

---

## Strongest Parts of the Plan

- **Two‑pass architecture** – preserves existing `0001` audit, reuses `ψ(t)` snapshots, and keeps the bridge optional.
- **Clear config extension** – backward‑compatible, with validation for site partition and trajectory parameters.
- **Comprehensive metrics** – includes per‑category fidelity drops, empirical equivariance, and jump counts.
- **Independent `bridge_status`** – decoupled from the original `goal_status`, avoiding conflation.
- **`max_lambda_dt` diagnostic** – provides a numerical guard against invalid discrete‑time sampling.
- **Explicit non‑goals and forbidden‑language audit** – reduces the risk of physics overclaim.

---

## Critical Blockers

1. **Missing specification of site‑ordering and conditional‑state mapping** – The plan says “split q into subQ/envQ” but does not define how the full configuration index is decomposed into subsystem and environment indices, especially for fermionic systems where Jordan‑Wigner order matters. Without a clear bijection, the conditional wavefunction may be incorrect.
2. **Time‑step alignment between ψ evolution and jump sampling is ambiguous** – It is not stated whether jumps are sampled before or after evolving `ψ` to the next step. This affects the rates used (`ψ(t)` vs `ψ(t+dt)`) and must be fixed.
3. **“monitoring_like_signal” is too strong** – Despite the disclaimer, this phrase implies a measurement‑like effect. It should be renamed to avoid any suggestion of MIPT or monitoring.

---

## High‑Priority Repairs (must be resolved before coding)

- **Explicitly define the bijection** between full configuration index and (subsystem index, environment index). Document that the basis ordering is fixed (e.g., all sites ordered, then split by site lists) and include tests that verify `Combine(Split(q)) == q` for all q.
- **Specify the per‑time‑step order**:
  1. At time `t_k`, have `ψ(t_k)` and `q_k`.
  2. Compute Bell rates using `ψ(t_k)`.
  3. Sample jump to `q_{k+1}` (could be same).
  4. Evolve `ψ` from `t_k` to `t_{k+1}` using the existing RK4 (independent of jump).
- **Rename `monitoring_like_signal`** → `conditional_update_correlation` or `environment_jump_correlation`. The possible values should be `not_assessed`, `no_clear_correlation`, `weak_correlation`, `candidate_correlation`.
- **Define quantitative thresholds** for:
  - `ConditionalNormFloor` (already set to 1e‑14, but specify if absolute or relative).
  - Norm‑failure tolerance: e.g., if > 5% of samples fail, set `bridge_toy_inconclusive`; if > 50%, set `bridge_toy_failed`.
  - Minimum transition counts: require at least 10 environment‑jump transitions and 10 non‑environment transitions before computing `conditional_update_ratio`; otherwise report as `null`.
- **Add explicit tests** for rate‑orientation (ensure `rates[destination][current]` is used) and for boundary‑crossing jumps (where both A and B change).

---

## Medium‑Priority Repairs (useful but non‑blocking)

- **Include confidence bounds** for empirical equivariance L1 (e.g., bootstrapped standard deviation) to distinguish noise from systematic deviation.
- **Add a warning in the report** when `max_lambda_dt` exceeds e.g. 0.1, advising that the discrete‑time approximation may be inaccurate.
- **Clarify debt‑status update** – only advance to `partially_paid_*` if `bridge_status == bridge_toy_passed`; otherwise leave debt unchanged.
- **Provide a test case** that forces known environment jumps (e.g., by setting a simple two‑site model) to verify metrics.

---

## Low‑Priority Polish (naming/schema/reporting)

- Consider bumping the report schema version to `bell_mipt_report_v0_2a` to reflect added fields.
- Sort subsystem and environment site lists internally to guarantee a canonical order, and record the sorted order in the report.
- If event counts are low, emit a warning in the `reason` field rather than hiding the ratio.
- Use JSON `null` for unavailable means/ratios, as planned.

---

## Specific Answer Table

| Q | Answer | Severity | Repair |
|---|--------|----------|--------|
| 1 | Conditional wavefunction is valid only if the basis ordering is consistent; fermionic signs are handled by the occupation basis. The plan must explicitly define the mapping. | blocker | Specify bijection and test it. |
| 2 | Pairing terms do not conserve particle number, but the occupation basis still supports conditional states; no issue. | none | – |
| 3 | The term “conditional subsystem state” is appropriate in the Bohmian/Bell context; no additional QFT conditions are needed for this toy. | low | Add a footnote in limitations. |
| 4 | Risk is low due to explicit non‑goals, but the phrase “monitoring_like_signal” could cause confusion. | high | Rename as above. |
| 5 | Fidelity drops can be influenced by phase evolution; the plan compares categories to isolate jump effects, which is reasonable. | low | – |
| 6 | A better null could include random jump shuffling, but for a toy this is sufficient. | low | – |
| 7 | Yes, “monitoring_like_signal” is too strong. | high | Rename. |
| 8 | Boundary‑crossing jumps are correctly counted as environment jumps (since Q_B changed) and also subsystem jumps if Q_A changes. The classification is consistent. | none | – |
| 9 | Empirical equivariance is noisy but useful as a descriptive check; the plan does not use it to gate pass/fail. | low | – |
|10| Finite‑size artifacts are possible; the plan should mention this in limitations. | medium | Add to limitations. |
|11| Discrete‑time thinning with exponential probability is standard for time‑dependent rates when rates vary slowly over dt; max_lambda_dt mitigates errors. | low | – |
|12| max_lambda_dt is a good diagnostic; a stricter rule could be to abort if >0.1, but the plan uses it as warning/inconclusive, which is acceptable. | low | – |
|13| Sampling at most one jump per dt is correct when λdt is small; the plan should abort or warn if max_lambda_dt is too large. | medium | Explicitly set a threshold (e.g., 0.1) for warning. |
|14| Rates must be computed using ψ(t_k) before evolving to t_{k+1}; the plan must state this explicitly. | blocker | Specify order. |
|15| Using RK4 sub‑stages for ψ while sampling jumps only at full steps is consistent if dt is small; max_lambda_dt check covers it. | low | – |
|16| Use the exact same rate function as 0001 to avoid discrepancies; the plan implies that. | low | – |
|17| Rate caching orientation must be clear; tests should verify. | medium | Add tests for orientation. |
|18| Empirical equivariance should be computed at the same sampled steps as fidelity (sample_every_steps); plan implies that. | low | – |
|19| Sampling initial distribution is sufficient; later marginal checks are done via empirical equivariance. | none | – |
|20| If |ψ_q|² is tiny, the actual configuration is unlikely, but the jump process still works; no special handling needed. | none | – |
|21| ConditionalNormFloor = 1e‑14 is reasonable for double precision. | low | – |
|22| Norm failure should be based on absolute norm squared; the plan says that. | none | – |
|23| Norm failures should make the bridge inconclusive if above a small percentage; the plan should define the threshold. | high | Define threshold. |
|24| Precomputing by environment using full‑basis indices works if the bijection is correct; test required. | high | Add test. |
|25| Sorting site lists internally avoids ambiguity; the plan should document the canonical order. | medium | Sort and document. |
|26| Yes, report should include the actual ordering. | low | Add to report. |
|27| Include test for Combine(Split(q)) == q. | high | Mandatory test. |
|28| Include non‑contiguous partitions tests. | medium | Add. |
|29| Include boundary‑crossing tests. | medium | Add. |
|30| Fidelity is sufficient; trace distance could be added but not necessary. | low | – |
|31| Schema version bump is optional but recommended. | low | Consider v0_2a. |
|32| Bridge section omitted when disabled is fine for backward compatibility. | none | – |
|33| “bridge_toy_passed” is acceptable; it clearly indicates toy scope. | low | – |
|34| Rename as suggested. | high | Rename. |
|35| Ratio should be reported as null when event counts are low; the plan does that. | low | – |
|36| JSON null is fine. | none | – |
|37| Warnings should include all those factors; plan can add. | medium | Add warnings. |
|38| goal_status remains independent; good. | none | – |
|39| Debt update should only occur if bridge_toy_passed; plan implies it but should be explicit. | medium | Clarify. |
|40| “partially_paid_conditional_state_toy_only” is okay if interpreted as toy diagnostic. | low | – |
|41| Tests with both orientation and fake rates needed. | high | Add tests. |
|42| Off‑by‑one alignment must be tested with a deterministic small system. | high | Add integration test. |
|43| Use deterministic RNG and fake rates for unit tests; stochastic tests can be statistical with fixed seed. | medium | Adopt both. |
|44| Semantic comparison is better; byte‑for‑byte may break due to JSON key order. | low | Use semantic. |
|45| Add a test with known environment jumps to ensure metrics are computed. | high | Add. |
|46| Zero‑rate case should produce inconclusive; test required. | medium | Add. |
|47| Forbidden‑language scan should cover both JSON and Markdown; plan includes. | none | – |
|48| Bridge‑disabled path should preserve all 0001 metrics; integration test needed. | high | Add. |
|49| Test max_lambda_dt warning logic. | medium | Add. |
|50| Additional tests: check that conditional state normalization matches analytical for small partition with known ψ; test the rate sampling distribution for simple rates. | medium | Add if time. |

---

## Recommended Revised Plan

1. **Clarify the site‑split bijection**:  
   - Use a fixed canonical ordering of all sites.  
   - For a given subsystem site list `A` (sorted) and environment `B` (complement, sorted), define a function `fullIndex = combine(aIdx, bIdx)` that interleaves bits according to the sorted order.  
   - Include an invariant `combine(split(q)) == q` in tests.

2. **Define the per‑step order**:  
   - At each step `k` with `ψ_k` and `q_k`:  
     - Compute Bell rates `σ(n ← q_k)` using `ψ_k`.  
     - Sample jump to `q_{k+1}` with probability `1 - exp(-λ dt)`.  
     - Evolve `ψ_k` → `ψ_{k+1}` using the existing RK4 (unaffected by the jump).  

3. **Rename “monitoring_like_signal”** → `conditional_update_correlation` with values: `not_assessed`, `no_clear_correlation`, `weak_correlation`, `candidate_correlation`.  
   - Add a rule: if total environment‑jump transitions < 10, set to `not_assessed`.

4. **Set thresholds** for norm failures: if > 5% of sampled conditional states fail normalization → `bridge_toy_inconclusive`; if > 50% → `bridge_toy_failed`.

5. **Add tests**:  
   - Rate orientation test with a fake rate matrix.  
   - Integration test on a 2‑site system where environment jumps are forced.  
   - Test for zero‑rate eigenstate → inconclusive.  
   - Semantic comparison for bridge‑disabled runs.

6. **Extend limitations** to include finite‑size and finite‑sample warnings.

---

## EBP 2.1 Ledger

```json
{
  "needMap": "partially_paid_conditional_state_toy_only",
  "needInvariant": "partially_paid_equivariance_plus_empirical_trajectory_check",
  "needToyCheck": "partially_paid_rate_algebra_and_conditional_state_toy",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_reviewed_for_0001_only",
  "promotion_status": "unpromoted_plan_review_only"
}
```

---

## Final Recommendation

**Proceed to implementation after repairs.**  
The plan is fundamentally sound, but the three critical blockers (site‑split ordering, time‑step alignment, and naming of “monitoring_like_signal”) must be addressed before coding begins. Once those are corrected and the high‑priority tests are added, the implementation can go forward safely.
