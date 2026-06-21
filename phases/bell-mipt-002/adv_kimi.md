# Adversarial Review: BELL-MIPT-0002A Implementation Plan

## Executive Verdict

**use_as_base_with_repairs**

The plan is conceptually sound and implementable, but contains several critical blockers around time-step alignment, rate orientation, and status logic that must be repaired before implementation. The physics framing is mostly safe but has overclaim risks in naming and interpretation.

---

## Strongest Parts of the Plan

1. **Two-pass architecture** — cleanly separates BELL-MIPT-0001 master-equation audit from bridge audit, preserving backward compatibility.
2. **Deterministic RNG with explicit seed** — makes trajectories reproducible and tests deterministic.
3. **Conditional norm failure tracking** — honest about numerical edge cases rather than silently producing garbage.
4. **Forbidden language audit extension** — explicitly checks that bridge text does not overclaim.
5. **Separate `bridge_status` from `goal_status`** — prevents bridge noise from invalidating the base equivariance result.
6. **No hard empirical L1 threshold** — correctly treats finite-sample equivariance as descriptive only.
7. **Partition validation** — overlap and completeness checks prevent invalid subsystem splits.

---

## Critical Blockers

### Blocker 1: Time-step alignment between ψ(t) and Q(t) is unspecified
The plan says "use ψ snapshots to sample actual Bell configuration trajectories" but does not specify whether rates are computed from ψ(t_k) before evolution, ψ(t_{k+1}) after evolution, or an intermediate value. For a time-dependent Hamiltonian and time-dependent ψ, the Bell rates σ(n←q, t) depend on t. If Q(t_k) = q and we evolve ψ(t_k) → ψ(t_{k+1}), the jump decision at step k must use rates consistent with the ψ used for that interval. The plan must specify: **rates computed from ψ(t_k) before RK4 step, then ψ evolved, then Q updated.**

### Blocker 2: Rate matrix orientation is untested and ambiguous
The plan mentions "rate-row accessor" but does not specify whether `rates[n][q]` means "from q to n" or "from n to q." A single orientation bug would corrupt all jump probabilities. The plan must: (a) document the convention explicitly, (b) add a deterministic fake-rate test where the expected trajectory is known analytically, (c) verify that `Σ_n rates[n][q] == totalRate(q)` for every q.

### Blocker 3: "monitoring_like_signal" is too strong
Even with the disclaimer that it is "descriptive only," the phrase "monitoring-like signal" carries measurement/MIPT connotations. In the context of Bell trajectories, "monitoring" implies an observer effect. This must be renamed to something strictly descriptive like `conditional_update_pattern` or `fidelity_drop_pattern`.

### Blocker 4: `candidate_signal` is too strong
Same issue as above. Must be renamed to `conditional_update_candidate` or `fidelity_drop_candidate`.

### Blocker 5: Discrete-time thinning allows at most one jump per dt
The plan uses `p_jump = 1 - exp(-λdt)` which is the probability of *at least one* jump in a Poisson process over dt. But the implementation then samples *at most one* destination. For λdt ~ 0.1 this is fine, but `max_lambda_dt` must gate this. The plan must specify: **if λdt > 0.1, mark bridge as inconclusive and warn that multiple jumps per step may be missed.**

### Blocker 6: `conditional_update_ratio` must not gate `bridge_status`
The plan says this, but the status logic section does not explicitly show how `bridge_status` is computed. It must be made explicit: `bridge_status` depends only on whether trajectories were sampled successfully, conditional states computed reliably, and metrics emitted. It must NOT depend on the value of `conditional_update_ratio`.

---

## High-Priority Repairs

1. **Rename `monitoring_like_signal` → `conditional_update_pattern`**
2. **Rename `candidate_signal` → `conditional_update_candidate`**
3. **Specify exact time-step alignment**: rates from ψ(t_k), then RK4 evolution, then jump decision, then Q update.
4. **Add deterministic fake-rate trajectory test** to catch orientation bugs.
5. **Add `max_lambda_dt` threshold**: if exceeded, `bridge_status = bridge_toy_inconclusive` with warning.
6. **Document rate convention explicitly**: `rates[dest][src]` = rate from src to dest.
7. **Add test for non-contiguous partitions** (e.g., A=[0,2,5], B=[1,3,4]).
8. **Add test for boundary-crossing jumps** where both A and B change.
9. **Specify that `conditional_update_ratio` reports `null` when denominator events < 10** (not just when mean is zero).
10. **Add warning field to report** for `max_lambda_dt`, low event counts, norm failures, finite-size limitations.

