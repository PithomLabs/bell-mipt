# Implementation Planning Prompt: BELL-MIPT-0002A — Bell Trajectory + Conditional Subsystem State Audit

You are writing an implementation plan for the next ticket after `BELL-MIPT-0001`.

## Background

`BELL-MIPT-0001` has been implemented and accepted for limited toy scope. It computes Bell jump rates for a finite Kitaev-chain-style fermionic lattice and numerically verifies equivariance: the Bell master-equation distribution `rho(t)` tracks `|psi(t)|^2`.

Reported result:

```text
goal_status: toy_goal_passed
max_equivariance_l1_error: ~3.13e-11
physics_claim: none
toy_analysis_only: true
```

Code review accepted the implementation for finite Bell-rate/equivariance toy scope. The following remain unpaid:

```json
{
  "needMap": "unpaid",
  "needInvariant": "partially_paid_equivariance_only",
  "needToyCheck": "partially_paid_rate_algebra_only",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_reviewed"
}
```

## New Ticket

```text
BELL-MIPT-0002A:
Bell trajectory sampler + conditional subsystem state audit
```

## Goal

Extend the existing Go toy program so it can sample actual Bell configuration histories and compute the conditional subsystem wave function induced by the environment configuration.

The central question is:

> Do Bell jumps in the environment induce measurable changes in the subsystem conditional state, beyond ordinary smooth Schrödinger evolution?

Plain English:

```text
Universal ψ(t) evolves normally.
Actual Bell configuration Q(t) jumps according to Bell rates.
Split Q(t) into subsystem A and environment B.
Use actual Q_B(t) to define a conditional wave function ψ_A(a,t) ∝ Ψ(a, Q_B(t), t).
Measure whether environment jumps produce nontrivial conditional-state updates.
```

This ticket should test whether the Bell trajectory has any monitoring-like effect at the conditional-state level.

## Hard Non-Goals

Do not implement:

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
physics promotion
```

Do not claim:

```text
Bell jumps are measurements.
Bell-MIPT bridge is established.
MIPT was observed.
Holography is explained.
Bohmian mechanics is validated.
```

Allowed conclusion if successful:

```text
Bell trajectories were sampled, equivariance was checked empirically, and conditional subsystem-state changes were measured for the tested finite toy configuration.
```

## Preserve Existing Command Shape

Do not add subcommands.

Keep:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

If config omits the new bridge section, preserve existing `BELL-MIPT-0001` behavior.

## Existing v0 Architecture

Assume the repo currently has:

```text
cmd/bellmipt/main.go
internal/bellmipt/config.go
internal/bellmipt/basis.go
internal/bellmipt/fermion.go
internal/bellmipt/hamiltonian.go
internal/bellmipt/current.go
internal/bellmipt/rates.go
internal/bellmipt/evolve.go
internal/bellmipt/audit.go
internal/bellmipt/report.go
internal/bellmipt/markdown.go
internal/bellmipt/forbidden.go
internal/bellmipt/run.go
```

`BELL-MIPT-0002A` should extend this code, not rewrite it.

Use Go only. Prefer standard library only.

## Proposed Config Extension

Plan a minimal optional section:

```json
{
  "bridge": {
    "enabled": true,
    "subsystem_sites": [0, 1, 2],
    "environment_sites": [3, 4, 5],
    "trajectories": 200,
    "seed": 777,
    "sample_every_steps": 1
  }
}
```

Rules:

```text
bridge.enabled=false or omitted → run old BELL-MIPT-0001 path only.
subsystem_sites and environment_sites must partition all sites.
No overlapping sites.
No missing sites.
trajectories > 0 when enabled.
sample_every_steps >= 1.
```

Keep the default config with bridge disabled unless explicitly enabled.

## Required Concepts

### 1. Bell trajectory sampling

At each time step, given the current actual configuration `q` and current wavefunction `psi`:

1. Compute Bell current `J`.
2. Compute Bell rates `sigma(n <- q)` for all possible destination states `n`.
3. Total outgoing rate:

```text
lambda(q,t) = Σ_n sigma(n <- q)
```

4. For small `dt`, sample a jump with probability approximately:

```text
p_jump = 1 - exp(-lambda * dt)
```

5. If a jump occurs, choose destination `n` with probability:

```text
sigma(n <- q) / lambda
```

6. If no jump occurs, keep current configuration.

Use deterministic RNG seeded from the bridge config.

The first version may use discrete-time thinning with small `dt`, because `BELL-MIPT-0001` already uses small fixed time steps. Do not implement a full continuous-time Gillespie sampler in this ticket unless the plan argues it is strictly necessary.

### 2. Initial configuration sampling

At `t=0`, sample actual configuration `Q_0` from:

```text
P(Q_0 = q) = |psi_0(q)|^2
```

Track empirical distribution across trajectories and compare against `|psi_0|^2`.

### 3. Conditional subsystem wave function

Given a full basis configuration `q`, split it into subsystem part `a` and environment part `b`.

Given actual environment configuration `B(t) = b0`, define:

```text
psi_A(a,t) = Psi(a,b0,t)
```

then normalize over `a`:

```text
psi_A_normalized(a,t) = psi_A(a,t) / sqrt(Σ_a |psi_A(a,t)|^2)
```

If the norm is below a small threshold, record a conditional norm failure.

Important: the conditional wave function is a vector over subsystem configurations only.

### 4. Fidelity / update metric

At each sampled step, compute normalized conditional state `psi_A(t)`.

Compute conditional fidelity between consecutive sampled conditional states:

```text
F(t, t+dt) = |<psi_A(t) | psi_A(t+dt)>|^2
```

Then define fidelity drop:

```text
drop = 1 - F
```

Classify each transition:

```text
environment_jump: Q_B changed
subsystem_jump: Q_A changed
any_jump: Q changed
no_jump: Q unchanged
```

Track:

```text
mean_fidelity_drop_at_environment_jumps
mean_fidelity_drop_without_environment_jumps
mean_fidelity_drop_at_any_jumps
mean_fidelity_drop_no_jump
conditional_update_ratio =
  mean_fidelity_drop_at_environment_jumps /
  mean_fidelity_drop_without_environment_jumps
