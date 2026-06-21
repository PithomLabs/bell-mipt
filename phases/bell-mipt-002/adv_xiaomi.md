# Adversarial Review: BELL-MIPT-0002A Final Implementation Plan

---

## Executive Verdict

```
use_as_base_with_repairs
```

---

## Strongest Parts of the Plan

1. **Two-pass architecture with stored ψ snapshots.** Decoupling the universal wave function evolution from trajectory sampling is correct: ψ(t) is the same for all trajectories and must not branch. Storing snapshots avoids recomputation and ensures trajectory samplers use identical ψ(t) as the master-equation audit.

2. **Bridge-disabled preserves backward compatibility.** The decision to keep the BELL-MIPT-0001 code path completely unchanged when `bridge.enabled=false` is the right call. It avoids regression risk and makes the diff reviewable.

3. **conditional_update_ratio does not determine bridge_status.** This was explicitly called out in the rejected/repairs list. Keeping status logic based on implementation-level success (norm failures, jump counts, numerical stability) rather than physics-level signal strength is correct for a toy.

4. **Nullable unavailable metrics.** Using JSON `null` rather than `NaN`, `0`, or omission for unavailable ratios is the cleanest choice for downstream consumers.

5. **Explicit non-goals list and debt-status accounting.** The plan is disciplined about what it is not claiming and maintains honest EBP ledger entries.

6. **Rate caching per time step.** Avoiding redundant rate computation across trajectories is a good optimization with no correctness risk as long as cache orientation is verified.

7. **ConditionalNormFloor as a hard cutoff.** Defining a clear threshold below which conditional-state normalization is unreliable and the sample is skipped prevents numerical garbage from polluting metrics.

8. **max_lambda_dt diagnostic.** Flagging when discrete-time thinning is unreliable is essential for honest reporting.

---

## Critical Blockers

### Blocker 1: Jordan-Wigner sign handling under non-contiguous partitioning

The plan does not address JW sign conventions when splitting a full configuration `q` into subsystem and environment parts. In a JW representation, the sign of `c†_i |n⟩` depends on the number of occupied sites *below* site `i` in the canonical ordering. When a basis state is projected onto an environment branch `b₀`, the resulting conditional state `ψ_A(a,t)` inherits a JW sign that depends on the full configuration. If the site partition is non-contiguous (e.g., A=[0,2,5], B=[1,3,4]), the re-indexing of subsystem bits into a compact `[0, 2^|A|)` index must preserve these signs or the conditional state will be physically incorrect. The plan mentions bit masking and index mapping but does not mention sign tracking. **This must be addressed before implementation.**

### Blocker 2: Terminology risk of conflating with measurement/MIPT

The plan uses phrases like "monitoring_like_signal," "candidate_signal," and "conditional subsystem state" in a context that structurally resembles monitored quantum circuits. Even with explicit disclaimers, the naming creates a risk that the analysis framework will be read (internally or externally) as detecting measurement-like behavior. The reference content explicitly rejects claims that "Bell jumps are measurements" or that "MIPT was observed." The terminology should be tightened to avoid the appearance of claiming those results descriptively.

### Blocker 3: Time-step alignment between ψ(t) and Q(t) not specified exactly

The plan says trajectories use "ψ(t)" for rate computation but does not specify whether rates are computed at ψ(t_k) before evolving to ψ(t_{k+1}), or at ψ(t_{k+1}) after evolving, or at a midpoint. The existing BELL-MIPT-0001 master-equation audit uses RK4 with sub-stages, so the "ψ at step k" may refer to the state at the beginning, middle, or end of the integration interval. If the trajectory sampler and the master-equation audit use different conventions, their equivariance comparison becomes meaningless. **This must be pinned down exactly before implementation.**

---

## High-Priority Repairs

### H1: Add JW sign documentation and test for non-contiguous partitions

The plan's site-splitting code must explicitly track and document the Jordan-Wigner sign convention used when extracting the conditional state. Specifically:

- When computing `ψ_A(a, t) = Ψ(a, b₀, t)`, the amplitude for subsystem configuration `a` is the sum of all full-basis amplitudes whose subsystem bits match `a` and environment bits match `b₀`. Each such amplitude carries a phase from the JW string. The code must not re-sign these amplitudes during the split.
- Add a test with a non-contiguous partition (e.g., A=[0,2,5], B=[1,3,4]) on a 6-site system, verifying that `ConditionalFidelity(psi_A, psi_A) == 1.0` for a known ψ.

