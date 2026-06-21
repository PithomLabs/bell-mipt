The next step is **BELL-MIPT-0001: one-shot Go toy check**.

Goal:

> **Compute Bell jump rates for a finite fermionic lattice model and verify equivariance numerically.**

Not the goal:

```text
No MIPT claim.
No holography claim.
No Bell-jumps-equal-measurements claim.
No conditional-wave-function bridge yet.
```

The report is enough to justify this first toy because it found no direct Bell↔MIPT study and identified the finite fermionic Bell-rate calculation as the smallest useful next step. 

## Minimal program shape

Do **one simple Go command**, no bloated CLI:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

If `--config` is omitted, use a built-in default config.

The program should:

```text
1. Read flexible JSON input.
2. Build a finite Kitaev-chain-style fermion Hamiltonian.
3. Enumerate occupation-number configurations.
4. Compute Bell probability currents.
5. Compute Bell jump rates.
6. Evolve ψ by Schrödinger dynamics.
7. Evolve ρ by Bell master equation.
8. Compare ρ(t) against |ψ(t)|².
9. Write one JSON report and one Markdown summary.
10. Say whether the toy goal passed, failed, or was inconclusive.
```

## First input schema

Use something like this:

```json
{
  "schema_version": "bell_mipt_toy_v0",
  "model": "finite_kitaev_chain",
  "sites": 6,
  "boundary": "periodic",
  "parameters": {
    "mu": 1.0,
    "t": 1.0,
    "delta": 0.5
  },
  "initial_state": {
    "type": "random_normalized",
    "seed": 12345
  },
  "time": {
    "dt": 0.001,
    "steps": 1000
  },
  "audit": {
    "hermitian_tolerance": 1e-10,
    "norm_tolerance": 1e-8,
    "equivariance_tolerance": 1e-5
  }
}
```

Keep it flexible but small. For now, support only:

```text
model: finite_kitaev_chain
initial_state: random_normalized
boundary: open or periodic
```

Do not add subcommands yet.

## Output report

The program should write:

```text
out/bellmipt-run/
  input.json
  report.json
  report.md
```

The key output should look like:

```json
{
  "schema_version": "bell_mipt_report_v0",
  "toy_id": "BELL-MIPT-0001",
  "toy_analysis_only": true,
  "physics_claim": "none",
  "model": "finite_kitaev_chain",
  "sites": 6,
  "hilbert_dim": 64,
  "goal": "compute_bell_rates_and_verify_equivariance",
  "goal_status": "toy_goal_passed",
  "checks": {
    "hamiltonian_hermitian": true,
    "state_norm_preserved": true,
    "rates_nonnegative": true,
    "equivariance_error_within_tolerance": true
  },
  "metrics": {
    "max_hermitian_error": 0.0,
    "max_norm_error": 0.0,
    "max_negative_rate_violation": 0.0,
    "max_equivariance_l1_error": 0.0,
    "mean_total_bell_activity": 0.0,
    "max_total_bell_activity": 0.0
  },
  "debt_status": {
    "needMap": "unpaid",
    "needInvariant": "partially_paid_equivariance_only",
    "needToyCheck": "partially_paid_rate_algebra_only",
    "needNullModel": "unpaid",
    "needObstruction": "bell_jumps_are_not_measurements",
    "needFaithfulnessReview": "unpaid"
  }
}
```

## What “pass” means

The toy passes only if:

```text
Hamiltonian is Hermitian.
State norm stays near 1.
Bell rates are nonnegative.
Bell master-equation distribution tracks |ψ|².
Equivariance L1 error stays below tolerance.
Report contains no promotion language.
```

The toy fails if any of those fail.

The toy is inconclusive if numerical instability prevents judgment.

## Implementation path

Build it in this order:

```text
1. Basis
   Represent each fermion configuration as a uint64 bitstring.

2. Fermion operators
   Implement creation and annihilation with Jordan-Wigner sign.

3. Hamiltonian
   Build finite Kitaev chain:
   - chemical potential
   - hopping
   - pair creation
   - pair annihilation

4. Bell current
   For configurations n, m:
   J_nm = 2 Im(conj(ψ_n) H_nm ψ_m)

5. Bell rates
   rate n <- m = max(J_nm, 0) / |ψ_m|²

6. Evolution
   Use RK4 for ψ:
   dψ/dt = -i H ψ

   Use RK4 or Euler-small-step for ρ:
   dρ_n/dt = sum_m [rate n<-m ρ_m - rate m<-n ρ_n]

7. Audit
   Compare ρ_n(t) with |ψ_n(t)|² at every step.

8. Report
   Write JSON and Markdown.
```

For the first version, do **not** simulate random trajectories. You do not need them yet. The master-equation equivariance audit is cleaner and cheaper.

## Why no random trajectories first?

Because the first question is not:

> Do sampled histories look interesting?

The first question is:

> Did we implement Bell’s rate law correctly?

The master-equation check answers that directly. Random trajectories can come in `BELL-MIPT-0001.1`.

## Best next prompt to give the coding agent

```text
Implement BELL-MIPT-0001 in Go.

Create a small, non-bloated one-command Go program at cmd/bellmipt. It should read a JSON config from --config, or use a built-in default config if omitted, and write report.json plus report.md to --out.

Scope:
- finite Kitaev-chain-style fermion lattice only
- occupation-number bitstring basis
- Jordan-Wigner signs for fermion creation/annihilation
- Hamiltonian terms: chemical potential, hopping, pair creation, pair annihilation
- Bell probability current and positive-current jump rates
- Schrödinger evolution for ψ
- Bell master-equation evolution for ρ
- equivariance audit comparing ρ(t) to |ψ(t)|²
- final goal_status: toy_goal_passed, toy_goal_failed, or toy_goal_inconclusive

Do not implement MIPT, conditional wave functions, holography, black holes, Lean, agents, or random trajectory sampling in this ticket.

The output must explicitly state:
physics_claim: none
toy_analysis_only: true
needMap: unpaid
needToyCheck: partially_paid_rate_algebra_only
needObstruction: bell_jumps_are_not_measurements

Add tests for:
- basis enumeration
- fermion creation/annihilation signs
- Hamiltonian Hermiticity
- Bell current antisymmetry
- nonnegative rates
- equivariance audit on a small system
- deterministic output for fixed config/seed
- no forbidden promotion language in reports

Validation:
go test ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
```

That is the next step: **one input, one run, one honest report.**

