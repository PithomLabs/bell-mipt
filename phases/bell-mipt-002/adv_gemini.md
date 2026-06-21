### Executive verdict

`use_as_base_with_repairs`

The provided plan outlines a solid, backward-compatible software architecture that achieves the ticket's goals without disrupting `BELL-MIPT-0001`. However, it contains several conceptual traps related to fermionic basis ordering, time-step alignment, and terminology that risks violating the strict physics non-goals. With targeted repairs, it is safe to implement.

### Strongest parts of the plan

* **Two-pass execution mode:** Reusing the `0001` pass and saved snapshots ensures total backward compatibility and isolates the stochastic sampler from the deterministic master equation.
* **Status separation:** Keeping `goal_status` (master equation) completely independent of `bridge_status` (stochastic sampler) prevents the new toy features from invalidating previously verified equivariance baselines.
* **Discrete-time thinning:** Reusing the existing small $dt$ from the RK4 solver rather than building a full continuous-time Gillespie solver avoids massive architectural drift and keeps the physics tightly coupled to the numerical grid.

### Critical blockers

* **Fermionic Jordan-Wigner phase ambiguity:** The plan proposes splitting the binary configuration $q$ into $Q_A$ and $Q_B$ but ignores fermionic antisymmetry. While simply extracting amplitudes for fixed $Q_B$ works numerically, if the user-specified `subsystem_sites` are not monotonically ordered, mapping the conditional array back to physical fermion operators risks sign errors.
* **Terminology boundary violation:** The phrase `monitoring_like_signal` directly violates the non-goal "do not claim Bell jumps are measurements." It must be completely scrubbed from the JSON schema and reporting logic.
* **Diverging rates at wave-function nodes:** The plan does not specify what to do when the actual configuration $Q(t)$ occupies a state $q$ where $|\psi_q(t)|^2 \to 0$, causing outgoing Bell rates to diverge (division by zero in rate definitions).

### High-priority repairs

* **Rename signal diagnostic:** Change `monitoring_like_signal` to `conditional_update_diagnostic`, with allowed values: `not_assessed`, `no_clear_signal`, `weak_signal`, `candidate_signal`.
* **Strict site sorting:** The config parser must enforce that `subsystem_sites` and `environment_sites` are sorted in ascending order to preserve the original Jordan-Wigner string ordering.
* **Fidelity drop null-model adjustment:** Smooth Schrödinger evolution naturally causes fidelity drops. The metric `mean_fidelity_drop_without_environment_jumps` is a weak null. The implementation must document that this metric conflates unitary evolution with subsystem jumps.
* **Boundary-crossing jump isolation:** Add a transition classification for `boundary_jump` (both $Q_A$ and $Q_B$ change simultaneously due to a hopping term crossing the cut). Conflating this with a pure `environment_jump` corrupts the conditional update ratio.

### Medium-priority repairs

* **Change success status:** Rename `bridge_toy_passed` to `bridge_audit_completed`. "Passed" implies a physical theorem was proven, which violates the EBP boundaries.
* **JSON Null serialization:** Ensure the Go implementation uses pointers (`*float64`) for ratio metrics so unavailable values serialize as `null` rather than `0.0`.
* **Explicit alignment definition:** Specify that rates for the jump $t \to t+dt$ are computed using $\psi(t)$, *before* the RK4 step updates the wave function to $\psi(t+dt)$.

### Low-priority polish

* **Bump Report Schema Version:** Update the implicit or explicit schema version string (e.g., `bell_mipt_report_v0_2a`) to reflect the new optional JSON fields.
* **Deterministic RNG config:** Document that `config.Bridge.Seed` exactly dictates the trajectory paths for regression testing.

---

### Specific answer table