### H2: Rename "monitoring_like_signal" to "environment_correlated_conditional_update"

Replace:
- `monitoring_like_signal` → `environment_correlated_conditional_update`
- `candidate_signal` → `candidate_update_enhancement`
- `weak_signal` → `weak_update_enhancement`
- `no_clear_signal` → `no_clear_enhancement`
- `not_assessed` → `not_assessed` (this one is fine)

This removes the word "monitoring" from the report entirely, which avoids the appearance of claiming monitored-circuit behavior.

### H3: Add hard threshold on max_lambda_dt

The plan says to "log a warning" when `λdt > 0.1` but does not specify a consequence for bridge_status. Add:

- If `max_lambda_dt > 0.5` at any step, set `bridge_status = bridge_toy_inconclusive` with reason `"discrete-time thinning unreliable: max_lambda_dt exceeds threshold"`.
- If `max_lambda_dt > 0.1` but < 0.5, include a warning in the report but do not force inconclusive.

### H4: Specify exact time-step alignment

Define: rates for the transition `Q_k → Q_{k+1}` are computed using `ψ(t_k)` (the wave function at the beginning of the interval, before the RK4 step that produces `ψ(t_{k+1})`). This matches the Euler-forward convention where the rate is evaluated at the current state. Document this explicitly in the implementation plan. The trajectory sampler and master-equation audit must both use ψ snapshots indexed by the same convention.

### H5: Require minimum event counts for conditional_update_ratio reporting

Do not report `conditional_update_ratio` unless:
- `environment_jump_transitions >= 10`
- `non_environment_jump_transitions >= 10`

Below these thresholds, report `null` with a reason string.

### H6: Add Schrödinger-only fidelity baseline comparison

Add one more metric: compute fidelity drops of the conditional state along a single reference trajectory where **no jumps ever occur** (i.e., Q(t) = Q(0) for all t, while ψ(t) evolves normally). This establishes the Schrödinger-only baseline for conditional-state fidelity decay. Then the comparison becomes:

```
environment_jump_enhancement = mean_drop_at_env_jumps / mean_drop_schrodinger_only
```

This is a strictly better null than "non-environment-jump transitions" because the Schrödinger-only baseline isolates smooth phase evolution from all jump effects. Add this as a single extra trajectory (or average over a few) at marginal computational cost.

---

## Medium-Priority Repairs

### M1: Add deterministic fake-RNG tests for stochastic components

The trajectory sampling tests must include cases with deterministic fake RNGs (e.g., a sequence that always returns 0.5, or a pre-recorded sequence). This catches orientation and probability-computation bugs without stochastic flakiness. The statistical distribution tests can use real RNG with generous tolerances, but the correctness tests must be deterministic.

### M2: Add trace distance alongside fidelity

Fidelity alone can miss certain conditional-state changes (e.g., amplitude redistribution that preserves overlap with the previous state). Add a trace distance metric:

```
D(ψ₁, ψ₂) = Σ_a |ψ₁(a)|² - |ψ₂(a)|²|
```

(note: for pure states this reduces to L1 distance of probability distributions). Report it alongside fidelity as a secondary diagnostic.

### M3: Add boundary-crossing jump classification test

Add a test where a Hamiltonian term crosses the A/B boundary, causing a single jump event that changes both `Q_A` and `Q_B`. Verify the classification correctly tags `envJumped=true` and `subsysJumped=true` simultaneously.

### M4: Verify rate matrix orientation in tests

Add a test that constructs a known 2- or 3-site system, computes rates, and verifies `rates[destination][source]` by hand. This catches `rates[n][q]` vs `rates[q][n]` transposition bugs, which are the most common matrix-orientation error.

### M5: Use semantic field comparison, not byte-for-byte, for integration tests

Byte-for-byte comparison of `report.json` breaks on harmless changes (timestamp fields, map iteration order, floating-point formatting differences across platforms). Compare key semantic fields: `goal_status`, `bridge_status`, metric values within tolerance, debt_status strings.

### M6: Include conditional_update_ratio availability in a structured way

When ratio is unavailable, include:

```json
{
  "conditional_update_ratio": null,
  "conditional_update_ratio_status": "unavailable_insufficient_events",
  "conditional_update_ratio_env_event_count": 3,
  "conditional_update_ratio_no_env_event_count": 7
}
```

This is more informative than a bare `null`.

### M7: Verify bridge-disabled path preserves all 0001 metrics exactly

Add an explicit test that runs the program with bridge disabled and compares all 0001 metric fields against a stored baseline. This is the backward-compatibility regression test.

---

## Low-Priority Polish

### L1: Rename `bridge_toy_passed` to `bridge_audit_completed`

"passed" implies a physics result was confirmed. "completed" is more honest: the audit ran to completion and produced valid metrics.

### L2: Bump report schema version to `bell_mipt_report_v0_2a`

The bridge section is a nontrivial addition to the report structure. Bumping the version signals to consumers that the schema has changed. When bridge is disabled, the report can still use `bell_mipt_report_v0` for exact backward compatibility.

### L3: Report the actual normalized site ordering used

In the bridge section of the report, include:

```json
"subsystem_sites_normalized": [0, 1, 2],
"environment_sites_normalized": [3, 4, 5]
```

This documents whether internal sorting was applied and removes ambiguity.

### L4: Include warnings array in bridge report

```json
"warnings": [
  "max_lambda_dt=0.083 at step 456, approaching reliability threshold",
  "conditional norm failures: 12/200 samples (6.0%)",
  "environment jump count per trajectory: mean=2.3, some trajectories have 0"
]
```

This makes diagnostic information machine-readable.

### L5: Include test for Schrödinger phase drift on fidelity

Add a test where ψ(t) evolves freely (no jumps) and verify that fidelity drops between conditional states at consecutive time steps are small (bounded by `O(dt * energy_spread)`). This verifies the plan's claim that phase evolution effects are distinguishable from jump effects.

---

## Specific Answer Table

### A. Physics/Conceptual Correctness

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 1 | JW sign/convention issue under partitioning? | Yes, this is a real risk. The JW string depends on the full configuration. When extracting `ψ_A(a,t) = Ψ(a,b₀,t)`, the amplitudes carry JW phases from the full basis. Re-indexing subsystem bits into compact `[0,2^|A|)` must not alter these phases. The plan does not mention sign handling. | **blocker** | Document sign convention; add non-contiguous partition test. |
| 2 | Does conditioning on occupation config make sense without particle-number conservation? | Yes, it is well-defined in the occupation-number basis. The Kitaev chain's pairing terms don't conserve particle number, but the basis states are still eigenstates of number operators per site. Conditioning on environment occupation is just selecting a branch of the wave function in a specific basis. No issue. | none | — |
| 3 | Is "conditional subsystem state" faithful without Bohmian/Bell QFT conditions? | Not fully. The computation is a projection onto an environment branch, which is a well-defined linear-algebraic operation. But calling it a "conditional subsystem state" in the Bell/Bohmian sense implies guidance-equation conditioning and QFT ordering, which are absent. The plan should call it "environment-projected conditional vector" or similar. | **high** | Rename to avoid Bohmian/Bell terminology implications. |
| 4 | Risk of confusing Bell jumps with measurements? | Yes. The structure (stochastic jumps → conditional-state updates → fidelity drops → "signal" detection) mirrors monitored quantum circuit analysis. Even with disclaimers, the naming ("monitoring_like_signal") invites the confusion. | **high** | Rename signal classification terms. |
| 5 | Are fidelity drops meaningful or dominated by phase evolution? | In a finite system with small dt, Schrödinger phase evolution between consecutive samples can produce nontrivial fidelity drops even without jumps. The plan compares env-jump drops against non-env-jump drops, which partially controls for this, but the non-env-jump category includes smooth evolution with occasional subsystem-only jumps. A Schrödinger-only baseline would be cleaner. | **high** | Add Schrödinger-only fidelity baseline trajectory. |
| 6 | Is "non-environment-jump" a good null comparison? | Reasonable but imperfect. Non-environment-jump transitions include (a) no-jump (pure Schrödinger evolution) and (b) subsystem-only jumps. Mixing these makes the null noisy. Separating no-jump from subsystem-jump helps, which the plan already does (it tracks `mean_fidelity_drop_no_jump` separately). Adding a pure Schrödinger baseline is still better. | **medium** | Add Schrödinger-only baseline; the plan already tracks no-jump separately which partially helps. |
| 7 | Is "monitoring_like_signal" too strong? | Yes. The word "monitoring" directly evokes monitored quantum circuits, which is one of the explicitly rejected concepts. | **high** | Rename (see H2). |
| 8 | Boundary-crossing jump classification? | A jump that changes both Q_A and Q_B simultaneously should be classified as `envJumped=true` AND `subsysJumped=true`. The plan's classification logic should handle this, but it is not explicitly tested. | **medium** | Add boundary-crossing test. |
| 9 | Is empirical trajectory equivariance useful? | Marginally. With 200 trajectories and 64 basis states, each state is sampled ~3 times on average per time step. The L1 error will be dominated by finite-sample noise (~1/√N per state). It serves as a sanity check that nothing is catastrophically wrong with the sampling, but it cannot detect subtle equivariance violations. Should be reported as descriptive only, not used for pass/fail. | **low** | Already descriptive-only in the plan. Add explicit statement that it is noise-dominated. |
| 10 | Risk of finite-size artifacts producing misleading signals? | Yes. With 6 sites (Hilbert dim 64), the system may exhibit recurrences, discrete-spectrum effects, or atypically large conditional-state changes that would not appear in a thermodynamic limit. The plan should note this as a known limitation. | **medium** | Add to limitations list. |

