# Final Adversarial Review: Repaired BELL-MIPT-0002A Plan

## Executive Verdict

**ready_for_implementation_with_minor_repairs**

The repaired plan is now conceptually sound, numerically honest, and EBP-safe. The terminology repairs successfully defuse the overclaim risks. Three minor clarifications are needed before coding begins: (1) which site ordering is used for computation, (2) removal of an unexplained metric field, and (3) one additional test for statistical equivariance validation.

---

## Strongest Repaired Decisions

1. **Terminology repair**: `environment-projected conditional vector` replaces `conditional wave function`, stripping away measurement/state-collapse connotations. The new term is a purely mathematical description of a vector slice.
2. **`environment_correlated_conditional_update`**: Removes the "monitoring" framing entirely. The four-tier descriptive scale (`not_assessed` → `no_clear_update` → `weak_update_enhancement` → `candidate_update_enhancement`) is honest about uncertainty.
3. **Exact time-step convention**: The `ψ[k]` → rates → jump → `ψ[k+1]` sequence is now unambiguous and implementable.
4. **Locked rate orientation**: `rates[dest][src]` with explicit convention and three mandatory orientation tests (`TestRateOrientationDestinationSource`, `TestFakeRateTrajectoryKnownDestination`, `TestBridgeRatesMatch0001Rates`) makes orientation bugs mechanically detectable.
5. **Discrete-time thinning safeguards**: `max_lambda_dt` thresholds (0.1 warning, 0.5 inconclusive) are numerically conservative and correctly framed as reliability diagnostics, not physics criteria.
6. **Ratio nullification rules**: `conditional_update_ratio` is explicitly gated on minimum event counts and emits structured `null` with reason when unavailable. It cannot be manufactured from noise.
7. **Debt-status gating**: Debt only advances on `bridge_audit_completed`. Inconclusive or failed runs do not overstate progress.
8. **Comprehensive limitations list**: Eight explicit limitations, including finite-size and finite-sample noise warnings, make overclaiming difficult.
9. **Two-pass architecture**: Clean separation preserves 0001 semantics and makes backward compatibility testable.
10. **Deterministic RNG**: Reproducible trajectories make tests deterministic and CI-stable.

---

## Remaining Blockers

**None.** All prior critical blockers have been resolved. The items below are clarifications, not blockers.

---

## High-Priority Repairs Before Coding

1. **Clarify canonical vs. requested ordering for computation**
   The plan reports both `subsystem_sites_requested` and `subsystem_sites_canonical` but does not state which ordering is used to construct the conditional-vector basis. **The computation must use the user-specified (requested) ordering.** Canonical ordering should only be reported for transparency, never used internally. Add explicit rule: "Conditional-vector basis ordering follows `subsystem_sites_requested` exactly. Canonical ordering is reported for audit transparency only."

2. **Remove or explain `fixed_environment_reference_mean_drop`**
   This field appears in the metrics schema example but is not defined anywhere in the plan. It reads like scope creep. **Remove it from the schema.** If it is intended as a better null model, defer to a future ticket.

3. **Add `TestEmpiricalEquivarianceAgainstPsiSquared`**
   The existing `TestEmpiricalEquivarianceSampledStepsOnly` tests the computation formula, not the statistical agreement. Add a test that runs multiple trajectories and verifies the empirical frequency of each configuration at the final sampled step is within a loose L1 distance (e.g., < 0.5 for 200 trajectories, 64 states) of `|ψ(T)|²`. This catches drift in the sampler.

---

## Medium-Priority Repairs

1. **Clarify that `environment_jump_transitions` includes boundary-crossing jumps**
   The classification rule should be explicit: "A transition is an environment jump if `Q_B[k] != Q_B[k+1]`, regardless of whether `Q_A` also changed." This prevents misclassification debates.

2. **Add `TestBoundaryCrossingJumpClassification`**
   The plan has `TestBoundaryCrossingJumpBothSidesChanged` but it should explicitly verify that such a transition is counted in `environment_jump_transitions`, `any_jump_transitions`, and also `subsystem_jump_transitions` if `Q_A` changed.