---

## Medium-Priority Repairs

1. **Report schema version bump** — add `"schema_version": "bell_mipt_report_v0_2a"` to report root.
2. **Bridge section present when disabled** — use `"bridge": {"enabled": false, "bridge_status": "bridge_disabled"}` for forward compatibility.
3. **Conditional norm floor** — `1e-14` is acceptable for double precision with ~64-dimensional Hilbert space, but should be parameterized and tested.
4. **Empirical equivariance** — use only sampled steps, not all micro-steps, to match the fidelity sampling cadence.
5. **Initial distribution verification** — add a test that marginal distribution of Q(t) at intermediate times matches `|ψ(t)|²` with a loose tolerance (e.g., L1 < 0.5 for 200 trajectories), to catch drift in the sampler.
6. **Rate caching** — if implemented, cache must be keyed by (step, q) and invalidated if ψ changes. Document that caching is an optimization only.
7. **Subsystem site ordering** — preserve user-specified ordering because it defines the subsystem basis ordering. Do not sort internally.
8. **Fidelity vs. trace distance** — fidelity is sufficient for 0002A. Do not add trace distance unless requested later.
9. **Zero-rate eigenstate test** — add a test where the initial state is a Hamiltonian eigenstate (no jumps expected), verify `bridge_status` handles this honestly.
10. **Byte-for-byte backward compatibility** — integration test should compare key fields, not full JSON, to avoid brittle failures on whitespace/schema additions.

---

## Low-Priority Polish

1. **Rename `bridge_toy_passed` → `bridge_audit_completed`** — "passed" implies success criteria were met, but the bridge has no success threshold.
2. **Rename `bridge_toy_failed` → `bridge_audit_failed`** — same reasoning.
3. **Rename `bridge_toy_inconclusive` → `bridge_audit_inconclusive`** — same reasoning.
4. **Unavailable ratios** — serialize as JSON `null` with a `reason` string in warnings, not as `0`.
5. **Debt status** — if `bridge_status` is `inconclusive`, do not update debt to `partially_paid`; keep it at the previous level.
6. **Markdown rendering** — ensure bridge section renders even when disabled, showing "Bridge audit disabled."
7. **Config file naming** — `bellmipt_bridge.json` is fine, but document that any config file can contain the bridge section.

---

## Specific Answer Table

