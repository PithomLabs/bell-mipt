# Code Review Prompt: BELL-MIPT-0001 Source/Artifact Audit

You are reviewing the source code for **BELL-MIPT-0001**, a Go one-shot toy check for Bell-type jump rates on a finite Kitaev-chain-style fermionic lattice.

Your job is to determine whether the implementation faithfully matches the intended limited scope:

> Compute Bell probability currents and Bell positive-current jump rates for a finite fermionic lattice model, evolve both the wavefunction ψ and Bell master-equation distribution ρ, and numerically verify equivariance: ρ(t) tracks |ψ(t)|².

This review is **not** a physics promotion. It must not claim a Bell-MIPT bridge, MIPT behavior, holography, conditional-wave-function mechanism, or black-hole information result.

## 0. Expected Scope

The implementation should contain:

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
internal/bellmipt/*_test.go
bellmipt.json
go.mod
```

The command should be:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

If `--config` is omitted, it should use a built-in default config.

The output should include:

```text
input.json
report.json
report.md
```

## 1. Hard Non-Goals

Reject or flag any code/report language that implies:

```text
MIPT was observed.
The Bell-MIPT bridge was proved.
Bell jumps are measurements.
A conditional-wave-function bridge was constructed.
Holography was explained.
Black-hole information was addressed.
Bohmian mechanics was validated as physics.
```

Allowed conclusion if the implementation is correct:

```text
The finite Bell-rate algebra and numerical equivariance audit passed for the tested toy configuration.
```

## 2. Review Commands

Run:

```bash
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
go run ./cmd/bellmipt --config bellmipt.json --out /tmp/bellmipt-custom
```

Then inspect:

```bash
cat /tmp/bellmipt-default/report.json
cat /tmp/bellmipt-default/report.md
```

Verify that:

```text
report.json exists
report.md exists
input.json exists
goal_status is one of toy_goal_passed, toy_goal_failed, toy_goal_inconclusive
toy_analysis_only is true
physics_claim is "none"
debt_status fields are present
limitations are present
forbidden_language_audit.passed is true
```

If the run passes, record the key metrics:

```text
max_hermitian_error
max_norm_error
max_rho_sum_error
max_rho_negative_violation
max_current_antisymmetry_error
max_rate_negative_violation
max_equivariance_l1_error
final_equivariance_l1_error
mean_equivariance_l1_error
mean_total_bell_activity
max_total_bell_activity
probability_floor_hits
```

## 3. High-Risk Code Areas to Inspect

### A. Fermion basis

Verify:

```text
Basis states are uint64 bitstrings.
Bit i = occupation at site i.
Hilbert dimension is 2^sites.
sites are capped at 10 for v0.
```

Check:

```text
NewBasis
Occupied
PopCount
CountBelow
EnumerateStates
```

Expected behavior:

```text
sites=3 gives 8 states: 0..7.
CountBelow(state, site) counts occupied bits strictly below site.
```

### B. Jordan-Wigner fermion operators

Inspect:

```text
ApplyAnnihilate
ApplyCreate
ApplyOps
```

Verify convention:

```text
c_j |n> = 0 if site j unoccupied
c_j |n> = (-1)^(CountBelow(n,j)) |n with bit j cleared>

c†_j |n> = 0 if site j occupied
c†_j |n> = (-1)^(CountBelow(n,j)) |n with bit j set>
```

Verify `ApplyOps`:

```text
Input operators are in physics notation order, left-to-right.
Execution is right-to-left on the ket.
Signs are computed on the intermediate state before each operator flips its bit.
```

Load-bearing tests:

```text
TestApplyOpsRightToLeft
TestFermionAnticommutationSmallBasis
```

The review should explicitly confirm whether `{c_i, c†_j} = δ_ij` is tested on all basis states for sites=2 and sites=3.

### C. Periodic boundary signs

This is a critical review point.

The implementation should **not** add an extra manual parity factor for the periodic wrap bond.

Correct approach:

```text
The periodic bond (N-1,0) uses the same generic ApplyOps machinery as all other bonds.
No separate (-1)^Nhat correction is added.
```

Flag as a bug if code special-cases the periodic boundary with an extra parity sign.

Also flag as a bug if the code uses an “adjacent sites always sign +1” shortcut, because that is wrong for the wrap bond in a linear Jordan-Wigner ordering.

### D. Hamiltonian construction

Inspect:

```text
BuildKitaevHamiltonian
HermitianError
Matrix.At/Add/Set
```

Expected matrix convention:

```text
H[n][m] = <n|H|m>
row = destination state n
column = source state m
```

Expected Hamiltonian convention:

```text
H =
  - μ Σ_i n_i
  - t Σ_<i,j> (c†_i c_j + c†_j c_i)
  + Δ Σ_<i,j> (c†_i c†_j + c_j c_i)
```

For v0, the pairing sign should be `+Δ`, exactly as specified.

Verify:

```text
Open bonds: (0,1), ..., (N-2,N-1)
Periodic bonds: open bonds plus (N-1,0)
```

Critical rule:

```text
The code must build all Hermitian-conjugate terms explicitly.
It must not symmetrize H after construction to hide sign errors.
```

Flag if the code post-processes H to force Hermiticity.

Expected tests:

```text
TestHamiltonianHermitianOpen
TestHamiltonianHermitianPeriodic
TestHamiltonianDimension
TestHamiltonianDiagonalWhenNoHoppingNoPairing
TestPeriodicBoundaryUsesGenericJWNoSpecialParity
```

### E. Bell probability current

Inspect:

```text
BellCurrent
```

Expected formula:

```text
J_nm = 2 * Im(conj(psi[n]) * H[n,m] * psi[m])
```

Expected property:

```text
J_nm = -J_mn
J_nn ≈ 0
```

Confirm tests:

```text
TestBellCurrentAntisymmetric
TestBellCurrentDiagonalZero
```

Flag if indices are swapped or formula uses `conj(psi[m])` instead of `conj(psi[n])`.

### F. Bell jump rates

Inspect:

```text
BellRates
```

Expected formula:

```text
sigma(n <- m) = max(J_nm, 0) / |psi[m]|²
```

Expected behavior:

```text
Rates are nonnegative.
Negative current gives zero rate.
If |psi[m]|² < ProbabilityFloor, outgoing rates from m are set to zero and a floor hit is recorded.
```

Check:

```text
ProbabilityFloor = 1e-14 or equivalent
probability_floor_hits tracked
probability-floor behavior affects goal_status only according to documented rule
```

Flag if the code divides by zero or silently produces Inf/NaN.

### G. Evolution coupling

Inspect:

```text
Derivative
RK4Step
```

Expected:

```text
ψ evolves by dψ/dt = -i H ψ.
ρ evolves by Bell master equation:
dρ_n/dt = Σ_m [sigma(n <- m) ρ_m - sigma(m <- n) ρ_n].
```

Important requirement:

```text
The implementation should use joint RK4 for (ψ, ρ).
At every RK4 sub-stage, recompute Bell current and Bell rates from the stage ψ.
```

Do not accept a hidden “rates at start of step only” implementation unless clearly documented as not used in v0. For BELL-MIPT-0001, joint RK4 recomputation should be the only supported strategy.

Confirm:

```text
k1 uses rates from ψ
k2 uses rates from ψ + dt*k1ψ/2
k3 uses rates from ψ + dt*k2ψ/2
k4 uses rates from ψ + dt*k3ψ
```

Flag if `rates` are computed only once per time step.

### H. Norm and rho handling

Check:

```text
ψ is not silently renormalized every step.
Norm drift is measured honestly.
ρ sum is tracked.
Small numerical negativity is reported.
Large negativity fails or makes run inconclusive.
NaN/Inf is detected.
```

Flag if the implementation renormalizes ψ or clamps ρ without reporting it.

### I. Equivariance audit

Inspect:

```text
L1Distance
AuditAccumulator
DetermineGoalStatus
```

Expected:

```text
born = Probabilities(psi)
equivariance_l1_error = Σ_n |rho[n] - born[n]|
max, final, and mean equivariance errors are reported
```

Pass only if:

```text
Hamiltonian Hermitian error <= hermitian_tolerance
ψ norm error <= norm_tolerance
ρ sum error <= norm_tolerance
current antisymmetry passes
rates nonnegative
equivariance L1 error <= equivariance_tolerance
no NaN/Inf
forbidden-language audit passes
```

### J. Forbidden-language audit

Inspect:

```text
AuditForbiddenLanguage
```

Expected:

```text
It catches positive promotional claims.
It allows negated limitation statements.
```

It must allow:

```text
No MIPT claim.
No holography claim.
This does not show Bell jumps are measurements.
This is not a physics promotion.
```

It should catch:

```text
proves the Bell-MIPT bridge
establishes MIPT
confirms holography
Bell jumps are measurements
Bell jumps equal measurements
explains black holes
explains holography
```

Flag if `report.json` uses `"hits": null`; this is minor, not blocking. Prefer `"hits": []`.

## 4. Artifact Contract

Inspect generated `report.json`.

Required fields:

```json
{
  "schema_version": "bell_mipt_report_v0",
  "toy_id": "BELL-MIPT-0001",
  "toy_analysis_only": true,
  "physics_claim": "none",
  "model": "finite_kitaev_chain",
  "goal": "compute_bell_rates_and_verify_equivariance",
  "goal_status": "toy_goal_passed | toy_goal_failed | toy_goal_inconclusive"
}
```

Required debt status:

```json
{
  "needMap": "unpaid",
  "needInvariant": "partially_paid_equivariance_only",
  "needToyCheck": "partially_paid_rate_algebra_only",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "unpaid"
}
```

Required limitations:

```text
This checks Bell-rate algebra in a finite toy model only.
This does not implement MIPT.
This does not show Bell jumps are measurements.
This does not construct a conditional-wave-function bridge.
This does not support any holography or black-hole claim.
This is not a physics promotion.
```

## 5. Review Output Format

Return a structured review with these sections.

### 1. Executive Verdict

Choose one:

```text
accept_for_limited_toy_scope
accept_with_minor_repairs
reject_pending_repairs
inconclusive_needs_more_artifacts
```

### 2. Validation Commands Run

List commands and results:

```text
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
```

### 3. Artifact Summary

Report:

```text
goal_status
max_hermitian_error
max_norm_error
max_rho_sum_error
max_current_antisymmetry_error
max_rate_negative_violation
max_equivariance_l1_error
final_equivariance_l1_error
mean_equivariance_l1_error
mean_total_bell_activity
max_total_bell_activity
probability_floor_hits
forbidden_language_audit result
```

### 4. Code Findings

Organize by severity:

```text
Critical blockers
High-priority repairs
Medium repairs
Low-priority polish
```

### 5. Physics/Math Faithfulness Review

Answer:

```text
Was the Bell current formula implemented correctly?
Was the Bell rate formula implemented correctly?
Were fermionic signs handled correctly?
Was the periodic boundary handled without extra parity double-counting?
Was H checked rather than force-symmetrized?
Were rates recomputed at RK4 sub-stages?
Did the equivariance audit actually compare ρ(t) against |ψ(t)|²?
```

### 6. EBP 2.1 Ledger

Use:

```json
{
  "needMap": "unpaid",
  "needInvariant": "partially_paid_equivariance_only",
  "needToyCheck": "partially_paid_rate_algebra_only",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_reviewed | partially_reviewed | unpaid",
  "promotion_status": "unpromoted_toy_check_only"
}
```

### 7. Final Recommendation

Choose one:

```text
Accept BELL-MIPT-0001 as complete for finite toy equivariance scope.
Accept after minor artifact repairs.
Request code repairs before acceptance.
Request more artifacts/source files.
```

## 6. Important Interpretation Boundary

Even if the code passes all checks, the allowed conclusion is only:

```text
The finite Bell-rate algebra and numerical equivariance audit passed for the tested toy configuration.
```

Do **not** conclude:

```text
Bell jumps behave like measurements.
Bell-MIPT bridge is supported.
MIPT was observed.
Holography was explained.
Bohmian QFT was validated.
```