3. **Document the rate formula for pure states**
   The plan says "reuse the same Bell current/rate definitions as 0001." Add one sentence: "For pure-state trajectory sampling, rates are computed from the Bell current using `|ψ_q|²` in place of `ρ_{qq}`, which is exact because `ρ = |ψ⟩⟨ψ|`."

4. **Add `TestConditionalVectorDimension`**
   Verify that the conditional vector has length `2^|A|` and that the full wavefunction can be reconstructed from all conditional vectors for a fixed time.

5. **Warn if `sample_every_steps` is large**
   If `sample_every_steps > 1`, jumps between samples are missed in fidelity classification. Add a warning: `sparse_sampling_warning`.

---

## Low-Priority Polish

1. **Schema version**: Bump unconditionally to `bell_mipt_report_v0_2a` regardless of bridge status, since the code now supports it.
2. **Bridge section when disabled**: Include it as `{"enabled": false, "bridge_status": "bridge_disabled"}` for forward compatibility.
3. **`candidate_update_enhancement`**: Slightly loaded but acceptable with disclaimer. Could be `candidate_conditional_update` to remove "enhancement" entirely.
4. **Debt string length**: `partially_paid_environment_projected_conditional_vector_toy_only` is 67 characters. Acceptable but unwieldy. Fine as-is.
5. **Test naming**: `TestConditionalProjectorNoResign` is unclear. Rename to `TestConditionalProjectionPreservesPhase` or similar.

---

## Answer Table