| Question | Answer | Severity | Repair |
|----------|--------|----------|--------|
| 1. Conditional WF validity for fermionic basis | Valid for occupation basis; no Jordan-Wigner sign issue because configurations are labeled by occupation numbers, not fermionic operators. The conditional state is a slice of the full amplitude vector, not a partial trace. | none | None |
| 2. Conditioning on env occupation for pairing terms | Valid but subtle. Pairing terms do not conserve particle number, but the conditional state is still well-defined as a slice of the full wavefunction. The physical interpretation is weaker, but the math is fine. | low | Add note in limitations: "Pairing terms present; conditional state is formal, not particle-number eigenstate." |
| 3. Is Q_B(t) enough to call this a conditional subsystem state? | Yes for this toy. In Bell-type QFT there are additional conditions (e.g., Bell's equivariance, current conservation), but for a finite lattice toy, slicing the amplitude vector by environment configuration is the standard conditional-state construction. | none | None |
| 4. Risk of confusing jumps with measurements | Present. The phrase "monitoring-like signal" is the main risk. The report text must be audited carefully. | high | Rename signal field; strengthen forbidden-language audit. |
| 5. Fidelity drops dominated by Schrödinger evolution? | Possible. Smooth Schrödinger evolution changes ψ_A even without jumps. The comparison "environment jumps vs. non-environment jumps" partially controls for this, but a better null would be "same environment config, different time steps." | medium | Add note in interpretation: "Fidelity drops may include smooth evolution contributions." |
| 6. Better null model for fidelity drops? | Yes, but out of scope for 0002A. The current null is acceptable for a first diagnostic. | low | Document as future work. |
| 7. "monitoring_like_signal" too strong? | Yes. Implies measurement/MIPT. | high | Rename to `conditional_update_pattern`. |
| 8. Subsystem/env jump classification with boundary-crossing terms? | A single Hamiltonian term can change both A and B. The plan classifies by whether Q_A and/or Q_B changed. This is correct for the discrete configuration basis. | none | Add test for boundary-crossing jumps. |
| 9. Empirical equivariance usefulness? | Mostly noise with 200 trajectories and 64 states. It is a sanity check, not a precise test. | low | Keep as descriptive only; do not gate status. |
| 10. Finite-size artifacts misleading signals? | Yes. Small Hilbert space means conditional states are high-dimensional and jumps are rare. Signals may not generalize. | medium | Add to limitations: "Finite toy; signals may be finite-size artifacts." |
| 11. Discrete-time thinning correctness for time-dependent rates? | Approximate. For small dt and smooth ψ(t), it is acceptable. The error is O((λdt)²) per step. | medium | Add `max_lambda_dt` check. |
| 12. Is `max_lambda_dt` sufficient? | Barely. A threshold of 0.1 is reasonable. Should warn if exceeded. | medium | Add explicit threshold and warning. |
| 13. At most one jump per dt? | The current algorithm samples at most one. For λdt << 1, probability of multiple jumps is negligible. | medium | Document assumption; gate with `max_lambda_dt`. |
| 14. Rates computed before or after evolution? | Must be from ψ(t_k) before evolution. The plan is ambiguous. | high | Specify explicitly in plan. |
| 15. RK4 sub-stages vs. jump sampling at full steps? | Sampling only at full steps is consistent with the discrete-time approximation. No need to sample at RK4 sub-stages. | none | Document that jumps are sampled at full steps only. |
| 16. Same rate computation as 0001? | Yes, must use identical rate function to ensure consistency. A rate-row accessor is safe if it wraps the same function. | medium | Add test verifying rate consistency between 0001 and bridge. |
| 17. Rate caching risks? | Low if keyed by (step, q). Risk if orientation is misunderstood. | medium | Add fake-rate test. |
| 18. Empirical equivariance: all micro-steps or sampled steps? | Sampled steps only, to match fidelity cadence. | low | Specify in plan. |
| 19. Initial distribution sampling sufficient? | Yes for t=0. Should also verify marginal at final time descriptively. | low | Add loose L1 check at final time. |
| 20. What if |ψ_q|² below rate floor? | If the actual configuration has negligible probability, the conditional state may be ill-defined. This is caught by norm failure. | medium | Document: "If current Q has negligible |ψ|², conditional state may be unreliable." |
| 21. ConditionalNormFloor = 1e-14? | Acceptable for double precision with dim ~ 64. | none | Parameterize and test. |
| 22. Absolute vs. relative norm failure? | Absolute is fine for this scale. Relative would be more robust but overkill. | low | Keep absolute; document. |
| 23. Norm failures → inconclusive, failed, or reported? | Reported. If too many (>10% of samples), mark inconclusive. | medium | Specify threshold in plan. |
| 24. ByEnvironment precomputation for arbitrary partitions? | Yes, if split/reconstruct uses bit masks derived from site lists. | none | Add `CombineConfig(SplitConfig(q)) == q` invariant test. |
| 25. Site list ordering preserved? | Yes, user-specified ordering defines subsystem basis ordering. Do not sort. | medium | Document explicitly. |
| 26. Report normalized site ordering? | Yes, helpful for reproducibility. | low | Add `actual_subsystem_sites` and `actual_environment_sites` to report. |
| 27. Combine(Split(q)) == q catch all bugs? | Catches most bit-manipulation bugs. Good invariant. | none | Add as test. |
| 28. Test non-contiguous partitions? | Yes, essential. | high | Add test. |
| 29. Test boundary-crossing jumps? | Yes, essential. | high | Add test. |
| 30. Fidelities enough, or also trace distance? | Fidelity is sufficient for 0002A. | none | None |
| 31. Schema version bump? | Yes. | low | Add `"schema_version": "bell_mipt_report_v0_2a"`. |
| 32. Bridge section when disabled? | Present with `enabled: false`. Better for forward compatibility. | low | Include in report. |
| 33. "bridge_toy_passed" too strong? | Yes. "audit_completed" is better. | medium | Rename. |
| 34. "candidate_signal" too strong? | Yes. | high | Rename to `conditional_update_candidate`. |
| 35. `conditional_update_ratio` with low event counts? | Hide/null if denominator events < 10. | high | Specify minimum count threshold. |
| 36. Unavailable ratios as null/NaN/omitted? | JSON `null` is clean. Add `reason` in warnings. | medium | Use `null`. |
| 37. Warnings for diagnostics? | Yes, essential for honest reporting. | high | Add `warnings` array to report. |
| 38. `goal_status` fail if `bridge_status` fails? | No, must remain independent. | none | Document independence. |
| 39. Debt update when inconclusive? | Do not update debt if inconclusive. Overstates payment. | high | Only update debt if `bridge_audit_completed`. |
| 40. `partially_paid_conditional_state_toy_only` overstates? | Yes, if metrics are noisy. Only update if bridge completed successfully. | medium | Gate debt update on `bridge_audit_completed`. |
| 41. Tests catch rate orientation bugs? | Current plan insufficient. Need fake-rate test. | high | Add deterministic fake-rate test. |
| 42. Tests catch time-step alignment bugs? | Current plan insufficient. Need test with known trajectory. | high | Add test with analytically known rates. |
| 43. Stochastic tests stable for CI? | No, need deterministic fake RNG for unit tests. | high | Use deterministic fake RNG in tests. |
| 44. Byte-for-byte backward compatibility? | Compare semantic fields, not full JSON. | medium | Use field-level comparison. |
| 45. Tests force known environment jumps? | Yes, needed to avoid always-inconclusive. | high | Add test with high-rate env jumps. |
| 46. Zero-rate eigenstate test? | Yes, needed for honest inconclusive handling. | medium | Add test. |
| 47. Forbidden-language audit on full report? | Yes, must scan assembled Markdown and JSON. | high | Add integration test. |
| 48. Bridge-disabled path preserves 0001 metrics? | Yes, must verify. | high | Add integration test. |
| 49. Tests verify `max_lambda_dt` logic? | Yes, needed. | medium | Add test. |
| 50. Additional tests required? | (a) Fake-rate deterministic trajectory, (b) Non-contiguous partition, (c) Boundary-crossing jump, (d) Zero-rate eigenstate, (e) Forbidden-language scan, (f) `max_lambda_dt` warning, (g) Rate consistency between 0001 and bridge. | high | Add all listed tests. |

---

## Recommended Revised Plan

### Architecture (unchanged)
Two-pass mode. Pass 1: BELL-MIPT-0001 master-equation audit. Pass 2: bridge audit if enabled.

### File Layout (revised)
```
Modify:
  internal/bellmipt/config.go        // Add BridgeConfig, Validate()
  internal/bellmipt/run.go           // Two-pass orchestration
  internal/bellmipt/report.go        // Add schema_version, BridgeReport, warnings
  internal/bellmipt/markdown.go      // Render bridge section
  internal/bellmipt/forbidden.go     // Extend to scan bridge text

Add:
  internal/bellmipt/bridge_config.go // BridgeConfig struct and validation
  internal/bellmipt/site_split.go    // Site partition, split/reconstruct, tests
  internal/bellmipt/trajectory.go    // TrajectorySampler with deterministic RNG
  internal/bellmipt/conditional.go   // ConditionalBuilder, fidelity
  internal/bellmipt/bridgemetrics.go // BridgeAuditor, metrics accumulation
  internal/bellmipt/equivariance_empirical.go // Empirical equivariance L1
  internal/bellmipt/bridge_report.go // BridgeReport, status logic, warnings
  internal/bellmipt/bridge_test.go   // All bridge tests (fake RNG, fake rates, etc.)
```

### Time-Step Alignment (new)
```
For each step k:
  1. ψ_k is the wavefunction at t_k (from RK4 evolution or initial state).
  2. Compute rates σ(n ← q_k, t_k) from ψ_k.
  3. Compute λ(q_k, t_k) = Σ_n σ(n ← q_k, t_k).
  4. Check max_lambda_dt. If λ*dt > 0.1, record warning.
  5. Sample jump using p_jump = 1 - exp(-λ*dt) and destination proportional to rates.
  6. Evolve ψ_k → ψ_{k+1} using RK4 (same as 0001).
  7. Update q_{k+1} = destination if jump, else q_k.
  8. If step % sample_every == 0, record sample.
```

### Status Logic (revised)
```
bridge_status:
  bridge_disabled                         // bridge.enabled == false or omitted
  bridge_audit_completed                  // trajectories sampled, metrics emitted, no impl failures
  bridge_audit_failed                   // trajectory sampling or conditional-state computation failed
  bridge_audit_inconclusive             // >10% norm failures, or λ*dt > 0.1, or too few env jumps

conditional_update_ratio does NOT determine bridge_status.
```

### Report Schema (revised)
```json
{
  "schema_version": "bell_mipt_report_v0_2a",
  "toy_analysis_only": true,
  "physics_claim": "none",
  "goal_status": "...",
  "bridge": {
    "enabled": true,
    "bridge_goal": "sample_bell_trajectories_and_audit_conditional_subsystem_state",
    "bridge_status": "bridge_audit_completed",
    "actual_subsystem_sites": [0, 1, 2],
    "actual_environment_sites": [3, 4, 5],
    "trajectories": 200,
    "sample_every_steps": 1,
    "metrics": { ... },
    "interpretation": {
      "conditional_update_pattern": "not_assessed | no_clear_pattern | weak_pattern | candidate_pattern",
      "reason": ""
    },
    "warnings": [
      {"code": "max_lambda_dt_exceeded", "message": "..."},
      {"code": "low_event_count", "message": "..."}
    ]
  },
  "debt_status": { ... },
  "limitations": [ ... ]
}
```

### Debt Update (revised)
Only update debt if `bridge_status == "bridge_audit_completed"`. If `inconclusive` or `failed`, retain old debt.

### Tests (revised, critical additions)
1. **Fake-rate deterministic trajectory test**: Use a 2-state system with known rates. Verify exact trajectory for fixed seed.
2. **Rate orientation test**: Verify `Σ_n rates[n][q] == totalRate(q)` for all q.
3. **Non-contiguous partition test**: A=[0,2,5], B=[1,3,4]. Verify split/reconstruct roundtrip.
4. **Boundary-crossing jump test**: Hamiltonian term changing both A and B. Verify classification.
5. **Zero-rate eigenstate test**: Initial state = eigenstate. Verify no jumps, bridge inconclusive or completed with zero jumps.
6. **Forbidden-language integration test**: Scan fully assembled Markdown and JSON for forbidden phrases.
7. **Backward compatibility test**: Bridge disabled → report fields identical to 0001 (semantic comparison).
8. **max_lambda_dt warning test**: Force high rate, verify warning and inconclusive status.
9. **Rate consistency test**: Bridge rates must equal 0001 rates for same ψ.
10. **Conditional norm failure test**: Construct ψ with zero support on env config. Verify failure recorded.

---

## EBP 2.1 Ledger

```json
{
  "needMap": "unpaid",
  "needInvariant": "partially_paid_equivariance_only",
  "needToyCheck": "partially_paid_rate_algebra_only",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_reviewed",
  "promotion_status": "unpromoted_plan_review_only"
}
```

---

## Final Recommendation

**Proceed to implementation after repairs.**

The plan is fundamentally sound and EBP-safe, but the following repairs are mandatory before coding begins:

1. Specify exact time-step alignment (rates from ψ_k before evolution).
2. Add fake-rate deterministic trajectory test to catch orientation bugs.
3. Rename `monitoring_like_signal` → `conditional_update_pattern`.
4. Rename `candidate_signal` → `conditional_update_candidate`.
5. Rename `bridge_toy_passed/failed/inconclusive` → `bridge_audit_completed/failed/inconclusive`.
6. Add `max_lambda_dt` threshold (0.1) with warning and inconclusive gating.
7. Gate debt update on `bridge_audit_completed` only.
8. Add `warnings` array to report schema.
9. Add tests for non-contiguous partitions and boundary-crossing jumps.
10. Use JSON `null` for unavailable ratios, not `0`.

After these repairs, the plan is ready for implementation.
