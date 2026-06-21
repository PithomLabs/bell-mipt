# BELL-MIPT-0001 Implementation Plan: One-Shot Go Toy Check

## 1. Purpose

Implement **BELL-MIPT-0001** as a small Go program that computes Bell jump rates for a finite fermionic Kitaev-chain-style lattice model and numerically verifies equivariance.

The toy goal is:

> Compute Bell probability currents and Bell jump rates for a finite fermionic lattice model, evolve both the wavefunction and Bell master-equation distribution, and check whether the master-equation distribution tracks (|\psi(t)|^2).

This ticket is strictly a **toy algebra/numerics check**.

It must not claim:

```text
No MIPT claim.
No holography claim.
No Bell-jumps-equal-measurements claim.
No conditional-wave-function bridge.
No physics promotion.
No proof of a Bell-MIPT bridge.
```

The report must explicitly preserve:

```json
{
  "toy_analysis_only": true,
  "physics_claim": "none"
}
```

---

## 2. Command Shape

Create one small command:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

If `--config` is omitted, the program uses a built-in default config.

If `--out` is omitted, default to:

```text
out/bellmipt-run
```

No subcommands.

No agents.

No random trajectories.

No MIPT simulation.

---

## 3. Proposed File Layout

```text
cmd/bellmipt/
  main.go

internal/bellmipt/
  config.go
  basis.go
  fermion.go
  hamiltonian.go
  current.go
  rates.go
  evolve.go
  audit.go
  report.go
  markdown.go
  forbidden.go

internal/bellmipt/testdata/
  small_default.json

internal/bellmipt/
  basis_test.go
  fermion_test.go
  hamiltonian_test.go
  current_test.go
  rates_test.go
  evolve_test.go
  report_test.go
```

This keeps the command thin and puts testable logic in `internal/bellmipt`.

---

## 4. Config Schema

Support this minimal config:

```go
type Config struct {
	SchemaVersion string       `json:"schema_version"`
	Model         string       `json:"model"`
	Sites         int          `json:"sites"`
	Boundary      string       `json:"boundary"`
	Parameters    Parameters   `json:"parameters"`
	InitialState  InitialState `json:"initial_state"`
	Time          TimeConfig   `json:"time"`
	Audit         AuditConfig  `json:"audit"`
}

type Parameters struct {
	Mu    float64 `json:"mu"`
	T     float64 `json:"t"`
	Delta float64 `json:"delta"`
}

type InitialState struct {
	Type string `json:"type"`
	Seed int64  `json:"seed"`
}

type TimeConfig struct {
	DT    float64 `json:"dt"`
	Steps int     `json:"steps"`
}

type AuditConfig struct {
	HermitianTolerance     float64 `json:"hermitian_tolerance"`
	NormTolerance         float64 `json:"norm_tolerance"`
	EquivarianceTolerance float64 `json:"equivariance_tolerance"`
}
```

Supported values for v0:

```text
schema_version: bell_mipt_toy_v0
model: finite_kitaev_chain
boundary: open | periodic
initial_state.type: random_normalized
```

Reject unsupported values clearly.

Default config:

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

Add a practical guardrail for v0:

```text
sites <= 12 by default
```

Reason: Hilbert dimension is (2^N). For `sites = 12`, the matrix dimension is 4096, already large for dense matrix evolution. The first version should stay simple and dense.

---

## 5. Basis Representation

Represent each occupation-number configuration as a `uint64` bitstring.

For `sites = N`, Hilbert dimension is:

```go
dim := 1 << N
```

Basis state index equals bitstring value:

```text
index 0  -> 000000
index 1  -> 000001
index 2  -> 000010
...
```

Types:

```go
type Basis struct {
	Sites int
	Dim   int
}
```

Functions:

```go
func NewBasis(sites int) (Basis, error)
func Occupied(state uint64, site int) bool
func CountBelow(state uint64, site int) int
```

Tests:

```text
sites=3 enumerates 8 states.
Bit occupancy is correct.
CountBelow gives correct Jordan-Wigner parity counts.
```

---

## 6. Fermion Operators

Implement fermion annihilation and creation using Jordan-Wigner signs.

For site `j`:

