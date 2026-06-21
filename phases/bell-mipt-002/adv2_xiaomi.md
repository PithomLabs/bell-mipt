# Final Adversarial Review: Repaired BELL-MIPT-0002A Plan

---

## Executive Verdict

```
ready_for_implementation_with_minor_repairs
```

---

## Strongest Repaired Decisions

1. **"Environment-projected conditional vector" replacing "conditional wave function."** This is the single most important terminological repair. The computation is a linear projection `ψ_A(a,t|b) = Ψ(full_config(a,b), t)`, which is a well-defined algebraic operation in the occupation basis. Calling it a "conditional wave function" invokes Bohmian/Bell guidance-equation conditioning and QFT ordering concerns that do not apply here. The new name is honest about what the code actually computes.

2. **No additional JW sign during projection.** This is correct for the occupation-number basis. The full wavefunction amplitude `Ψ(q,t)` already carries all Jordan-Wigner phases from the Hamiltonian evolution. Extracting `Ψ(a,b₀,t)` for a fixed `b₀` simply reads the amplitude at the corresponding index — no re-signing needed. The prior review's blocker about JW signs under projection is resolved by this clarification.

3. **max_lambda_dt hard threshold at 0.5 with warning at 0.1.** This replaces the previous "log a warning" with a mechanically testable status gate. The thresholds are physically motivated: at `λdt = 0.5`, the multi-jump probability within one step is `1 - (1+λdt)exp(-λdt) ≈ 0.09`, meaning ~9% of steps would incorrectly be single-jump. At 0.1, the multi-jump probability is ~0.5%, which is acceptable for a toy.

4. **conditional_update_ratio never determines bridge_status.** This was the right call from the prior review and is preserved. Status is determined by implementation-level success (norm failures, lambda_dt, numerical stability), not by whether a physics-like signal appeared.

5. **Two mandatory canonical site lists (requested and canonical).** This provides auditability: if the user specifies `[5,0,2]` and the code sorts to `[0,2,5]`, both are visible in the report. This catches ordering-related bugs in downstream analysis.

6. **Structured warnings with machine-readable codes.** The `BridgeWarning{Code, Message}` type enables programmatic filtering and prevents consumers from having to parse warning text strings.

7. **Debt status conditional on bridge_status.** The rule that debt advances only when `bridge_audit_completed` prevents the plan from claiming partial credit for failed or inconclusive runs.

---

## Remaining Blockers

**There are no remaining blockers.** The prior review's blocker (JW sign handling under projection) has been resolved by the explicit "no additional sign during projection" statement, which is correct for the occupation-number basis. All other prior blockers (time-step alignment, rate orientation) have been addressed with specific conventions and mandatory tests.

---

## High-Priority Repairs Before Coding

### H1: Add near-zero |ψ_q|² handling specification

The plan mentions `near_zero_current_configuration_probability` as a warning code but does not specify the exact behavior. The implementation must define:

- **Threshold:** If `|ψ_q|² < 1e-30` for the current actual configuration `q`, set all outgoing rates to zero, do not sample a jump (stay at `q`), and emit a warning with code `near_zero_current_configuration_probability`.
- **Rationale:** This prevents division-by-zero in `σ(n←q) = max(J_nq, 0) / |ψ_q|²` and prevents `lambda` from exploding.

Without this, a trajectory that wanders into a configuration with negligible probability will produce infinite rates and crash or silently corrupt the output.

### H2: Specify basis-index-to-config mapping explicitly

The plan says basis states are uint64 bitstrings but does not state whether `basis[i] == i` (identity mapping) or whether basis is an arbitrary ordered slice. The rate matrix `rates[dest][src]` uses integer indices into the basis slice, not raw bitmask values. If `basis[5] = 0b11010` then `rates[5][3]` is the rate from `basis[3]` to `basis[5]`. The trajectory sampler stores `Q[k]` as either the raw bitmask or the basis index — this must be consistent throughout. Define:

```
Q_index[k] = integer index into basis slice
Q_config[k] = basis[Q_index[k]] = raw bitmask
```

The trajectory sampler must work with `Q_index` for rate lookups and `Q_config` for partition splitting.

### H3: Define the "fixed_environment_reference_mean_drop" metric precisely