| Question | Answer | Severity | Repair |
|----------|--------|----------|--------|
| 1. Remaining conceptual blockers? | None. The conditional-vector construction is mathematically well-defined as a slice of the amplitude vector. No measurement postulate is invoked. | none | None |
| 2. `environment-projected conditional vector` safer? | Yes, significantly. It describes a mathematical projection operation, not a physical state or collapse. | none | None |
| 3. `environment_correlated_conditional_update` too close? | Borderline but acceptable. "Update" is less loaded than "signal" or "monitoring." The four-tier scale and explicit disclaimer make it safe. | low | Could rename `candidate_update_enhancement` → `candidate_conditional_update` |
| 4. Risk implying Bell jumps = measurements? | Minimal. The plan explicitly forbids this claim 8 times. The comparison of env-jump vs non-env-jump fidelity drops is framed as a descriptive correlation, not a measurement model. | low | None |
| 5. Debt-status overstate payment? | No. The rule gates debt advancement on `bridge_audit_completed` only. Inconclusive/failed runs retain old debt. The "partially paid" string is verbose but accurate for a toy diagnostic. | none | None |
| 6. Time-step convention correct? | Yes. Rates from `ψ[k]`, jump decision, then `ψ[k+1]` from RK4. This is the standard discrete-time thinning approach. | none | None |
| 7. Jump sampling before/after ψ[k+1]? | The proposed convention suffices. The RK4 evolution is deterministic and independent of the jump. The jump only updates Q, not ψ. | none | None |
| 8. Acceptable not to sample at RK4 sub-stages? | Yes for 0002A toy scope. The plan documents this as a limitation. | none | None |
| 9. `rates[dest][src]` unambiguous? | Yes, with the explicit convention and three mandatory orientation tests. | none | None |
| 10. Mandatory rate-orientation tests sufficient? | Yes. `TestRateOrientationDestinationSource`, `TestFakeRateTrajectoryKnownDestination`, and `TestBridgeRatesMatch0001Rates` form a defense-in-depth against orientation bugs. | none | None |
| 11. No additional JW sign correct? | Yes. In the occupation-number basis, the wavefunction amplitudes are scalars indexed by bit patterns. Extracting a slice does not introduce new signs. All phases are inherited from the full Hamiltonian diagonalization. | none | None |
| 12. Non-contiguous partition tests sufficient? | Yes. `TestNonContiguousPartition` and `TestSplitCombineRoundTripAllConfigs` catch indexing errors. Sign errors are not possible in this basis. | none | None |
| 13. Canonical sorting safe? | **Unsafe if used for computation.** User order defines the subsystem basis. The plan must explicitly state that computation uses requested order, and canonical is reported only. | high | Add rule: computation uses requested order |
| 14. Report both orderings? | Yes, transparent and useful for audit. | none | None |
| 15. Need `basis_index_to_config` abstraction? | No. The existing 0001 basis is presumably already a bitmask or index. The plan reuses it. | none | None |
| 16. `max_lambda_dt` thresholds reasonable? | Yes. 0.1 (warning) and 0.5 (inconclusive) are conservative for first-order thinning. | none | None |
| 17. Should >0.1 already be inconclusive? | No. At 0.1, the probability of missing a double jump is ~0.005. Warning is sufficient. | none | None |
| 18. Norm-failure thresholds reasonable? | Yes. 10% inconclusive, 50% failed are sensible heuristics for a toy. | none | None |
| 19. `ConditionalNormFloor = 1e-14` acceptable? | Yes. For Hilbert spaces up to ~1024 dimensions, typical amplitudes are ~0.03. 1e-14 is 28 orders of magnitude below typical, effectively zero. | none | None |
| 20. Near-zero `|ψ_q|²` handling? | Correctly handled by `near_zero_current_configuration_probability` warning. If the actual Q has negligible probability, the conditional vector is unreliable. This is honest. | none | None |
| 21. Fidelity drop enough? | Yes. Fidelity is the natural metric for state-vector comparison. Trace distance adds complexity without additional insight for this toy. | none | None |
| 22. `fixed_environment_reference_mean_drop` useful? | No, it is scope creep. It appears in the schema example but is undefined in the plan. | medium | Remove from schema |
| 23. Ratio event-count rule sufficient? | Yes. 10 events is a reasonable minimum for a toy diagnostic. The structured `null` with reason prevents misinterpretation. | none | None |
| 24. Denominator: without-env or no-jump? | "Without environment jumps" is correct. It includes no-jump and subsystem-only-jump transitions, controlling for smooth evolution while allowing subsystem dynamics. | none | None |
| 25. Exclude boundary-crossing from env jumps? | No. Boundary-crossing jumps where both A and B change are still environment jumps. The classification should be: `env_jump = (Q_B changed)`. | none | Clarify in plan |
| 26. Bridge section when disabled? | Present with `enabled: false`. Better for forward compatibility and report uniformity. | none | None |
| 27. Schema version bump only when enabled? | No. Bump unconditionally. The code version is the code version. | low | Bump always |
| 28. `bridge_audit_*` names safe? | Yes. "Completed/failed/inconclusive" describes audit execution, not physics success. | none | None |
| 29. `candidate_update_enhancement` safe? | Borderline. "Enhancement" implies improvement. `candidate_conditional_update` is safer. | low | Rename to remove "enhancement" |
| 30. Warnings and limitations sufficient? | Yes. The 8 limitations and 7 warning types comprehensively fence off overclaiming. | none | None |
| 31. Mandatory tests sufficient? | Almost. One test is missing: statistical validation that empirical trajectory distribution matches `|ψ|²`. | high | Add `TestEmpiricalEquivarianceAgainstPsiSquared` |
| 32. Most likely missing test? | Statistical validation of empirical equivariance against `|ψ(t)|²` with loose tolerance. The existing test only checks the formula, not the agreement. | high | Add test |
| 33. Most overkill test? | None are overkill. The list is lean for the complexity of the task. | none | None |
| 34. Fake RNGs vs statistical tolerances? | Yes, the plan uses deterministic RNG. All tests should use fixed seeds or fake RNGs. Statistical tolerances should only be used in the new equivariance validation test. | none | None |
| 35. Semantic vs byte-for-byte regression? | Semantic metrics. Byte-for-byte is brittle against schema additions. | none | None |

---

## Final Repaired Implementation Checklist

The coding agent must verify each item before declaring implementation complete:

### Config and validation
- [ ] `BridgeConfig` struct added with all fields
- [ ] Partition validation: no overlap, no missing sites, completeness check
- [ ] `trajectories > 0` and `sample_every_steps > 0` enforced when enabled
- [ ] Bridge omitted or `enabled=false` preserves 0001 behavior exactly