| Question | Answer | Severity | Repair |
| --- | --- | --- | --- |
| 1. Fermionic sign/convention? | Amplitudes extract safely, but arbitrary site ordering corrupts operator parity. | blocker | Force `subsystem_sites` and `environment_sites` arrays to be sorted. |
| 2. Conditioning on non-conserving pairing? | Yes, valid in the occupation basis, though total particle number will fluctuate. | none | None. |
| 3. Actual $Q_B(t)$ enough? | Mathematically yes for the Bohmian/Bell projection, but it's a toy diagnostic. | low | Include explicit disclaimer in report strings. |
| 4. Measurement confusion? | High risk. `monitoring_like_signal` invites misinterpretation. | blocker | Rename to `conditional_update_diagnostic`. |
| 5. Fidelity drops vs Schrödinger? | Yes, unitary evolution dominates drops for small $dt$. | high | Document conflation; track `pure_schrodinger` drop if possible. |
| 6. Better null model? | Yes, comparing against deterministic unitary evolution of $\psi_A$ is better. | medium | Not required for 0002A, but log as known limitation. |
| 7. "monitoring_like_signal"? | Too strong, violates EBP rules. | blocker | Rename field completely. |
| 8. Boundary-crossing jumps? | They change both, polluting the `environment_jump` bucket. | high | Add `boundary_jump` classification. |
| 9. Empirical L1 useful? | Mostly noise for 200 samples. Strictly a descriptive diagnostic. | low | Keep as descriptive; do not gate `bridge_status` on it. |
| 10. Finite size artifacts? | Yes, small toy dimensions yield wild fluctuations. | none | Keep limitations text strict. |
| 11. Discrete-time thinning? | Valid for $\lambda dt \ll 1$, but unsafe if $\lambda$ spikes. | medium | Enforce `max_lambda_dt` diagnostic limit. |
| 12. `max_lambda_dt` sufficient? | Needs a strict threshold, e.g., $> 0.1$ triggers warning. | medium | Add hard threshold for `bridge_toy_inconclusive`. |
| 13. Multiple jumps per dt? | Unnecessary complexity for the toy model; keep at max 1. | none | None. |
| 14. Rate evaluation timing? | Compute at $t_k$ to determine jump for interval $[t_k, t_{k+1}]$. | high | Explicitly document this loop alignment in `trajectory.go`. |
| 15. RK4 consistency? | Consistent if $dt$ is small enough to treat as Markovian steps. | none | None. |
| 16. Reuse 0001 rates? | Must reuse the exact exact same rate definitions to test equivariance. | high | Use existing rate module API; do not rewrite formulas. |
| 17. Rate caching orientation? | `rates[n][q]` vs `rates[q][n]` is a massive risk. | blocker | Write strict deterministic unit tests verifying transition targets. |
| 18. Empirical L1 micro-steps? | Only at sampled steps to match statistical availability. | none | None. |
| 19. Check all marginals? | Overkill for 0002A; initial + final + max L1 is sufficient. | low | None. |
| 20. Zero probability nodes? | Bell rates diverge when $ | \psi_q | ^2 \to 0$. |
| 21. `ConditionalNormFloor`? | $10^{-14}$ is too low; float precision loss occurs earlier. | medium | Raise to $10^{-12}$ to avoid numerical garbage states. |
| 22. Absolute or relative norm? | Absolute norm of the conditional slice is correct. | none | None. |
| 23. Norm failure status? | High rate ($>5\%$) should trigger `bridge_toy_inconclusive`. | high | Add $>5\%$ check to status logic. |
| 24. Precomputing `ByEnvironment`? | Yes, provided site mappings are sorted and validated. | none | None. |
| 25. Sort site lists? | Must be strictly sorted to avoid Jordan-Wigner phase corruption. | blocker | Config parser must reject or auto-sort unsorted lists. |
| 26. Record site ordering? | Yes, report the exact partitioning used. | low | Add `actual_subsystem_sites` to report. |
| 27. Combine == Split test? | Yes, a highly effective invariant test for bitmask math. | medium | Mandate this test in `conditional_test.go`. |
| 28. Non-contiguous partitions? | Yes, bitmask math must support this (e.g., A=[0,2], B=[1,3]). | medium | Add specific unit test for interleaved sites. |
| 29. Boundary-jump tests? | Yes, fidelity logic needs to prove it handles both changing. | high | Add synthetic test case for $A$ and $B$ flipping simultaneously. |
| 30. Other distance metrics? | Fidelity is fine for 0002A; keep it simple. | none | None. |
| 31. Bump schema version? | Yes, distinguishes parser capabilities. | low | Add field or update docstring. |
| 32. Bridge section if disabled? | Omit entirely (`omitempty` JSON tag) to preserve 0001 shape. | high | Enforce `omitempty` on `BridgeReport`. |
| 33. `bridge_toy_passed`? | Overclaim. | medium | Change to `bridge_audit_completed`. |
| 34. `candidate_signal`? | Acceptable if key is renamed to `conditional_update_diagnostic`. | medium | Rename parent key. |
| 35. Hide ratio on low events? | Yes, $<10$ events is purely noise. | medium | Return `null` if denominator events $< 10$. |
| 36. Unavailable JSON format? | Must be `null`, not `0.0` or `NaN`. | high | Use `*float64` in Go struct. |
| 37. Warnings array? | Yes, warnings clarify limitations automatically. | medium | Add `warnings: []string` to bridge report. |
| 38. `goal_status` links? | No, keep them independent. | none | None. |
| 39. Debt status overstatement? | Yes, "partially paid" is too strong if inconclusive. | medium | Debt updates only if status is `bridge_audit_completed`. |
| 40. "conditional_state_toy_only"? | Accurate description of the limit. | none | None. |
| 41. Rate orientation tests? | Mandatory. | high | Add test mocking `rates[n][q]`. |
| 42. Alignment tests? | Mandatory. | high | Add test verifying step $k$ rate applies to interval $k \to k+1$. |
| 43. Stochastic CI tests? | Flaky. Use deterministic mock rates/RNG for CI. | medium | Mandate dependency injection for RNG in tests. |
| 44. Integration test strictness? | Semantic field checks, not byte-for-byte. | low | Update test suite rules. |
| 45. Force known-jump tests? | Yes, ensures ratio logic executes. | high | Create a synthetic state with 100% transition probability. |
| 46. Eigenstate test? | Yes, ensures zero-rate/no-jump logic executes correctly. | medium | Add stationary state test. |
| 47. Forbidden language test? | Must scan entire final JSON string. | high | Extend `forbidden_test.go` to hit `json.Marshal` output. |
| 48. Bridge-disabled equivalence? | Critical regression check. | blocker | Add test comparing 0001 output hash with 0002A-disabled output. |
| 49. `max_lambda_dt` tests? | Yes, force a high rate to trigger inconclusive. | medium | Add test case. |
| 50. Additional tests? | Divide-by-zero safeguard test when $ | \psi_q | ^2 < 10^{-14}$. |

