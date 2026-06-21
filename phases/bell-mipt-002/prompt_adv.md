# Adversarial Review Prompt: BELL-MIPT-0002A Final Implementation Plan

You are performing an adversarial review of a proposed implementation plan for:

```text
BELL-MIPT-0002A:
Bell trajectory sampler + conditional subsystem state audit
```

This ticket extends an already accepted Go toy implementation, `BELL-MIPT-0001`, which computed Bell jump rates for a finite Kitaev-chain-style fermionic lattice and numerically verified equivariance: the Bell master-equation distribution `rho(t)` tracks `|psi(t)|²`.

The proposed `0002A` plan adds an optional bridge audit:

```text
Sample Bell configuration trajectories.
Split each actual configuration Q(t) into subsystem Q_A(t) and environment Q_B(t).
Construct conditional subsystem wave function ψ_A(a,t) ∝ Ψ(a,Q_B(t),t).
Measure fidelity drops between consecutive conditional subsystem states.
Compare drops at environment jumps versus non-environment-jump transitions.
Report empirical trajectory equivariance as descriptive diagnostic.
```

Your task is to **criticize the plan before implementation**.

Do not merely summarize it. Look for conceptual mistakes, software-design traps, numerical traps, misleading status logic, insufficient tests, and hidden physics overclaims.

## 1. Critical boundaries

The plan must not implement or claim:

```text
MIPT phase diagram
area-law / volume-law scaling
entanglement entropy
mutual information
purity
monitored quantum circuits
projective measurements
Lindblad dynamics
holography
black holes
Lean theorem proving
AI agents
multi-flavor Majorana chains
continuous-time Gillespie sampler
Bell jumps are measurements
Bell-MIPT bridge is established
MIPT was observed
Bohmian mechanics was validated
physics promotion
```

Allowed conclusion, if implemented correctly:

```text
Bell trajectories were sampled, empirical trajectory equivariance was checked descriptively, and conditional subsystem-state changes were measured for the tested finite toy configuration.
```

## 2. Plan summary to review

The current proposed plan says:

```text
Base plan: Claude
Merged additions:
  - Xiaomi max_lambda_dt diagnostic and implementation phasing
  - Z optional per-time-step rate-cache optimization
  - Gemini compact status wording and limitation list

Rejected/repair items:
  - no hard empirical L1 threshold
  - unavailable ratios serialize as null, not 0
  - no raw global ban on words like MIPT/holography/measurement
  - no mandatory math/rand/v2
  - conditional_update_ratio must not determine bridge_status
```

Architecture:

```text
Modify:
  internal/bellmipt/config.go
  internal/bellmipt/run.go
  internal/bellmipt/report.go
  internal/bellmipt/markdown.go
  internal/bellmipt/forbidden.go

Add:
  internal/bellmipt/bridge_config.go
  internal/bellmipt/site_split.go
  internal/bellmipt/trajectory.go
  internal/bellmipt/conditional.go
  internal/bellmipt/bridgemetrics.go
  internal/bellmipt/equivariance_empirical.go
  internal/bellmipt/bridge_report.go

No new binary, no subcommands, no external packages.
```

Core design:

```text
Two-pass mode.

Pass 1:
  Run existing BELL-MIPT-0001 master-equation/equivariance audit.
  Store ψ(t) snapshots.
  Keep rho master-equation audit unchanged.

Pass 2:
  If bridge.enabled=true, use ψ snapshots to sample actual Bell configuration trajectories.
  Compute conditional subsystem states from Q_B(t).
  Compute fidelity-drop metrics and empirical trajectory equivariance.
```

Trajectory sampling:

```text
Q0 sampled from |ψ0(q)|².
At each time step:
  compute Bell rates σ(n <- q) using ψ(t)
  λ = Σ_n σ(n <- q)
  p_jump = 1 - exp(-λ dt)
  if jump, destination n sampled proportional to σ(n <- q)
```

Conditional state:

```text
Split q into subQ/envQ.
ψ_A(a,t) = Ψ(a, envQ, t).
Normalize if norm >= ConditionalNormFloor.
If norm too small, record failure and skip fidelity.
```

Metrics:

```text
mean_jump_count
mean_environment_jump_count
mean_subsystem_jump_count
conditional_norm_failures
mean_fidelity_drop_at_environment_jumps
mean_fidelity_drop_without_environment_jumps
mean_fidelity_drop_at_any_jumps
mean_fidelity_drop_no_jump
conditional_update_ratio
initial/max/final empirical equivariance L1
max_lambda_dt
environment_jump_transitions
non_environment_jump_transitions
any_jump_transitions
no_jump_transitions
```

Status logic:

```text
goal_status remains the original 0001 master-equation audit status.

bridge_status:
  bridge_disabled
  bridge_toy_passed
  bridge_toy_failed
  bridge_toy_inconclusive

conditional_update_ratio does not determine bridge_status.
monitoring_like_signal is descriptive only:
  not_assessed
  no_clear_signal
  weak_signal
  candidate_signal
```

## 3. Specific adversarial questions

Answer each question directly.

### A. Physics/conceptual correctness

1. Is the conditional wave function definition `ψ_A(a,t) = Ψ(a,Q_B(t),t)` valid for this finite fermionic occupation basis toy, or does fermionic antisymmetry/Jordan-Wigner ordering introduce a missing sign/convention issue?

2. Does conditioning on environment occupation configuration make sense for a Kitaev-chain-style model with pairing terms that do not conserve particle number?

3. Is using the actual environment configuration `Q_B(t)` enough to call this a “conditional subsystem state,” or are there additional Bohmian/Bell-type QFT conditions needed before that term is faithful?

4. Does the plan risk confusing Bell configuration jumps with measurement outcomes even if the report says it does not?

5. Are fidelity drops between conditional states a meaningful first diagnostic, or could they be dominated by ordinary Schrödinger phase evolution unrelated to jumps?

6. Should the plan compare environment-jump fidelity drops against a better null than “non-environment-jump transitions” even in 0002A?

7. Is “monitoring_like_signal” too strong a phrase? Should it be renamed to avoid implying measurement/MIPT?

8. Are subsystem/environment jumps properly classified when one Hamiltonian term crosses the A/B boundary and changes both sides?

9. Does empirical trajectory equivariance across finite stochastic samples actually test anything useful here, or is it mostly noise?

10. Does using a finite Kitaev-chain-style toy with small Hilbert dimension risk producing misleading conditional-state signals due to finite-size artifacts?

### B. Bell trajectory sampling correctness

11. Does discrete-time thinning with `p_jump = 1 - exp(-λdt)` correctly approximate the Bell jump process when rates are time-dependent through `ψ(t)`?

12. Is `max_lambda_dt` sufficient to detect when discrete-time thinning is unreliable? Should there be a stricter rule?

13. Should the code sample at most one jump per `dt`, or should it allow multiple jumps if `λdt` is not tiny?

14. Should rates be computed at `ψ(t_k)` before or after evolving to `ψ(t_{k+1})` for the transition `Q_k -> Q_{k+1}`?

15. If the existing 0001 RK4 evolves `ψ` on sub-stages, does sampling jumps only at full steps introduce a consistency issue?

16. Should trajectory sampling use the same rate computation as 0001 exactly, or is a rate-row accessor safe?

17. Does rate caching per time step create any risk if `rates[n][q]` orientation is misunderstood?

18. Should empirical trajectory equivariance use all micro-steps or only sampled steps?

19. Is the initial distribution sampling from `|ψ0|²` enough, or should the trajectory sampler also verify later marginal distributions against `|ψ(t)|²` with statistical confidence intervals?

20. What should happen when `|ψ_q|²` is below the rate probability floor for the current actual configuration?

### C. Conditional-state construction

21. Is `ConditionalNormFloor = 1e-14` too low, too high, or acceptable?

22. Should norm failure be judged by absolute norm or relative to trajectory/sample probability?

23. Should conditional-state norm failures make the bridge inconclusive, failed, or simply be reported?

24. Does precomputing `ByEnvironment` using full-basis indices correctly handle arbitrary site partitions?

25. Should subsystem and environment site lists be sorted internally, or should user-specified ordering be preserved because it defines subsystem basis ordering?

