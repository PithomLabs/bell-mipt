### Executive verdict

`ready_for_implementation_with_minor_repairs`

The repaired plan is vastly superior to the initial drafts. It successfully isolates the new stochastic audit from the accepted `BELL-MIPT-0001` deterministic master-equation logic. The time-step alignment is now mathematically unambiguous, the rate orientation is explicitly locked, and the terminology has been scrubbed of measurement/MIPT overclaims.

However, there are two remaining physical/numerical traps that will manufacture a fake signal if not repaired: **boundary-crossing jumps** and **Jordan-Wigner parity contamination**. With these patched, the plan is safe to implement.

### Strongest repaired decisions

1. **Strict separation of Pass 1 and Pass 2:** Ensuring `goal_status` is untouched guarantees the `0001` equivariance invariants are protected.
2. **Rate orientation lockdown:** Explicitly defining `rates[dest][src]` prevents the most common and devastating stochastic matrix bug.
3. **Nullifying unavailable metrics:** Rejecting `conditional_update_ratio` when event counts $< 10$ stops finite-sample noise from being reported as a physics signal.
4. **Explicit time-step alignment:** Defining that interval $k \to k+1$ uses $\psi[k]$ prevents off-by-one interpolation errors with the RK4 solver.

### Remaining blockers

1. **The Boundary-Hopping Trap:** If a fermionic hopping term crosses the cut between $A$ and $B$, a single Bell jump changes *both* $Q_A$ and $Q_B$. If this is classified as an `environment_jump`, the subsystem fidelity will plummet simply because the subsystem physically lost/gained a particle. This creates a massive fake "monitoring" correlation. These jumps must be classified as `boundary_crossing_jump` and **strictly excluded** from the `conditional_update_ratio`.
2. **Jordan-Wigner Parity Contamination:** If the environment $B$ changes parity, the naive extraction of $a$ amplitudes from the global bitmask inherits relative sign flips depending on the physical site ordering. A fidelity drop between $\psi_A(t | b)$ and $\psi_A(t+dt | b')$ will be artificially inflated by this JW string artifact. The plan must either apply a parity-correction phase or explicitly log `jordan_wigner_parity_contamination_risk` as a permanent warning for this toy.

### High-priority repairs before coding

1. **Add `boundary_jump_transitions` metric:** Implement a strict mutual exclusion in jump classification: `no_jump`, `strict_environment_jump` (only B changed), `strict_subsystem_jump` (only A changed), and `boundary_crossing_jump` (both changed).
2. **Change Ratio Denominator:** The `conditional_update_ratio` should compare `mean_fidelity_drop_at_strict_environment_jumps` against `mean_fidelity_drop_no_jump`. Comparing against `without_environment_jumps` conflates smooth Schrödinger evolution with direct subsystem jumps, muddying the null model.
3. **Tighten `max_lambda_dt` thresholds:** 0.5 is far too high for a discrete Poisson approximation. Set `LambdaDTWarning = 0.05` and `LambdaDTInconclusive = 0.20`.
4. **Near-zero safeguards:** If the actual configuration probability $|\psi_q|^2 < 1\times 10^{-12}$, the code must force $\lambda = 0$ to prevent floating-point division explosions in the Bell rate formulas.

### Medium-priority repairs

1. **Drop `fixed_environment_reference_mean_drop`:** This is scope creep and adds unnecessary complexity to the state tracking. Remove it.
2. **Rename Enhancement:** Change `candidate_update_enhancement` to `candidate_update_correlation`. "Enhancement" still implies a directional physical improvement, whereas "correlation" is mathematically neutral.
3. **Raise Norm Floor:** Change `ConditionalNormFloor` from `1e-14` to `1e-12` to stay safely above standard float64 noise accumulation from the RK4 steps.

### Low-priority polish

1. **Omit empty bridge:** Ensure the JSON struct uses `omitempty` for the `Bridge` field so it is entirely absent when disabled, preserving exact byte compatibility with `0001` where possible.
2. **Report schema version:** Unconditionally output `schema_version: "bell_mipt_report_v0_2a"` at the top level to signal binary capabilities, regardless of bridge activation.

---

### Answer table

| Question | Answer | Severity | Repair |
| --- | --- | --- | --- |
| 1. Remaining blockers? | Boundary-crossing jumps and JW parity contamination will manufacture fake signals. | blocker | Isolate boundary jumps; document JW phase artifact. |
| 2. "environment-projected..." safer? | Yes, removes Bohmian/measurement baggage. | none | Keep. |
| 3. "...conditional_update" safe? | Yes, descriptive of the mathematical operation. | none | Keep. |
| 4. Measurement implication risk? | Neutralized by the explicit limitation strings. | none | Keep limitations exactly as written. |
| 5. Debt status overstatement? | "partially paid" is fine given the "toy_only" qualifiers. | none | Keep. |
| 6. Time-step convention correct? | Yes, computing $k \to k+1$ transitions using $\psi[k]$ is standard and safe. | none | Implement exactly as specified. |
| 7. Jump sampling timing? | Using $\psi[k]$ before/alongside RK4 computing $\psi[k+1]$ is correct. | none | Keep. |
| 8. RK4 sub-stages skip OK? | Yes, interpolating stochastic jumps inside RK4 is overly complex for a toy. | none | Keep. |
| 9. `rates[dest][src]` unambiguous? | Yes, standard stochastic column-stochastic matrix notation. | none | Keep. |
| 10. Orientation tests sufficient? | Yes, provided they use an asymmetric mock rate matrix. | none | Keep. |
| 11. JW Sign correctly handled? | No. Naive projection ignores relative phase shifts when B parity changes. | high | Add `jordan_wigner_parity_contamination` warning. |
| 12. Non-contiguous tests sufficient? | Good for bitwise logic, but doesn't fix physics signs. | low | Keep. |
| 13. Canonical sorting safe? | Mandatory. User inputs must be sorted to match global Hamiltonian ordering. | high | Enforce canonical sorting in config. |
| 14. Report both orderings? | Yes, provides transparency. | low | Keep. |
| 15. `basis_index_to_config`? | Yes, necessary if the basis isn't a simple $1 \ll \text{site}$ mapping. | medium | Add interface/function requirement. |
| 16. `max_lambda_dt` thresholds? | 0.5 is too high; Poisson assumption breaks down entirely. | high | Change warning to 0.05, inconclusive to 0.20. |
| 17. `> 0.1` inconclusive? | `> 0.2` is a better inconclusive hard limit. | medium | Use 0.20 for inconclusive. |
| 18. Norm-failure thresholds? | 10% and 50% are reasonable descriptive limits. | none | Keep. |
| 19. `1e-14` floor acceptable? | Too close to float64 epsilon accumulation. | low | Raise to `1e-12`. |
| 20. Near-zero $ | \psi_q | ^2$ logic? | Rate division will explode. |
| 21. Fidelity drop enough? | Yes, sufficient for `0002A` diagnostic. Trace distance not needed yet. | none | Keep. |
| 22. `fixed_environment_...`? | Scope creep. Remove. | low | Delete from schema. |
| 23. Ratio event-count rule? | Excellent guardrail against noise. | none | Keep. |
| 24. Ratio denominator? | Must use `no_jump` to isolate smooth Schrödinger drops. | high | Change denominator to `no_jump_transitions`. |
| 25. Boundary jumps excluded? | MUST be excluded. They change the subsystem directly, faking correlation. | blocker | Add `boundary_crossing_jump` classification. |
| 26. Bridge section if disabled? | Omit entirely. | low | Use JSON `omitempty`. |
| 27. Schema version bump? | Always bump to signal the binary version. | low | Keep. |
| 28. Status names safe? | Yes, `audit_completed` avoids "theorem proven" implications. | none | Keep. |
| 29. Enhancement names safe? | Change `enhancement` to `correlation`. | medium | Use `candidate_update_correlation`. |
| 30. Warnings sufficient? | Excellent, but add the JW parity warning. | medium | Add JW warning to the list. |
| 31. Mandatory tests sufficient? | Almost. Needs test for boundary jump exclusion. | high | Add `TestBoundaryCrossingJumpExcluded`. |
| 32. Missing test? | Divide-by-zero safeguard test for zero probability nodes. | high | Add `TestNearZeroProbabilityForceZeroRate`. |
| 33. Unnecessary test? | None. | none | Keep all. |
| 34. Stochastic vs fake RNG? | CI must use injected fake RNGs. Statistical bounds are too flaky. | medium | Mandate dependency-injected RNG interface. |
| 35. Disabled regression test? | Semantic is safer for floats, but byte-for-byte is fine if seeds match. | low | Use semantic JSON equality. |

---

### Final repaired implementation checklist

The coding agent must implement the following checklist:

1. **Core Extension:** Implement `config` parser for the Bridge section, enforcing canonically sorted site arrays.
2. **Safety First:** Update `report.go` to emit `schema_version: "bell_mipt_report_v0_2a"`. Ensure the `bridge` field is `omitempty`. Add all hardcoded disclaimer strings to pass `forbidden_test.go`.
3. **Engine Alignment:** Implement trajectory loop. For interval $k \to k+1$, compute rates using $\psi[k]$. **Critical:** If $|\psi(q, t_k)|^2 < 10^{-12}$, set outgoing $\lambda = 0$.
4. **Diagnostics:** Track `max_lambda_dt`. Trigger warning if $>0.05$, inconclusive if $>0.20$.
5. **Conditional Vector:** Split configurations. Skip normalization and flag failure if norm $< 10^{-12}$.
6. **Jump Classification:** Strictly partition jumps into `no_jump`, `strict_environment_jump`, `strict_subsystem_jump`, and `boundary_crossing_jump`.
7. **Ratio Math:** Compute `conditional_update_ratio` = (mean drop at `strict_environment_jump`) / (mean drop at `no_jump`). Return JSON `null` if denominator events $<10$ or numerator events $<10$.
8. **Warning Assembly:** Emit `jordan_wigner_parity_contamination_risk` if bridge enabled, alongside any lambda or norm failure warnings.
9. **Test Suite:** Write all 16 mandatory tests, using deterministic injected RNGs, explicitly including `TestBoundaryCrossingJumpExcluded`.

---

### EBP 2.1 ledger

```json
{
  "needMap": "ready_for_repaired_conditional_vector_toy_attempt",
  "needInvariant": "0001_partially_paid; 0002A_empirical_trajectory_diagnostic_planned",
  "needToyCheck": "0002A_plan_repaired_pending_implementation",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "0002A_pending_source_review_after_implementation",
  "promotion_status": "unpromoted_final_plan_review_only"
}

```

### Final recommendation

Proceed to implementation.