---

### Recommended revised plan

**Phase 1: Validation & Architecture**

1. Extend `Config` with `Bridge` settings. Add validation ensuring `subsystem_sites` and `environment_sites` are non-overlapping, exhaustive, and **strictly ascending** to preserve Jordan-Wigner ordering.
2. Maintain the two-pass execution logic. If `bridge.enabled == false`, output exactly matches `BELL-MIPT-0001` (ensure JSON `omitempty` hides bridge fields).

**Phase 2: Trajectory Engine**

1. Initialize $Q_0$ from $|\psi_0|^2$.
2. At each step $t_k$, before RK4 updates the state:
* If $|\psi_q(t_k)|^2 < 10^{-14}$, set $\lambda = 0$ (avoid diverging rates).
* Otherwise, fetch total rate $\lambda = \sum \sigma(n \leftarrow q)$.
* Track `max_lambda_dt = max(max_lambda_dt, \lambda \cdot dt)`.
* Calculate $p_{\text{jump}} = 1 - \exp(-\lambda \cdot dt)$.
* Sample uniform $r$. If $r < p_{\text{jump}}$, sample destination $n$ proportionally from $\sigma(n \leftarrow q)$.



**Phase 3: Conditional Audit**

1. Split configuration using bitwise operations (add `Combine(Split(q)) == q` test).
2. Extract conditional vector amplitudes.
3. If absolute norm $< 10^{-12}$, log failure and skip fidelity.
4. Compute fidelity $F = |\langle\psi_A(t) | \psi_A(t+dt)\rangle|^2$, drop $= 1 - F$.
5. Classify transitions mutually exclusively: `no_jump`, `subsystem_jump`, `environment_jump`, and **`boundary_jump`** (both change).

**Phase 4: Reporting & Status**

1. Rename `monitoring_like_signal` to `conditional_update_diagnostic`.
2. Set `bridge_status` to `bridge_audit_completed` (not "passed"), `failed`, or `inconclusive`.
3. Trigger `inconclusive` if `max_lambda_dt > 0.1` or norm failures $> 5\%$.
4. Use `*float64` for ratios to allow JSON `null` if event counts $< 10$.
5. Embed hardcoded limitation strings in output to pass the forbidden-language audit.

---

### EBP 2.1 ledger

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

### Final recommendation

Proceed to implementation after repairs.