```

If denominator is zero or too small, report `null` or mark ratio as unavailable.

### 5. Empirical trajectory equivariance

Across many sampled trajectories, at selected times compare empirical distribution of `Q(t)` with `|psi(t)|^2`.

Use L1 distance:

```text
empirical_equivariance_l1(t) = Σ_q |freq(q,t)/trajectories - |psi_q(t)|^2|
```

This will be noisy with finite trajectories, so do not use the same strict tolerance as master-equation equivariance.

Report it as a descriptive diagnostic unless the plan proposes a statistically meaningful threshold.

## Required Outputs

Extend the report schema carefully.

Existing fields must remain:

```json
{
  "toy_analysis_only": true,
  "physics_claim": "none",
  "goal_status": "toy_goal_passed | toy_goal_failed | toy_goal_inconclusive"
}
```

Add bridge section:

```json
{
  "bridge": {
    "enabled": true,
    "bridge_goal": "sample_bell_trajectories_and_audit_conditional_subsystem_state",
    "bridge_status": "bridge_toy_passed | bridge_toy_failed | bridge_toy_inconclusive | bridge_disabled",
    "trajectories": 200,
    "subsystem_sites": [0, 1, 2],
    "environment_sites": [3, 4, 5],
    "sample_every_steps": 1,
    "metrics": {
      "trajectory_count": 200,
      "mean_jump_count": 0.0,
      "mean_environment_jump_count": 0.0,
      "mean_subsystem_jump_count": 0.0,
      "conditional_norm_failures": 0,
      "mean_fidelity_drop_at_environment_jumps": 0.0,
      "mean_fidelity_drop_without_environment_jumps": 0.0,
      "conditional_update_ratio": 0.0,
      "max_empirical_equivariance_l1": 0.0,
      "final_empirical_equivariance_l1": 0.0
    },
    "interpretation": {
      "monitoring_like_signal": "not_assessed | no_clear_signal | weak_signal | candidate_signal",
      "reason": ""
    }
  }
}
```

Important: `monitoring_like_signal` is descriptive only. It must not become a physics promotion.

## Goal Status Logic

The old `goal_status` from `BELL-MIPT-0001` should still reflect the master-equation/equivariance audit.

Add a separate `bridge_status`.

Suggested:

```text
goal_status:
  still reports whether finite Bell-rate/equivariance audit passed.