### B. Bell Trajectory Sampling Correctness

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 11 | Is discrete-time thinning correct when rates are time-dependent? | Yes, for small λdt. The exact jump probability is `1 - exp(-∫λ(t')dt')` over the interval. With time-dependent rates (via ψ(t)), using `p = 1 - exp(-λ(t_k) * dt)` is a first-order approximation. For the small dt values used (0.001) and typical λ, this is accurate. | none | — |
| 12 | Is max_lambda_dt sufficient? | A warning log is insufficient; a hard threshold should trigger inconclusive status. The boundary between "reliable" and "unreliable" is not sharp, but λdt > 0.5 means multi-jump probability is >20%, which is clearly unreliable. | **high** | Add hard threshold at 0.5 (see H3). |
| 13 | Should multiple jumps be allowed per dt? | No. Allowing multiple jumps per dt with the current formula would require either (a) sub-stepping within dt or (b) a compound Poisson process. The single-jump-per-dt model is correct when λdt ≪ 1. If λdt is not small, the code should flag this (via max_lambda_dt) rather than trying to handle multi-jump events. | none | — |
| 14 | Rate computation timing? | Must be specified. The natural choice is `ψ(t_k)` for the transition `Q_k → Q_{k+1}` (rates at beginning of interval). The plan is silent on this. | **blocker** | Specify explicitly (see H4). |
| 15 | RK4 sub-stage consistency? | If 0001's RK4 uses sub-stages internally, the "ψ snapshot at step k" should be the value at the full step boundary, not a sub-stage intermediate. The trajectory sampler must use the same snapshots. As long as snapshots are stored at step boundaries only, this is consistent. | **low** | Document that snapshots are at step boundaries. |
| 16 | Rate-row accessor vs full matrix? | A rate-row accessor is safe as long as it produces the same values as the full matrix computation. Add a test that verifies the accessor output matches the full matrix row. | **low** | Add consistency test. |
| 17 | Rate cache orientation risk? | If `rates[n][q]` means "rate from q to n" in 0001, the cache must use the same convention. Misunderstanding this orientation is a high-probability bug. | **high** | Add explicit orientation test (see M4). |
| 18 | All micro-steps or sampled steps for equivariance? | Sampled steps only. The trajectory is only defined at sampled points. Using micro-steps would require interpolating Q(t) between samples, which is undefined for a jump process. | none | — |
| 19 | Later marginal distribution verification? | Empirical trajectory equivariance already does this for sampled time points. No additional mechanism needed, but the plan should explicitly state that equivariance is checked at all sampled times, not just t=0 and t_final. | **low** | Clarify in plan. |
| 20 | What happens when |ψ_q|² is near zero for current config? | If `|ψ_q|²` is below a floor, all rates from q are zero (or infinity, depending on implementation). The code must treat `|ψ_q|² < floor` as "no jumps possible, stay at q." This prevents division by zero in rate computation. | **high** | Add explicit handling for near-zero `|ψ_q|²`. |