The report schema includes `fixed_environment_reference_mean_drop` but the repaired plan summary does not define how it is computed. The prior review recommended a Schrödinger-only baseline trajectory where Q(t) = Q(0) for all t. Define:

- Run one additional trajectory where no jumps ever occur (Q remains at its initial sampled value for all time steps).
- At each sampled step, compute the conditional vector using the actual Q_B(0) (which never changes) and the evolved ψ[k].
- Compute fidelity drops between consecutive conditional vectors.
- Report the mean as `fixed_environment_reference_mean_drop`.

This isolates the smooth-Schrödinger component of conditional-vector evolution from all jump effects.

---

## Medium-Priority Repairs

### M1: Add explicit test for rate accessor consistency with BELL-MIPT-0001

The mandatory test `TestBridgeRatesMatch0001Rates` should verify that the rate values used by the trajectory sampler are identical to those computed by the existing 0001 rate code, not just numerically close. If the trajectory sampler uses a "rate-row accessor" that computes only the column for a single source state, it must produce bit-identical results (within floating-point ordering) to the corresponding column of the full rate matrix.

### M2: Clarify whether fidelity drops at environment jumps include or exclude boundary-crossing jumps

The plan defines `environment_jump_transitions` as transitions where Q_B changed. When a single jump changes both Q_A and Q_B (boundary-crossing), this is counted as an environment jump. The fidelity drop in this case reflects both subsystem and environment changes. The plan should explicitly note this and report the count of boundary-crossing jumps as a separate diagnostic so consumers can disentangle the two effects.

### M3: Add test for multiple consecutive jumps

Add a test with a deliberately high-rate configuration (e.g., large `λ`) where multiple jumps occur over consecutive time steps. Verify that the trajectory correctly records each jump, that environment/subsystem classification is correct at each step, and that fidelity drops are computed between consecutive conditional vectors (not between the first and last).

### M4: Verify that empirical trajectory equivariance uses sampled steps only

The test `TestEmpiricalEquivarianceSampledStepsOnly` should verify that the equivariance L1 computation does not accidentally use all micro-steps (which would overweight early time points where ψ changes slowly). The empirical distribution should be accumulated only at `k % sample_every_steps == 0`.

---

## Low-Priority Polish

### L1: Consider renaming `fixed_environment_reference_mean_drop` to `schrodinger_only_baseline_mean_drop`

The current name is descriptive but ambiguous — "fixed environment" could mean "fixed environment configuration" or "fixed environment Hamiltonian." The Schrödinger-only phrasing makes the purpose immediately clear.

### L2: Add `boundary_crossing_jumps` count to metrics

Include in the bridge metrics:

```json
"boundary_crossing_jumps": 0
```

This is the count of transitions where both `envJumped` and `subsysJumped` are true. It aids interpretation of fidelity-drop diagnostics.

### L3: Include `effective_sample_count` in empirical equivariance reporting

Report how many trajectory-timepoint pairs contributed to each equivariance L1 computation. This helps consumers assess statistical significance:

```json
"empirical_equivariance_effective_samples": 200
```

---

## Answer Table

### A. Remaining Blockers

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 1 | Any remaining conceptual blockers? | No. The JW-sign blocker is resolved. Time-step alignment is pinned. Rate orientation is locked. All conceptual issues from the prior review have been addressed. | none | — |
| 2 | Is "environment-projected conditional vector" safer than "conditional wave function"? | Yes, significantly. The computation is a linear projection in the occupation basis, which is well-defined without invoking guidance equations, QFT ordering, or Bohmian conditioning. The new name accurately describes what the code does. | none | — |
| 3 | Is "environment_correlated_conditional_update" still too close to measurement/MIPT? | No. It describes a correlation between environment jumps and conditional-vector changes without invoking measurement, monitoring, or MIPT. The word "update" refers to vector value changes, not wavefunction collapse. Safe. | none | — |
| 4 | Does the plan still risk implying Bell jumps are measurements? | Minimally. The structure (stochastic jumps → conditional-state changes → fidelity diagnostics) superficially resembles measurement backaction analysis. However, the plan's explicit limitations, non-promotion rules, debt-status constraints, and terminology choices collectively prevent this implication from being actionable. The risk is cosmetic, not structural. | low | No repair needed; existing guardrails are sufficient. |
| 5 | Does repaired debt-status rule overstate? | No. The conditional rule (`bridge_audit_completed` required to advance) and the explicit preservation of `needObstruction: "bell_jumps_are_not_measurements"` are correct. An inconclusive run does not advance debt. | none | — |

