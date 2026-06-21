The next step is:

```text
BELL-MIPT-0002:
Bell trajectory + conditional-wave-function bridge toy
```

Not MIPT yet. Not holography. Not area-law/volume-law yet.

BELL-MIPT-0001 passed the finite Bell-rate/equivariance toy: the report shows `toy_goal_passed`, all checks true, and max equivariance L1 error about `3.13e-11`.  The code review also accepted it for limited toy scope and confirmed the Bell current, rate formula, fermion signs, periodic boundary handling, RK4 coupling, and equivariance audit. 

## The missing bridge

The real unresolved question is still:

> **Do Bell jumps merely move the hidden configuration, or do they induce meaningful effective conditional-state changes for subsystems?**

That is the Bell–MIPT bridge bottleneck.

MIPT is about **conditioned state trajectories**. Bell-type QFT gives **configuration trajectories**. To compare them, we need to construct the Bohmian/Bell conditional subsystem state and see whether actual Bell jumps behave like effective monitoring.

## Next ticket

```text
BELL-MIPT-0002:
Conditional-wave-function bridge toy
```

Goal:

> Simulate actual Bell configuration histories for the same finite Kitaev-chain toy, split the lattice into subsystem A and environment B, and test whether jumps in B cause nontrivial changes in the conditional wave function of A.

Plain-English test:

```text
Universal ψ(t) evolves normally.
Actual Bell configuration Q(t) jumps.
Split Q(t) into Q_A(t), Q_B(t).
Use Q_B(t) to define a conditional state for subsystem A.
Ask: do Bell jumps in B update A’s conditional state in a monitoring-like way?
```

## Minimal implementation

Keep one command shape:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

Add config fields only if needed:

```json
{
  "bridge": {
    "enabled": true,
    "subsystem_sites": [0, 1, 2],
    "environment_sites": [3, 4, 5],
    "trajectories": 200,
    "seed": 777
  }
}
```

Do not add subcommands.

## What to compute

For each Bell trajectory:

1. Evolve universal `ψ(t)` as before.
2. Generate actual configuration jumps using Bell rates.
3. At each time step, split actual configuration:

```text
Q(t) = (Q_A(t), Q_B(t))
```

4. Define conditional subsystem wave function:

```text
ψ_A(a, t) ∝ Ψ(a, Q_B(t), t)
```

5. Normalize it if possible.
6. Track changes when `Q_B(t)` jumps.

Useful metrics:

```text
trajectory_equivariance_error
environment_jump_count
subsystem_jump_count
conditional_state_norm_failures
mean_conditional_fidelity_drop_at_environment_jumps
mean_conditional_fidelity_drop_without_environment_jumps
conditional_update_ratio
```

The key metric is:

```text
conditional_update_ratio =
  mean fidelity drop at environment jumps
  /
  mean fidelity drop at non-jump steps
```

If this ratio is near 1, Bell jumps are probably not doing monitoring-like work.

If this ratio is clearly greater than 1, the bridge becomes worth pursuing.

## Pass/fail logic

`toy_goal_passed` only means:

```text
Bell trajectories were sampled correctly.
Empirical trajectory distribution approximately matched |ψ|².
Conditional ψ_A was computed.
Environment jumps produced measurable conditional-state updates.
```

`toy_goal_failed` means:

```text
Trajectory sampling failed, equivariance broke, or conditional ψ_A could not be computed reliably.
```

`bridge_not_supported` should be allowed as an honest result:

```text
Bell jumps did not produce stronger conditional-state updates than ordinary smooth evolution.
```

That is not a failure of the code. That is a negative physics result.

## What not to do yet

Do not implement:

```text
MIPT phase diagram
area-law / volume-law scaling
entanglement entropy scaling
Majorana multi-flavor chain
holography discussion
black-hole claims
```

Those come only if BELL-MIPT-0002 shows the conditional-state bridge is nontrivial.

## Why this is the correct next step

The Bell–MIPT bridge cannot be tested by more equivariance alone. BELL-MIPT-0001 already paid that finite toy debt. The next unpaid debt is:

```text
needMap:
Map Bell configuration jumps to conditional-state dynamics.
```

So the next useful experiment is not “bigger lattice” or “MIPT now.”

It is:

> **Do Bell histories create conditional-state updates that look even remotely like monitoring?**

## Final recommendation

Start with:

```text
BELL-MIPT-0002A:
Bell trajectory sampler + conditional subsystem state audit
```

Then, only if that passes with a nontrivial signal:

```text
BELL-MIPT-0002B:
Compare conditional-state update strength against a simple monitored-trajectory null model
```

Then, only after that:

```text
BELL-MIPT-0003:
Actual MIPT comparison on a known monitored Majorana/fermion model
```

No BS sequence:

```text
0001 passed: Bell rates/equivariance work.
0002 next: Do Bell jumps affect conditional subsystem states?
0003 later: If yes, compare to MIPT.
```

That is the clean path.