26. Should the report record the actual normalized site ordering used?

27. Does `CombineConfig(SplitConfig(q)) == q` catch all possible site-splitting bugs?

28. Should the implementation include tests for non-contiguous partitions such as A=[0,2,5], B=[1,3,4]?

29. Should the implementation include tests for boundary-crossing jumps where both subsystem and environment change?

30. Are conditional fidelities enough, or should the plan also include trace distance / L2 distance / phase-insensitive metrics?

### D. Report/schema/status logic

31. Should the report schema version be bumped from `bell_mipt_report_v0` to something like `bell_mipt_report_v0_2a`?

32. Should the bridge section be present even when disabled, or omitted for backward compatibility?

33. Does “bridge_toy_passed” sound too strong? Should it be renamed to “bridge_audit_completed” or similar?

34. Is `candidate_signal` too strong? Should it be renamed to `candidate_diagnostic_signal` or `conditional_update_candidate`?

35. Should `conditional_update_ratio` be reported if event counts are low, or hidden/null until minimum counts are met?

36. Should unavailable means/ratios be JSON `null`, `NaN`, omitted, or explicit objects with `{available:false, reason:"..."}`?

37. Should warnings include `max_lambda_dt`, low event counts, norm failures, finite-size limitations, and finite-sample noise?

38. Should `goal_status` fail if `bridge_status` fails, or remain independent?

39. Does the debt-status update overstate what is paid when the bridge is inconclusive?

40. Does the phrase `partially_paid_conditional_state_toy_only` overstate the result if conditional-state metrics are noisy?

### E. Testing/validation

41. Are the proposed tests sufficient to catch rate-orientation bugs `rates[n][q]` vs `rates[q][n]`?

42. Are the proposed tests sufficient to catch off-by-one time-step alignment bugs in trajectory sampling?

43. Are stochastic tests stable enough for CI, or should they use deterministic fake RNG / deterministic rate matrices?

44. Should integration tests compare default run reports byte-for-byte, or just compare semantic fields?

45. Should tests force a case with known environment jumps so `bridge_status` is not always inconclusive?

46. Should tests include zero-rate/eigenstate cases that honestly produce inconclusive bridge reports?

47. Should tests verify forbidden-language audit scans the fully assembled Markdown and JSON report?

48. Should tests verify the bridge-disabled path preserves all 0001 metrics exactly?

49. Should tests verify `max_lambda_dt` warning/inconclusive logic?

50. What additional tests would you require before implementation acceptance?

## 4. Expected review output format

Return your review in this structure.

### Executive verdict

Choose one:

```text
use_as_base
use_as_base_with_repairs
reject_for_now
needs_more_source_context
```

### Strongest parts of the plan

List the strongest design decisions.

### Critical blockers

List any blockers that would make the plan unsafe to implement.

### High-priority repairs

List repairs that should be made before implementation.

### Medium-priority repairs

List useful but non-blocking improvements.

### Low-priority polish

List naming/schema/reporting polish.

### Specific answer table

Answer the 50 adversarial questions in a compact table:

```text
Question | Answer | Severity | Repair
```

Severity options:

```text
blocker
high
medium
low
none
```

### Recommended revised plan

Provide a short corrected implementation plan, but do not write code.

### EBP 2.1 ledger

Use:

```json
{
  "needMap": "...",
  "needInvariant": "...",
  "needToyCheck": "...",
  "needNullModel": "...",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "...",
  "promotion_status": "unpromoted_plan_review_only"
}
```

### Final recommendation

Choose one:

```text
Proceed to implementation after repairs.
Request another planning revision.
Reject and redesign.
```

## 5. Review standard

Be strict. A good answer should be willing to say:

```text
This plan is useful but the phrase “monitoring-like signal” is too strong.
The empirical equivariance diagnostic is noisy and should not gate pass/fail.
The time-step alignment between ψ(t) and Q(t) must be specified exactly.
The test suite needs deterministic fake-rate tests to catch orientation bugs.
The report schema should avoid overclaiming even when conditional_update_ratio is high.
```

Do not give credit for vibes. Judge whether the plan is implementable, auditable, and EBP-safe.