### B. Time-Step and Rate Correctness

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 6 | Is repaired time-step convention correct and implementable? | Yes. The convention "rates from ψ[k] determine Q[k] → Q[k+1]; conditional vector at k uses (ψ[k], Q_B[k])" is internally consistent and matches a forward-Euler interpretation of the jump process. It is implementable with a single pass over time steps. | none | — |
| 7 | Should jump sampling happen before or after ψ[k+1] is computed? | The proposed convention suffices. Jump sampling at step k uses ψ[k] (before RK4 evolution to ψ[k+1]). This is a forward-rate convention: the rate is evaluated at the current state, and the transition is sampled. ψ[k+1] is then used at step k+1 for the next conditional vector and the next rate computation. This is consistent and correct for discrete-time thinning. | none | — |
| 8 | Is it acceptable not to sample at RK4 sub-stages? | Yes, for the toy scope. The RK4 sub-stages are internal to the ψ evolution integrator and do not correspond to physical time points in the jump process. Sampling jumps only at full step boundaries is the standard approach for discrete-time thinning. The error introduced is O(dt²), which is the same order as the thinning approximation itself. | none | — |
| 9 | Is rates[dest][src] unambiguous enough? | Yes, with the mandatory test `TestRateOrientationDestinationSource`. The convention `rates[dest][src] = σ(dest <- src)` maps directly to the mathematical definition. The only risk is a transposition bug, which the test catches. | none | — |
| 10 | Are mandatory rate-orientation tests sufficient? | They are necessary and should be sufficient if `TestRateOrientationDestinationSource` verifies a hand-calculated case and `TestBridgeRatesMatch0001Rates` verifies consistency with the existing code. These two tests together catch both orientation bugs and accessor-vs-matrix divergence. | none | — |

### C. Conditional-Vector Construction

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 11 | Does "no additional JW sign" correctly handle the occupation-basis toy? | Yes. In the occupation-number basis with Jordan-Wigner encoding, the full Hamiltonian matrix H already incorporates all JW signs. The time-evolved wavefunction ψ(q,t) carries these signs in its complex amplitudes. Projecting onto an environment branch b₀ by reading Ψ(a,b₀,t) for each subsystem configuration a simply indexes into the wavefunction vector. No sign manipulation is needed because no operator is being applied — it is a passive read of an existing amplitude. | none | — |
| 12 | Are non-contiguous partition tests sufficient? | The test `TestNonContiguousPartition` with A=[0,2,5], B=[1,3,4] is a good starting point. It should verify: (a) SplitConfig/CombineConfig round-trip for all 64 configurations, (b) conditional vector dimension is 2³=8, (c) conditional fidelity of identical states is 1.0. This catches both bit-masking and index-mapping errors for non-contiguous sites. | none | — |
| 13 | Is canonical sorting of site lists safe? | Yes, provided the conditional vector basis ordering is defined by the sorted site indices. If A=[5,0,2] is sorted to [0,2,5], then the conditional vector index a=0b101 means site 0 is occupied, site 2 is unoccupied, site 5 is occupied. This is deterministic and reproducible. Preserving user order would create ambiguity when two users specify the same partition in different orders. | none | — |
| 14 | Should report include both orderings? | Yes. This is already in the plan (`subsystem_sites_requested` and `subsystem_sites_canonical`). This provides auditability without adding implementation complexity. | none | — |
| 15 | Does plan need explicit basis_index_to_config abstraction? | Yes, explicitly. The trajectory sampler must distinguish between the integer index into the basis slice and the raw bitmask value. Define `Q_index` and `Q_config` as separate variables, with `Q_config = basis[Q_index]`. This prevents the class of bugs where the sampler accidentally uses a bitmask as a matrix index or vice versa. | **high** | Define Q_index/Q_config mapping explicitly in the plan (see H2). |