```text
c_j |n> = 0 if site j unoccupied
c_j |n> = (-1)^(number of occupied sites below j) |n with j cleared>

c†_j |n> = 0 if site j occupied
c†_j |n> = (-1)^(number of occupied sites below j) |n with j set>
```

Go shape:

```go
type OpResult struct {
	OK    bool
	State uint64
	Sign  float64
}

func Annihilate(state uint64, site int) OpResult
func Create(state uint64, site int) OpResult
```

Also add helper composition:

```go
func ApplyOps(state uint64, ops []FermionOp) OpResult
```

Important ordering rule:

If a Hamiltonian term is written as:

```text
c†_i c_j
```

then apply operators right-to-left to the ket:

```text
first c_j, then c†_i
```

Tests:

```text
c_0 |01> gives |00> with sign +1.
c_1 |10> gives zero.
c_1 |11> gives -|01> if site 0 is occupied.
c†_1 |01> gives zero.
c†_1 |00> gives |10> with sign +1.
```

---

## 7. Hamiltonian Construction

Build a dense complex Hamiltonian for the finite Kitaev-chain-style model.

Suggested Hamiltonian:

[
H =
-\mu \sum_i n_i
-t \sum_{\langle i,j\rangle} \left(c_i^\dagger c_j + c_j^\dagger c_i\right)
+\Delta \sum_{\langle i,j\rangle} \left(c_i^\dagger c_j^\dagger + c_j c_i\right)
]

Use real parameters for v0.

Boundary handling:

```text
open: bonds (0,1), (1,2), ..., (N-2,N-1)
periodic: open bonds plus (N-1,0)
```

Dense matrix type:

```go
type Matrix [][]complex128
```

Functions:

```go
func BuildKitaevHamiltonian(cfg Config, basis Basis) (Matrix, error)
func AddTerm(H Matrix, coeff complex128, ops []FermionOp)
func HermitianError(H Matrix) float64
```

For each basis ket `|m>`:

1. Apply operator term to `m`.
2. If nonzero, get output basis state `n`.
3. Add contribution to `H[n][m]`.

Hermiticity check:

```go
maxErr := max over n,m of abs(H[n][m] - conj(H[m][n]))
```

Tests:

```text
Hamiltonian is Hermitian for open boundary.
Hamiltonian is Hermitian for periodic boundary.
Hamiltonian has expected dimension 2^N.
Hamiltonian rejects unsupported model/boundary.
```

---

## 8. Initial State

Support only:

```text
random_normalized
```

Use deterministic `math/rand` with the provided seed.

Generate each complex component from two normal random values:

```go
re := rng.NormFloat64()
im := rng.NormFloat64()
psi[i] = complex(re, im)
```

Normalize:

```go
psi[i] /= sqrt(sum_i |psi[i]|^2)
```

Initialize Bell distribution as:

```go
rho[i] = abs2(psi[i])
```

Functions:

```go
func RandomNormalizedState(dim int, seed int64) []complex128
func Probabilities(psi []complex128) []float64
func Norm(psi []complex128) float64
```

Tests:

```text
Same seed gives same state.
Norm is 1 within tolerance.
rho sums to 1.
```

---

## 9. Bell Current

Use the Bell current:

[
J_{n m} = 2,\mathrm{Im}\left(\overline{\psi_n} H_{n m} \psi_m\right)
]

Implement:

```go
func BellCurrent(H Matrix, psi []complex128) [][]float64
```

Expected property:

```text
J_nm = -J_mn
```

up to floating-point tolerance, assuming `H` is Hermitian.

Tests:

```text
Bell current is antisymmetric.
Diagonal current is approximately zero.
```

---

## 10. Bell Jump Rates

Define positive-current rates:

[
\sigma_{n \leftarrow m}
=======================

\frac{\max(J_{n m}, 0)}{|\psi_m|^2}
]

If (|\psi_m|^2) is very small, avoid division instability.

Use a small epsilon:

```go
const ProbabilityFloor = 1e-14
```

If denominator is below the floor:

```text
rate n<-m = 0 unless numerator is also safely negligible.
```

Track whether a probability-floor event occurred. If such events become numerically significant, mark the run inconclusive.

Functions:

```go
func BellRates(J [][]float64, psi []complex128) (rates [][]float64, stats RateStats)
```

Metrics:

```go
type RateStats struct {
	MaxNegativeRateViolation float64
	MeanTotalActivity       float64
	MaxTotalActivity        float64
	ProbabilityFloorHits    int
}
```

Here, total Bell activity for source state `m` may be:

[
A_m = \sum_n \sigma_{n \leftarrow m}
]

Then report mean and max activity over time.

Tests:

```text
All rates are nonnegative.
Rates are zero when current is negative.
Positive-current direction is assigned correctly.
Probability-floor behavior does not panic.
```

---

## 11. Evolution

Evolve the wavefunction by Schrödinger dynamics:

[
\frac{d\psi}{dt} = -i H \psi
]

Evolve the Bell master equation:

[
\frac{d\rho_n}{dt}
==================

\sum_m
\left(
\sigma_{n \leftarrow m}\rho_m
-----------------------------

\sigma_{m \leftarrow n}\rho_n
\right)
]

The rates depend on the current wavefunction (\psi(t)), so evolve the pair ((\psi,\rho)) together.

Use RK4 for both:

```go
func Derivative(H Matrix, psi []complex128, rho []float64) (dpsi []complex128, drho []float64, stats RateStats)

func RK4Step(H Matrix, psi []complex128, rho []float64, dt float64) (nextPsi []complex128, nextRho []float64, stepStats StepStats)
```

At each RK4 stage:

1. Compute `dpsi` from current stage `psi`.
2. Compute Bell currents from current stage `psi`.
3. Compute rates from current stage `psi`.
4. Compute `drho` using those rates and current stage `rho`.

After every full step:

```text
Optionally renormalize ψ only if tiny numerical drift appears.
Do not silently hide major norm drift.
Clamp tiny negative rho values only if within numerical tolerance.
If rho has serious negativity, mark inconclusive or failed.
```

Recommended:

```text
Do not renormalize ψ by default.
Measure norm drift honestly.
For rho, allow tiny negative values like -1e-12, but report max violation.
```

Tests:

```text
Small system evolves without NaN/Inf.
Norm remains near 1 for small dt.
rho sum remains near 1.
Equivariance error remains small for a small stable config.
```

---

## 12. Equivariance Audit

At every time step:

```go
born := Probabilities(psi)
err := L1Distance(rho, born)
```

Track:

```go
maxEquivarianceL1Error
finalEquivarianceL1Error
meanEquivarianceL1Error
maxNormError
maxRhoSumError
maxRhoNegativeViolation
```

Main pass condition:

```text
max_equivariance_l1_error <= config.audit.equivariance_tolerance
```

Also check:

```text
Hamiltonian hermitian error <= hermitian_tolerance
State norm error <= norm_tolerance
Rates nonnegative
No serious rho negativity
No NaN/Inf
No forbidden promotion language
```

Goal status logic:

```go
if numericalInstability {
    goal_status = "toy_goal_inconclusive"
} else if allPass {
    goal_status = "toy_goal_passed"
} else {
    goal_status = "toy_goal_failed"
}
```

Numerical instability includes:

```text
NaN or Inf in ψ, ρ, current, rates, or metrics.
Serious rho negativity.
Massive norm/rho-sum drift preventing meaningful comparison.
Probability-floor events that materially affect rates.
```

---

## 13. Report JSON

Write:

```text
out/bellmipt-run/input.json
out/bellmipt-run/report.json
out/bellmipt-run/report.md
```

Report shape:

```go
type Report struct {
	SchemaVersion string            `json:"schema_version"`
	ToyID         string            `json:"toy_id"`
	ToyAnalysisOnly bool            `json:"toy_analysis_only"`
	PhysicsClaim string             `json:"physics_claim"`
	Model         string            `json:"model"`
	Sites         int               `json:"sites"`
	HilbertDim    int               `json:"hilbert_dim"`
	Goal          string            `json:"goal"`
	GoalStatus    string            `json:"goal_status"`
	Checks        Checks            `json:"checks"`
	Metrics       Metrics           `json:"metrics"`
	DebtStatus    map[string]string `json:"debt_status"`
	Limitations   []string          `json:"limitations"`
	ForbiddenLanguageAudit ForbiddenLanguageAudit `json:"forbidden_language_audit"`
}
```

Required report values:

```json
{
  "schema_version": "bell_mipt_report_v0",
  "toy_id": "BELL-MIPT-0001",
  "toy_analysis_only": true,
  "physics_claim": "none",
  "goal": "compute_bell_rates_and_verify_equivariance"
}
```

Debt status:

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

Limitations should include:

```text
This checks Bell-rate algebra in a finite toy model only.
This does not implement MIPT.
This does not show Bell jumps are measurements.
This does not construct a conditional-wave-function bridge.
This does not support any holography or black-hole claim.
This is not a physics promotion.
```

---

## 14. Markdown Report

`report.md` should be short and reviewable:

```markdown
# BELL-MIPT-0001 Report

## Status

Goal status: toy_goal_passed

## Scope

Toy analysis only: true  
Physics claim: none

## What Was Checked

- Finite Kitaev-chain-style Hamiltonian construction
- Hermiticity
- Bell probability current antisymmetry
- Bell jump-rate nonnegativity
- Schrödinger evolution of ψ
- Bell master-equation evolution of ρ
- L1 comparison between ρ(t) and |ψ(t)|²

## Metrics

| Metric | Value |
|---|---:|
| max_hermitian_error | ... |
| max_norm_error | ... |
| max_equivariance_l1_error | ... |

## EBP Debt Status

| Debt | Status |
|---|---|
| needMap | unpaid |
| needInvariant | partially_paid_equivariance_only |
| needToyCheck | partially_paid_rate_algebra_only |
| needNullModel | unpaid |
| needObstruction | bell_jumps_are_not_measurements |
| needFaithfulnessReview | unpaid |

## Limitations

- No MIPT claim.
- No holography claim.
- No Bell-jumps-equal-measurements claim.
- No conditional-wave-function bridge.
- No physics promotion.
```

---

## 15. Forbidden Language Audit

Add a simple forbidden phrase scanner for both JSON and Markdown report strings.

Forbidden examples:

```text
proves the Bell-MIPT bridge
proven bridge
establishes MIPT
confirms holography
solves
breakthrough
validated theory
physics promotion
Bell jumps are measurements
```

Allowed terms:

```text
toy_goal_passed
toy analysis
numerical check
equivariance audit
rate algebra
```

Function:

```go
func AuditForbiddenLanguage(text string) ForbiddenLanguageAudit
```

Report field:

```go
type ForbiddenLanguageAudit struct {
	Passed bool     `json:"passed"`
	Hits   []string `json:"hits"`
}
```

Tests:

```text
Reports contain no forbidden promotion language.
Scanner catches known bad phrases.
```

---

## 16. Test Plan

Required tests:

### Basis

```text
TestBasisEnumeration
TestOccupied
TestCountBelow
```

### Fermion operators

```text
TestAnnihilationSigns
TestCreationSigns
TestApplyOpsRightToLeft
TestNilOnInvalidCreationOrAnnihilation
```

### Hamiltonian

```text
TestHamiltonianHermitianOpen
TestHamiltonianHermitianPeriodic
TestHamiltonianDimension
TestUnsupportedBoundaryRejected
```

### Bell current

```text
TestBellCurrentAntisymmetric
TestBellCurrentDiagonalZero
```

### Bell rates

```text
TestBellRatesNonnegative
TestBellRatesPositiveCurrentDirection
TestBellRatesProbabilityFloorSafe
```

### Evolution and audit

```text
TestEquivarianceAuditSmallSystem
TestNoNaNInfDuringEvolution
TestNormPreservedSmallSystem
TestRhoTracksBornDistribution
```

### Output

```text
TestDeterministicOutputFixedSeed
TestReportContainsRequiredDebtStatus
TestNoForbiddenPromotionLanguage
```

Validation command:

```bash
go test ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
```

Optional stronger validation:

```bash
go test -race ./...
```

---

## 17. Execution Flow

Program flow:

```text
1. Parse --config and --out.
2. Load default config or config file.
3. Validate config.
4. Create output directory.
5. Write normalized input.json.
6. Build basis.
7. Build Hamiltonian.
8. Check Hermiticity.
9. Generate deterministic random normalized ψ.
10. Set ρ = |ψ|².
11. For each time step:
    - Compute Bell current.
    - Compute rates.
    - RK4 evolve ψ and ρ.
    - Compute norm error.
    - Compute rho sum error.
    - Compute equivariance L1 error.
    - Track activity and violations.
12. Decide goal_status.
13. Build report.json.
14. Build report.md.
15. Run forbidden-language audit.
16. Write outputs.
```