bridge_status:
  bridge_disabled if bridge.enabled=false or omitted.
  bridge_toy_passed if trajectories sampled successfully, conditional states computed reliably, and metrics emitted.
  bridge_toy_failed if trajectory sampling or conditional-state computation has implementation-level failures.
  bridge_toy_inconclusive if too many conditional norm failures, too few environment jumps to compare, or empirical equivariance is too noisy to trust.
```

Do not mark the bridge as “passed” merely because `conditional_update_ratio > 1`. A signal is not a bridge proof.

## Debt Status Update

If bridge is enabled and runs successfully:

```json
{
  "needMap": "partially_paid_conditional_state_toy_only",
  "needInvariant": "partially_paid_equivariance_plus_empirical_trajectory_check",
  "needToyCheck": "partially_paid_rate_algebra_and_conditional_state_toy",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_reviewed_for_0001_only"
}
```

If bridge is disabled, retain old debt status.

## Required Tests

Plan tests for:

### Config

```text
bridge omitted defaults to disabled
bridge disabled preserves old behavior
bridge partition validation rejects overlap
bridge partition validation rejects missing site
bridge partition validation accepts valid split
trajectories must be positive when enabled
sample_every_steps must be positive
```

### Trajectory sampling

```text
initial configuration sampled from |psi|² approximately for many samples
jump probability uses total outgoing Bell rate
destination sampling proportional to outgoing rates
no jump occurs when total outgoing rate is zero
deterministic trajectory output for fixed seed
```

### Conditional wave function

```text
split full state into subsystem/environment configurations correctly
conditional state vector has dimension 2^|A|
conditional state normalizes when norm > threshold
conditional norm failure recorded when norm too small
fidelity of identical conditional states is 1
fidelity drop is in [0,1] within numerical tolerance
```

### Bridge metrics

```text
environment jumps counted correctly
subsystem jumps counted correctly
fidelity drops classified correctly
conditional_update_ratio handles zero denominator
empirical trajectory equivariance L1 is computed
bridge_status is bridge_toy_passed/failed/inconclusive according to rules
```

### Reports

```text
old report fields remain backward compatible
bridge_disabled appears when bridge omitted
bridge metrics appear when bridge enabled
limitations include no MIPT/no holography/no measurement claim
forbidden-language audit still passes
```

## Required Limitations

Report must include:

```text
This samples Bell configuration histories in a finite toy model only.
This conditional-state audit is not a monitored quantum trajectory simulation.
This does not implement MIPT.
This does not show Bell jumps are measurements.
This does not establish a Bell-MIPT bridge.
This does not support any holography or black-hole claim.
This is not a physics promotion.
```

## Validation Commands

Plan should require:

```bash
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge
```

## Expected Final Implementation Report

The coding agent should later report:

```text
1. Files added/modified
2. Whether BELL-MIPT-0001 behavior remained unchanged when bridge disabled
3. Bridge enabled run status
4. Key bridge metrics
5. Test commands run
6. Known limitations
7. EBP debt update
8. Explicit statement:
   No MIPT claim, no holography claim, no Bell-jumps-equal-measurements claim, no physics promotion.
```

## Your Task

Write a detailed implementation plan for `BELL-MIPT-0002A`.

The plan should be specific enough that a coding agent can implement it directly, but it should not write code yet.

It must include:

```text
file layout changes
new config structs
new data structures
trajectory sampling algorithm
conditional-wave-function construction
bridge metrics
report schema changes
goal/bridge status logic
tests
validation commands
EBP debt updates
known risks
non-goals
```

End with a clear recommendation on whether to implement `BELL-MIPT-0002A` as a direct extension of the current command or as a separate package under `internal/bellmipt`.