### D. Numerical Reliability

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 16 | Are max_lambda_dt thresholds reasonable? | Yes. At λdt=0.1, the Poisson multi-jump probability is ~0.5%, which is negligible for a toy. At λdt=0.5, it is ~9%, which is non-negligible. The warning/inconclusive boundary at these values is physically motivated. | none | — |
| 17 | Should max_lambda_dt > 0.1 already make audit inconclusive? | No. A warning at 0.1 is sufficient because the multi-jump error is still small. The 0.5 threshold for inconclusive is more appropriate. If the plan wanted to be more conservative, 0.2 would also be reasonable, but 0.1 as the inconclusive threshold would make too many runs inconclusive for marginal numerical reasons. | none | — |
| 18 | Are norm-failure thresholds reasonable? | `NormFailureInconclusiveRate = 0.10` means 10% of conditional-vector computations failing norm threshold triggers inconclusive. This is reasonable: if 1 in 10 samples fails, the fidelity-drop diagnostics are based on a biased subset. `NormFailureFailedRate = 0.50` as the "failed" threshold is generous — a 50% failure rate means the bridge audit is essentially nonfunctional. | none | — |
| 19 | Is ConditionalNormFloor = 1e-14 acceptable? | Yes. For double-precision arithmetic with dim ≤ 64, accumulated rounding error from summing ~8 amplitudes (for a 3-site subsystem) is ~8 × 2.2e-16 ≈ 1.8e-15. The floor at 1e-14 is ~5× above the noise floor, meaning norm failures will only trigger for genuinely negligible branches, not numerical artifacts. | none | — |
| 20 | How should near-zero |ψ_q|² be handled? | Define threshold `ProbFloor = 1e-30`. If `|ψ_q|² < ProbFloor` for the current actual configuration, set all outgoing rates to zero, do not sample a jump, emit warning code `near_zero_current_configuration_probability`, and continue the trajectory. This prevents division-by-zero in rate computation. | **high** | Add explicit specification (see H1). |

### E. Metrics and Interpretation

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 21 | Is fidelity drop enough for 0002A? | Yes for a first diagnostic. Fidelity is the standard overlap metric for pure states and is directly interpretable: `drop = 0` means no change, `drop = 1` means orthogonal. Trace distance would add marginal value for pure states (it is a monotonic function of fidelity for two-level systems and closely related for higher dimensions). Adding it now risks scope creep without material diagnostic improvement. | none | Defer trace distance to a future ticket. |
| 22 | Is fixed_environment_reference_mean_drop useful or scope creep? | Useful. It provides the Schrödinger-only baseline for conditional-vector evolution, which is the correct null for the fidelity-drop comparison. The prior review identified this as a high-priority repair, and including one extra no-jump trajectory is negligible computational cost. | none | — |
| 23 | Is event-count rule sufficient? | Yes. The minimum of 10 events per category ensures the mean has minimal statistical meaning (though still noisy). For 200 trajectories of 1000 steps each with typical jump rates, achieving 10+ events per category is realistic for a 6-site system. | none | — |
| 24 | Should ratio denominator use "without environment jumps" or only "no jump"? | Use `mean_fidelity_drop_without_environment_jumps` (which includes both no-jump and subsystem-only jumps) as the primary denominator. This compares "conditional-vector evolution during environment jumps" against "conditional-vector evolution during everything else." Additionally, report `mean_fidelity_drop_no_jump` as a separate metric for consumers who want the pure smooth-evolution baseline. The `fixed_environment_reference_mean_drop` serves as the cleanest Schrödinger-only null. | none | — |
| 25 | Should environment_jump_transitions exclude boundary-crossing? | No. A boundary-crossing jump where both Q_A and Q_B change does change the environment configuration, so it correctly qualifies as an environment jump. However, report `boundary_crossing_jumps` as a separate count so consumers can filter them if desired. | **medium** | Add boundary-crossing jump count to metrics (see L2). |