### Time-step and sampling
- [ ] Rates computed from `ψ[k]` before jump decision for interval `k → k+1`
- [ ] `rates[dest][src] = σ(dest ← src)` convention documented in code comments
- [ ] Deterministic RNG seeded from config
- [ ] `p_jump = 1 - exp(-λ*dt)` with at most one jump per dt
- [ ] `max_lambda_dt` computed and checked against thresholds
- [ ] `ψ[k+1]` obtained from existing RK4 evolution (Pass 1 snapshots)

### Conditional vector
- [ ] `ψ_A(a,t | b) = Ψ(full_config(a,b), t)` using requested site ordering
- [ ] Conditional vector normalized if norm ≥ `ConditionalNormFloor`
- [ ] Norm failures counted and tracked
- [ ] `CombineConfig(SplitConfig(q)) == q` invariant holds for all q
- [ ] Non-contiguous partitions supported and tested

### Metrics
- [ ] Fidelity drops computed between consecutive conditional vectors
- [ ] Transitions classified by which partitions changed
- [ ] `conditional_update_ratio` only reported when env events ≥ 10, non-env events ≥ 10, and denominator > 1e-14
- [ ] Unavailable ratios serialize as JSON `null` with `conditional_update_ratio_status`
- [ ] `fixed_environment_reference_mean_drop` **removed** from schema

### Status and warnings
- [ ] `goal_status` independent of `bridge_status`
- [ ] `bridge_status` computed from implementation success, not from ratio value
- [ ] Warnings emitted for: large lambda dt, severe lambda dt, low events, norm failures, finite sample, finite size, near-zero probability
- [ ] `environment_correlated_conditional_update` never set based on ratio alone

### Report
- [ ] `schema_version` set to `bell_mipt_report_v0_2a`
- [ ] Bridge section present when disabled with `enabled: false`
- [ ] Both requested and canonical site lists reported
- [ ] All 8 limitations included in report text
- [ ] Forbidden-language audit scans fully assembled Markdown and JSON

### Tests (all must pass)
- [ ] `TestTimeStepAlignmentUsesPsiKForIntervalK`
- [ ] `TestRateOrientationDestinationSource`
- [ ] `TestFakeRateTrajectoryKnownDestination`
- [ ] `TestBridgeRatesMatch0001Rates`
- [ ] `TestSplitCombineRoundTripAllConfigs`
- [ ] `TestNonContiguousPartition`
- [ ] `TestConditionalProjectionPreservesPhase` (renamed from NoResign)
- [ ] `TestBoundaryCrossingJumpBothSidesChanged`
- [ ] `TestBoundaryCrossingJumpClassification` (new)
- [ ] `TestZeroRateNoJumpInconclusive`
- [ ] `TestNearZeroCurrentConfigurationProbability`
- [ ] `TestConditionalNormFailureThresholds`
- [ ] `TestConditionalUpdateRatioNullWhenLowEvents`
- [ ] `TestMaxLambdaDTWarningAndInconclusive`
- [ ] `TestForbiddenAuditScansFullJSONAndMarkdown`
- [ ] `TestBridgeDisabledPreserves0001SemanticMetrics`
- [ ] `TestEmpiricalEquivarianceSampledStepsOnly`
- [ ] `TestEmpiricalEquivarianceAgainstPsiSquared` (new)
- [ ] `TestConditionalVectorDimension` (new)

### Validation commands
- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] `go run ./cmd/bellmipt --out /tmp/bellmipt-default` (bridge disabled)
- [ ] `go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge` (bridge enabled)

### Debt update
- [ ] Debt only updated if `bridge_status == "bridge_audit_completed"`
- [ ] Otherwise, old debt retained

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

**Proceed after minor plan edits.**

The three required edits are:
1. Add explicit rule: computation uses `subsystem_sites_requested` ordering; canonical is reported only.
2. Remove `fixed_environment_reference_mean_drop` from the metrics schema.
3. Add `TestEmpiricalEquivarianceAgainstPsiSquared` and `TestBoundaryCrossingJumpClassification` to the mandatory test list.

After these edits, the plan is implementation-ready, auditable, numerically honest, and EBP-safe. The risk of manufacturing a fake Bell-MIPT bridge result is minimized by: deterministic ratio nullification, status independence, comprehensive warnings, explicit limitations, and forbidden-language auditing.