### C. Conditional-State Construction

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 21 | Is ConditionalNormFloor = 1e-14 acceptable? | For double-precision arithmetic with dim ≤ 64, 1e-14 is reasonable. The machine epsilon is ~2.2e-16, and accumulated errors from 64 additions are ~64 * 2.2e-16 ≈ 1.4e-14. So 1e-14 is at the noise floor. This means norm failures will only occur when the branch is truly negligible. | none | — |
| 22 | Absolute norm vs relative? | Absolute norm is simpler and sufficient for a toy. Relative norm (norm / sqrt(|ψ|²)) would be more physically meaningful but adds complexity. For 0002A, absolute is fine. | **low** | — |
| 23 | What should norm failure trigger? | Report and count. Set bridge inconclusive only if failures exceed a percentage (the plan uses 50% of samples). Individual failures are expected for rare branches. | none | Already handled in plan. |
| 24 | Does precomputing ByEnvironment work for arbitrary partitions? | Yes, as long as bit masking correctly extracts environment bits for arbitrary site positions. The risk is in the index computation, not the grouping. | **medium** | Add non-contiguous partition test. |
| 25 | Sort site lists or preserve user ordering? | Sort internally. The user ordering should not define basis ordering because (a) it creates ambiguity if two users specify different orderings for the same partition, and (b) the physics is independent of labeling. Document the sort. | **medium** | Sort internally and report normalized ordering. |
| 26 | Report actual ordering? | Yes. Include normalized (sorted) site lists in the report for auditability. | **low** | Add to report (see L3). |
| 27 | Does CombineConfig(SplitConfig(q)) == q catch all bugs? | No. It catches bit-mask errors but not index-mapping errors. An incorrect subsystem index would pass the round-trip test if the mask is correct but would produce wrong conditional-state amplitudes. | **medium** | Add explicit index-range and uniqueness tests. |
| 28 | Test non-contiguous partitions? | Yes, critical. See M3 and H1. | **high** | — |
| 29 | Test boundary-crossing jumps? | Yes. See M3. | **high** | — |
| 30 | Are fidelity drops enough? | For a first diagnostic, yes. Trace distance is a useful complement (see M2). Phase-insensitive metrics are less of a concern here because the conditional state changes are expected to be amplitude-driven (not phase-driven) when environment jumps occur. | **medium** | Add trace distance (see M2). |

### D. Report/Schema/Status Logic

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 31 | Bump schema version? | Yes, when bridge is enabled. When disabled, keep `bell_mipt_report_v0` for backward compatibility. | **low** | See L2. |
| 32 | Bridge section present when disabled? | Omit for backward compatibility. The existing consumers expect the old schema. | none | Already in plan. |
| 33 | Rename bridge_toy_passed? | Yes, to `bridge_audit_completed` (see L1). "Passed" implies a physics result. | **low** | — |
| 34 | Rename candidate_signal? | Yes, to `candidate_update_enhancement` (see H2). | **high** | — |
| 35 | Report ratio with low counts? | No. Require minimum counts (see H5). | **high** | — |
| 36 | Format for unavailable means/ratios? | JSON `null` with additional status fields (see M6). | **medium** | — |
| 37 | Should warnings be included? | Yes, as a structured array (see L4). | **low** | — |
| 38 | Should goal_status fail if bridge_status fails? | No. They measure different things. The master-equation audit is independent of trajectory sampling. | none | Already in plan. |
| 39 | Does debt-status update overstate for inconclusive bridge? | Yes. If bridge is inconclusive, the debt status should not claim payment for conditional-state or empirical-trajectory work. Only claim payment if bridge_audit_completed. | **medium** | Add conditional logic to debt status based on bridge_status. |
| 40 | Does "partially_paid_conditional_state_toy_only" overstate? | It is appropriately hedged with "partially" and "toy_only." Acceptable. | none | — |