### F. Report/Schema/EBP Safety

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 26 | Bridge section omitted when disabled? | Omit. Existing consumers expect the old schema. Including an empty bridge section when disabled is unnecessary clutter and risks breaking downstream parsers that check for key presence. | none | Already in plan. |
| 27 | Schema version bump only when bridge enabled? | Yes. When bridge is disabled, use `bell_mipt_report_v0` for backward compatibility. When enabled, use `bell_mipt_report_v0_2a`. This is clean and unambiguous. | none | — |
| 28 | Are bridge_audit_* status names safe? | Yes. `bridge_audit_completed` correctly avoids "passed" (which implies a physics result). `bridge_audit_failed` and `bridge_audit_inconclusive` are implementation-level judgments, not physics claims. Safe. | none | — |
| 29 | Are *update_enhancement names safe? | Yes. "Enhancement" refers to an observed difference in conditional-vector fidelity drops between two categories of transitions. It does not invoke measurement, monitoring, or MIPT. "Candidate update enhancement" means "there is a measurable difference worth noting," not "we found a phase transition." | none | — |
| 30 | Are warnings and limitations sufficient? | Yes. The 8 warning codes cover all known numerical risks. The 9 limitation sentences explicitly reject every forbidden claim. The combination of structured warnings (machine-readable) and limitation text (human-readable) provides defense in depth against overclaiming. | none | — |

### G. Test Adequacy

| # | Question | Answer | Severity | Repair |
|---|---|---|---|---|
| 31 | Are mandatory tests sufficient to catch dangerous bugs? | Nearly. The 16 mandatory tests cover the critical paths: time-step alignment, rate orientation, round-trip config splitting, JW sign preservation, boundary-crossing, norm failures, lambda_dt thresholds, forbidden language, and backward compatibility. The one gap is the near-zero |ψ_q|² handling, which has a warning code but no mandatory test. | **medium** | Add `TestNearZeroActualConfigurationProbability` to mandatory tests (already listed; ensure it is implemented). |
| 32 | Which test is most likely still missing? | A test that verifies the trajectory sampler produces the correct **jump frequency distribution** for a known rate matrix. The existing `TestFakeRateTrajectoryKnownDestination` tests a single jump. A companion test should verify that over 1000+ steps with known constant rates, the empirical jump frequency matches `1 - exp(-λdt)` within tolerance. This catches cumulative probability-computation errors. | **medium** | Add `TestJumpFrequencyMatchesExpectedProbability`. |
| 33 | Which test is overkill? | `TestEmpiricalEquivarianceSampledStepsOnly` is useful but low-value compared to the others. It tests a data-accumulation detail that is unlikely to produce a subtle bug — if the code accumulates at every step instead of sampled steps, the L1 value will be wrong but not dangerously so, and it would be caught by comparing against a hand-calculated baseline. Not harmful to include, but not a priority. | low | Keep but do not prioritize. |
| 34 | Should stochastic tests use fake RNGs? | Yes, for correctness tests. The deterministic tests (`TestFakeRateTrajectoryKnownDestination`, `TestTimeStepAlignmentUsesPsiKForIntervalK`) must use a fake RNG that returns predetermined values. Statistical distribution tests (`TestJumpFrequencyMatchesExpectedProbability`, initial-config sampling) must use real RNG with generous tolerances and be documented as potentially flaky. | none | Already implied by plan; ensure implementation follows this split. |
| 35 | Bridge-disabled regression: semantic or byte-for-byte? | Semantic field comparison. Byte-for-byte comparison breaks on: (a) JSON key ordering, (b) floating-point formatting (e.g., `0.0` vs `0.00`), (c) platform-dependent behavior. Compare: goal_status, all check booleans, all metric values within `1e-10` tolerance, debt_status strings. | none | — |

---

## Final Repaired Implementation Checklist

The coding agent must follow this checklist in order:

### Phase 0: Pre-implementation
- [ ] Confirm all mandatory tests are defined before writing production code.
- [ ] Confirm Q_index vs Q_config mapping is specified.
- [ ] Confirm near-zero |ψ_q|² threshold (ProbFloor = 1e-30) and behavior are specified.

### Phase 1: Config and Partition (no behavior change)
- [ ] Add `BridgeConfig` struct with all fields from the report schema.
- [ ] Add validation: partition covers all sites, no overlap, no out-of-range, trajectories > 0, sample_every_steps >= 1.
- [ ] Bridge omitted or disabled preserves 0001 behavior exactly.
- [ ] Tests: `TestBridgeDisabledPreserves0001SemanticMetrics`, partition validation pass/fail cases.

### Phase 2: Site Split / Config Mapping
- [ ] Implement `SitePartition` with sorted canonical indices.
- [ ] Implement `SplitConfig`, `CombineConfig`, `SubsystemIndex`, `EnvironmentIndex`.
- [ ] Tests: `TestSplitCombineRoundTripAllConfigs`, `TestNonContiguousPartition`.