---

## 18. Goal Status Rules

### toy_goal_passed

Only if all are true:

```text
Hamiltonian is Hermitian.
State norm stays within tolerance.
Rates are nonnegative.
No serious rho negativity.
Bell master-equation ρ(t) tracks |ψ(t)|² within tolerance.
No NaN/Inf.
Report contains no forbidden promotion language.
```

### toy_goal_failed

Use this if the run is numerically meaningful but one or more checks fail:

```text
Hermiticity fails.
Norm preservation fails.
Rates become negative beyond tolerance.
Equivariance error exceeds tolerance.
Forbidden promotion language appears.
```

### toy_goal_inconclusive

Use this if numerical instability prevents judgment:

```text
NaN/Inf appears.
Probability floor dominates rate calculation.
ρ becomes badly invalid.
Time step is too large for meaningful comparison.
```

---

## 19. EBP 2.1 Claim Ledger

### Claim 1

BELL-MIPT-0001 computes Bell currents and Bell positive-current jump rates for a finite fermionic lattice toy model.

Status:

```text
implementation_target
```

Debt:

```text
needToyCheck: partially payable by this ticket
needFaithfulnessReview: unpaid
```

### Claim 2

The master-equation distribution should numerically track (|\psi(t)|^2) when Bell rates are implemented consistently.

Status:

```text
toy_check_target
```

Debt:

```text
needInvariant: partially_paid_equivariance_only if the audit passes
needNullModel: unpaid
needObstruction: bell_jumps_are_not_measurements
```

### Claim 3

This toy justifies later bridge work.

Status:

```text
not_promoted
```

Debt:

```text
needMap: unpaid
needNullModel: unpaid
needFaithfulnessReview: unpaid
```

Important boundary:

```text
Passing BELL-MIPT-0001 only means the finite Bell-rate algebra/equivariance toy behaved as expected.
It does not establish a Bell-MIPT bridge.
```

---

## 20. Non-Goals for This Ticket

Do not implement:

```text
MIPT circuits
measurement-induced phase transitions
conditional wave functions
holography
black-hole information claims
Lean theorem proving
random trajectory sampling
AI agents
multi-command CLI
visual dashboard
database persistence
web UI
```

Random trajectories can be a later ticket:

```text
BELL-MIPT-0001.1
```

Null models can be a later ticket:

```text
BELL-MIPT-0002
```

Bridge mapping can be a later ticket:

```text
BELL-MIPT-0003
```

---

## 21. Acceptance Criteria

The ticket is complete when:

```text
go test ./... passes.
go run ./cmd/bellmipt --out /tmp/bellmipt-default runs successfully.
The output directory contains input.json, report.json, and report.md.
report.json has schema_version bell_mipt_report_v0.
report.json has toy_id BELL-MIPT-0001.
report.json has toy_analysis_only true.
report.json has physics_claim none.
goal_status is one of:
  - toy_goal_passed
  - toy_goal_failed
  - toy_goal_inconclusive
The report includes the required EBP debt statuses.
The report includes explicit limitations.
The forbidden-language audit passes.
No MIPT, holography, conditional-wave-function, or random trajectory code is added.
```

---

## 22. Recommended Coding Agent Prompt

```text
Implement BELL-MIPT-0001 in Go.

Create a small one-command Go program at cmd/bellmipt. It should read a JSON config from --config, or use a built-in default config if omitted, and write input.json, report.json, and report.md to --out.

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

Use RK4 for the coupled pair (ψ, ρ), with Bell rates recomputed from ψ at each RK4 stage.

Do not implement MIPT, conditional wave functions, holography, black holes, Lean, agents, or random trajectory sampling in this ticket.

The output must explicitly state:
physics_claim: none
toy_analysis_only: true
needMap: unpaid
needInvariant: partially_paid_equivariance_only
needToyCheck: partially_paid_rate_algebra_only
needObstruction: bell_jumps_are_not_measurements
needFaithfulnessReview: unpaid

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

Return a concise implementation report with files changed, tests run, goal_status from the default run, and any remaining debts.
```