### E. Testing/Validation

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 41 | Sufficient for rate-orientation bugs? | The plan mentions tests for Bell current antisymmetry and nonnegative rates from 0001, but does not explicitly test the `rates[destination][source]` orientation for trajectory sampling. This is a gap. | **high** | Add orientation test (see M4). |
| 42 | Sufficient for time-step alignment bugs? | The plan does not explicitly test that trajectory Q(t) distributions match master-equation ρ(t) at the same time points. | **medium** | Add alignment test. |
| 43 | Stochastic tests stable for CI? | No, unless deterministic fake RNGs are used for correctness tests. Statistical tests should use generous tolerances and be flagged as potentially flaky. | **high** | Add deterministic fake-RNG tests (see M1). |
| 44 | Byte-for-byte or semantic comparison? | Semantic. Byte-for-byte is fragile. | **medium** | Use semantic field comparison (see M5). |
| 45 | Force known environment jumps? | Yes. Use parameters that produce high jump rates (e.g., high off-diagonal Hamiltonian elements, initial state that is a superposition of many configurations). | **medium** | Add forced-jump test case. |
| 46 | Zero-rate eigenstate tests? | Yes. Verify bridge_status = inconclusive when no jumps occur. | **low** | Already implied by plan; make explicit. |
| 47 | Scan fully assembled reports? | Yes. The forbidden-language audit must scan the final Markdown and JSON, not just individual components. | **low** | Make explicit in test plan. |
| 48 | Verify bridge-disabled preserves 0001 exactly? | Yes. See M7. | **medium** | — |
| 49 | Test max_lambda_dt logic? | Yes. Construct a case where λdt > 0.5 and verify inconclusive status. | **medium** | Add to test plan. |
| 50 | Additional tests needed? | (a) JW sign correctness under non-contiguous partitioning. (b) Rate orientation verification. (c) Deterministic fake-RNG trajectory tests. (d) Boundary-crossing jump classification. (e) Schrödinger-only fidelity baseline. (f) Near-zero |ψ_q|² handling. (g) Rate accessor vs full matrix consistency. | **varies** | Covered above. |

---

## Recommended Revised Plan

### Changes to the proposed plan before implementation:

1. **JW signs:** Add explicit documentation of how JW signs are preserved during conditional-state extraction. Add test with non-contiguous partition A=[0,2,5], B=[1,3,4]. *This is the only blocker-level repair.*

2. **Time-step alignment:** Specify that rates for transition `Q_k → Q_{k+1}` are computed using `ψ(t_k)`. Store ψ snapshots at step boundaries only. Document this convention.

3. **max_lambda_dt hard threshold:** Add `bridge_status = bridge_toy_inconclusive` when `max_lambda_dt > 0.5`.

4. **Schrödinger-only baseline:** Add one extra "no-jump" trajectory per run where Q(t) = Q(0) for all t, computing conditional fidelity drops from smooth evolution only. Report as `mean_fidelity_drop_schrodinger_only`.

5. **Terminology:** Rename `monitoring_like_signal` → `environment_correlated_conditional_update`; rename signal levels to use `enhancement` instead of `signal`.

6. **Rate orientation test:** Add a hand-calculated test verifying `rates[destination][source]` orientation.

7. **Deterministic fake-RNG tests:** Add trajectory tests using predetermined random sequences.

8. **Minimum event counts for ratio:** Require ≥10 events in each category before reporting conditional_update_ratio.

9. **Near-zero |ψ_q|² handling:** Add explicit guard: if `|ψ_q|² < 1e-30`, set all rates from q to zero and skip jump sampling (stay at q).

10. **Conditional debt status:** If `bridge_status != bridge_audit_completed`, do not update debt status from 0001 baseline.

11. **Report normalized site ordering** in bridge section.

12. **Add trace distance** as secondary metric alongside fidelity.

Everything else in the plan is sound and can be implemented as described.

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

The debt-status updates are appropriate **if and only if** `bridge_status = bridge_audit_completed`. If the bridge is inconclusive, only `needToyCheck` should advance (because the rate algebra was already partially validated in 0001 and the trajectory sampler at least ran), but `needMap` and `needInvariant` should not advance.

---

## Final Recommendation

```
Proceed to implementation after repairs.
```

The plan is well-structured, honest about its scope, and architecturally sound. The two-pass design, backward compatibility, nullable metrics, and explicit non-goals are all correct decisions. The repairs above are specific and implementable. The only blocker-level issue (JW sign handling) is a documentation-and-testing gap, not a fundamental design flaw — the code itself just needs to be careful about not altering phases during the conditional-state extraction, and a test must verify this for non-contiguous partitions. All other repairs are high- or medium-priority polish that improve correctness guarantees and report honesty.