### Phase 3: Conditional Vector
- [ ] Implement `ComputeConditionalVector` with ConditionalNormFloor = 1e-14.
- [ ] Implement `ConditionalFidelity`.
- [ ] Tests: `TestConditionalProjectorNoResign`, `TestConditionalNormFailureThresholds`, identity fidelity, orthogonal fidelity.

### Phase 4: Trajectory Sampler
- [ ] Implement `SampleInitialConfig` from |ψ₀|².
- [ ] Implement single-step jump sampling with `p_jump = 1 - exp(-λdt)`.
- [ ] Implement rate lookup: `rates[dest][Q_index[k]]` for outgoing rates from current config.
- [ ] Implement near-zero |ψ_q|² guard.
- [ ] Implement `max_lambda_dt` tracking.
- [ ] Implement full trajectory runner storing Q_index, Q_config, jump flags at sampled steps.
- [ ] Tests: `TestRateOrientationDestinationSource`, `TestFakeRateTrajectoryKnownDestination`, `TestBridgeRatesMatch0001Rates`, `TestZeroRateNoJumpInconclusive`, `TestNearZeroCurrentConfigurationProbability`, `TestJumpFrequencyMatchesExpectedProbability`, `TestTimeStepAlignmentUsesPsiKForIntervalK`.

### Phase 5: Bridge Orchestrator and Metrics
- [ ] Implement `BridgeMetrics` accumulator.
- [ ] Implement multi-trajectory orchestrator.
- [ ] Implement fidelity-drop binning by transition type.
- [ ] Implement conditional_update_ratio with event-count gating.
- [ ] Implement Schrödinger-only baseline trajectory.
- [ ] Implement empirical trajectory equivariance at sampled steps only.
- [ ] Implement `ClassifyUpdate` logic.
- [ ] Implement bridge_status determination.
- [ ] Tests: `TestConditionalUpdateRatioNullWhenLowEvents`, `TestMaxLambdaDTWarningAndInconclusive`, `TestBoundaryCrossingJumpBothSidesChanged`, `TestEmpiricalEquivarianceSampledStepsOnly`.

### Phase 6: Report and Run Integration
- [ ] Modify `run.go`: two-pass architecture, store ψ snapshots, call bridge when enabled.
- [ ] Modify `report.go`: add bridge section with schema version bump.
- [ ] Modify `markdown.go`: add bridge markdown with limitations.
- [ ] Update `forbidden.go`: audit full report, allowlist for literal limitation sentences.
- [ ] Implement structured warnings.
- [ ] Implement conditional debt-status update.
- [ ] Tests: `TestForbiddenAuditScansFullJSONAndMarkdown`, backward-compatibility regression.

### Phase 7: Validation
- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] Default run (bridge disabled) — verify 0001 metrics unchanged.
- [ ] Bridge-enabled run — verify report structure, metrics, warnings, limitations.

---

## EBP 2.1 Ledger

```json
{
  "needMap": "ready_for_repaired_conditional_vector_toy_attempt",
  "needInvariant": "0001_partially_paid; 0002A_empirical_trajectory_diagnostic_planned",
  "needToyCheck": "0002A_plan_repaired_and_reviewed_pending_implementation",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements_preserved",
  "needFaithfulnessReview": "0002A_pending_source_review_after_implementation",
  "promotion_status": "unpromoted_final_plan_review_only"
}
```

---

## Final Recommendation

```
Proceed after minor plan edits.
```

The three minor edits required before implementation begins:

1. **Add explicit near-zero |ψ_q|² handling specification** (ProbFloor = 1e-30, behavior: zero rates, no jump, emit warning). This is a one-paragraph addition to the plan but prevents a runtime crash.

2. **Add explicit Q_index vs Q_config mapping** to the trajectory sampler specification. This is a two-sentence clarification that prevents an entire class of matrix-index vs bitmask confusion bugs.

3. **Add `fixed_environment_reference_mean_drop` computation definition** (one no-jump trajectory, conditional vectors at sampled steps using fixed Q_B(0), mean fidelity drop). This is a one-paragraph addition that completes the metrics specification.

After these three edits, the plan is implementation-ready, auditable, numerically honest, EBP-safe, and mechanically testable. The coding agent should proceed through the phases in order, running tests at each phase gate.
