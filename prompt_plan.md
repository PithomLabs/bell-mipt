Below is the context and the coding-agent prompt I would use. The prompt is written to be handed directly to a Go coding agent.

The source plans agree on the locked scope: **BELL-MIPT-0001 is only a finite Bell-rate algebra/equivariance toy**, not a MIPT simulation, not holography, not conditional wave functions, and not a proof of a Bell–MIPT bridge. The strongest base is `plan_chatgpt.md`, which defines the purpose, one-command shape, config schema, basis representation, fermion operators, Hamiltonian construction, Bell currents/rates, evolution, report structure, limitations, and acceptance criteria.  Opus adds the key correction that in full Fock space the Jordan-Wigner string already handles the periodic wrap bond, so **no extra boundary parity correction should be added**.  Kimi and DeepSeek contribute the clearest compact test matrix and validation commands. 

Use this as the final coding-agent prompt:

# Coding Agent Prompt: Implement BELL-MIPT-0001 in Go

You are implementing **BELL-MIPT-0001**, a one-shot Go toy check.

Your job is to create a small, non-bloated Go program that computes Bell jump rates for a finite fermionic Kitaev-chain-style lattice model and numerically verifies equivariance.

This is a **toy algebra/numerics check only**.

The central question is:

> Can we compute Bell probability currents and Bell jump rates for a finite fermion model, evolve both the wavefunction and the Bell master-equation distribution, and verify that the Bell master-equation distribution tracks (|\psi(t)|^2)?

## 0. Locked Scope

Implement only:

* finite Kitaev-chain-style fermion lattice
* occupation-number bitstring basis
* Jordan-Wigner signed fermion creation/annihilation operators
* dense finite Hamiltonian construction
* Bell probability current
* positive-current Bell jump rates
* Schrödinger evolution for (\psi)
* Bell master-equation evolution for (\rho)
* equivariance audit comparing (\rho(t)) with (|\psi(t)|^2)
* JSON + Markdown report
* tests

Do **not** implement:

* MIPT simulation
* monitored quantum circuits
* projective measurements
* conditional wave functions
* entanglement entropy
* mutual information
* purity
* holography
* black-hole information
* Lean theorem proving
* AI agents
* random trajectory sampling
* database persistence
* dashboards
* web UI
* multi-command CLI

The report must explicitly state:

```json
{
  "toy_analysis_only": true,
  "physics_claim": "none"
}
```

Passing this ticket means only:

> The finite Bell-rate algebra/equivariance toy behaved as expected.

It does **not** establish a Bell–MIPT bridge.

---

# 1. Implementation Style

Use Go only.

Prefer zero external dependencies. Use only the standard library unless the repository already has an approved dependency policy.

Use:

* `encoding/json`
* `flag`
* `fmt`
* `math`
* `math/cmplx`
* `math/rand`
* `os`
* `path/filepath`
* `strings`
* `testing`

Do not use Gonum for v0. The default model is tiny: 6 sites, Hilbert dimension (2^6 = 64). Hand-rolled dense `complex128` operations are sufficient and more auditable.

Use a single command:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

If `--config` is omitted, use a built-in default config.

If `--out` is omitted, default to:

```text
out/bellmipt-run
```

No subcommands.

---

# 2. Recommended File Layout

Use this simple layout:

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
  run.go

internal/bellmipt/testdata/
  small_default.json

internal/bellmipt/
  config_test.go
  basis_test.go
  fermion_test.go
  hamiltonian_test.go
  current_test.go
  rates_test.go
  evolve_test.go
  audit_test.go
  report_test.go
```

Keep `cmd/bellmipt/main.go` thin. It should parse flags, load config, call `bellmipt.Run`, and write outputs.

Put testable logic in `internal/bellmipt`.

---

# 3. Config Schema

Implement this config:

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

Validation rules:

```text
schema_version must be bell_mipt_toy_v0
model must be finite_kitaev_chain
boundary must be open or periodic
initial_state.type must be random_normalized
sites must be >= 2
sites must be <= 10 for v0
dt must be > 0
steps must be > 0
all tolerances must be > 0
```

Use `sites <= 10` for v0, even though some plans allow 12. This keeps dense evolution safe and fast.

---

# 4. Basis Representation

Represent each occupation-number configuration as a `uint64` bitstring.

For `sites = N`, Hilbert dimension is:

```go
dim := 1 << N
```

Basis state index equals bitstring value:

```text
index 0 -> 000000
index 1 -> 000001
index 2 -> 000010
...
```

Implement:

```go
type Basis struct {
	Sites int
	Dim   int
}

func NewBasis(sites int) (Basis, error)
func Occupied(state uint64, site int) bool
func CountBelow(state uint64, site int) int
func PopCount(state uint64) int
func EnumerateStates(b Basis) []uint64
```

`CountBelow(state, site)` returns the number of occupied sites with index `< site`. This is used for Jordan-Wigner signs.

Use `math/bits.OnesCount64` if convenient.

Tests:

```text
sites=3 enumerates 8 states.
Bit occupancy is correct.
PopCount is correct.
CountBelow gives correct Jordan-Wigner parity counts.
```

---

# 5. Fermion Operators

Implement fermion annihilation and creation using Jordan-Wigner signs.

For site `j`:

```text
c_j |n> = 0 if site j unoccupied
c_j |n> = (-1)^(number of occupied sites below j) |n with j cleared>

c†_j |n> = 0 if site j occupied
c†_j |n> = (-1)^(number of occupied sites below j) |n with j set>
```

Use:

```go
type OpKind int

const (
	Create OpKind = iota
	Annihilate
)

type FermionOp struct {
	Kind OpKind
	Site int
}

type OpResult struct {
	OK    bool
	State uint64
	Sign  int
}

func ApplyAnnihilate(state uint64, site int) OpResult
func ApplyCreate(state uint64, site int) OpResult
func ApplyOps(state uint64, ops []FermionOp) OpResult
```

Important rule:

If a Hamiltonian term is written as:

```text
c†_i c_j
```

then apply operators right-to-left to the ket:

```text
first c_j, then c†_i
```

Therefore `ApplyOps` should accept operators in physics notation order, left-to-right, but execute them from right to left.

Example:

```go
ApplyOps(state, []FermionOp{
	{Kind: Create, Site: i},
	{Kind: Annihilate, Site: j},
})
```

should first apply annihilation at `j`, then creation at `i`.

## Periodic Boundary Rule

Do **not** add a special extra parity sign for the periodic boundary bond.

In full Fock space, the Jordan-Wigner sign from generic `Create` and `Annihilate` operators already handles the wrap bond `(N-1, 0)`.

For example, for `c_0† c_{N-1}`, execute:

```text
first c_{N-1}
then c_0†
```

The signs compose from the actual intermediate states.

Do not add a separate `(-1)^Nhat` correction. That risks double-counting.

Tests:

```text
c_0 |01> gives |00> with sign +1.
c_1 |10> gives zero.
c_1 |11> gives -|01> if site 0 is occupied.
c†_1 |01> gives zero.
c†_1 |00> gives |10> with sign +1.
ApplyOps applies right-to-left.
Anticommutation {c_i, c_j†} = δ_ij holds on all basis states for sites=2 and sites=3.
Periodic wrap terms use generic ApplyOps, not special-case parity.
```

---

# 6. Dense Matrix Representation

Use a dense flat row-major matrix:

```go
type Matrix struct {
	Dim  int
	Data []complex128
}

func NewMatrix(dim int) Matrix
func (m Matrix) At(row, col int) complex128
func (m Matrix) Add(row, col int, v complex128)
func (m Matrix) Set(row, col int, v complex128)
```

Use row = destination state `n`, column = source state `m`, so:

```text
H[n][m] = <n|H|m>
```

Flat index:

```go
idx := row*m.Dim + col
```

---

# 7. Hamiltonian Construction

Build a dense complex Hamiltonian for the finite Kitaev-chain-style model.

Use the convention:

```text
H =
  - μ Σ_i n_i
  - t Σ_<i,j> (c†_i c_j + c†_j c_i)
  + Δ Σ_<i,j> (c†_i c†_j + c_j c_i)
```

Use real parameters for v0.

Boundary handling:

```text
open:
  bonds (0,1), (1,2), ..., (N-2,N-1)

periodic:
  open bonds plus (N-1,0)
```

Implement:

```go
func BuildKitaevHamiltonian(cfg Config, basis Basis) (Matrix, error)
func HermitianError(H Matrix) float64
```

Construction algorithm:

```text
For each basis ket |m>:
  1. Chemical potential:
     For each site i:
       if occupied, add -mu to H[m][m].

  2. For each bond (i,j):
     Add hopping:
       -t * c†_i c_j
       -t * c†_j c_i

     Add pairing:
       +delta * c†_i c†_j
       +delta * c_j c_i

  3. For each operator string:
       ApplyOps(m, ops)
       If nonzero, get destination n and sign.
       Add coeff * sign to H[n][m].
```

Notes:

* Use the same generic `ApplyOps` for all terms, including periodic wrap terms.
* Do not symmetrize the Hamiltonian after construction to hide sign mistakes. Build all Hermitian-conjugate terms explicitly and then check Hermiticity.
* Hermiticity check should be a real audit, not a repair.

Tests:

```text
Hamiltonian has dimension 2^N.
Hamiltonian is Hermitian for open boundary.
Hamiltonian is Hermitian for periodic boundary.
Hamiltonian rejects unsupported model/boundary.
For sites=2 and simple parameters, matrix entries match hand-calculated expected values.
For t=0, delta=0, Hamiltonian is diagonal with chemical-potential entries only.
```

---

# 8. Initial State

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

Implement:

```go
func RandomNormalizedState(dim int, seed int64) []complex128
func Probabilities(psi []complex128) []float64
func Norm(psi []complex128) float64
```

Tests:

```text
Same seed gives same state.
Different seed usually gives different state.
Norm is 1 within tolerance.
rho sums to 1.
```

---

# 9. Bell Probability Current

Use the Bell current:

```text
J_nm = 2 * Im(conj(psi[n]) * H[n,m] * psi[m])
```

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

# 10. Bell Jump Rates

Define positive-current rates:

```text
sigma(n <- m) = max(J_nm, 0) / |psi[m]|^2
```

Use a small probability floor:

```go
const ProbabilityFloor = 1e-14
```

If `|psi[m]|^2 < ProbabilityFloor`, set all outgoing rates from `m` to zero and record a probability-floor hit.

If probability-floor hits become numerically significant, mark the run `toy_goal_inconclusive`.

Implement:

```go
type RateStats struct {
	MaxNegativeRateViolation float64 `json:"max_negative_rate_violation"`
	MeanTotalActivity       float64 `json:"mean_total_bell_activity"`
	MaxTotalActivity        float64 `json:"max_total_bell_activity"`
	ProbabilityFloorHits    int     `json:"probability_floor_hits"`
}

func BellRates(J [][]float64, psi []complex128) (rates [][]float64, stats RateStats)
```

Tests:

```text
All rates are nonnegative.
Rates are zero when current is negative.
Positive-current direction is assigned correctly.
Probability-floor behavior does not panic.
```

---

# 11. Evolution

Evolve the wavefunction by Schrödinger dynamics:

```text
dψ/dt = -i H ψ
```

Evolve the Bell master equation:

```text
dρ_n/dt = Σ_m [ sigma(n <- m) * rho_m - sigma(m <- n) * rho_n ]
```

The rates depend on the current wavefunction (\psi(t)), so evolve the pair ((\psi,\rho)) together.

Use RK4 for both (\psi) and (\rho).

Important: at each RK4 stage, recompute Bell currents and Bell rates from the stage (\psi).

Implement:

```go
type DerivativeStats struct {
	RateStats RateStats
}

func Derivative(H Matrix, psi []complex128, rho []float64) (
	dpsi []complex128,
	drho []float64,
	stats DerivativeStats,
)

func RK4Step(H Matrix, psi []complex128, rho []float64, dt float64) (
	nextPsi []complex128,
	nextRho []float64,
	stepStats StepStats,
)
```

Derivative algorithm:

```text
1. Compute dψ = -i H ψ.
2. Compute Bell current J from ψ.
3. Compute rates sigma from J and ψ.
4. Compute dρ using the Bell master equation and sigma.
5. Return derivatives and rate stats.
```

RK4 algorithm:

```text
k1 = Derivative(ψ, ρ)
k2 = Derivative(ψ + dt*k1ψ/2, ρ + dt*k1ρ/2)
k3 = Derivative(ψ + dt*k2ψ/2, ρ + dt*k2ρ/2)
k4 = Derivative(ψ + dt*k3ψ,   ρ + dt*k3ρ)

ψ_next = ψ + dt*(k1ψ + 2k2ψ + 2k3ψ + k4ψ)/6
ρ_next = ρ + dt*(k1ρ + 2k2ρ + 2k3ρ + k4ρ)/6
```

Do not silently hide major norm drift.

Do not renormalize (\psi) by default. Measure drift honestly.

For (\rho), allow tiny negative values within numerical tolerance, but report the maximum negative violation.

If serious negativity appears, mark failed or inconclusive.

Tests:

```text
Small system evolves without NaN/Inf.
Norm remains near 1 for small dt.
rho sum remains near 1.
Equivariance error remains small for a small stable config.
```

---

# 12. Equivariance Audit

At every time step:

```go
born := Probabilities(psi)
err := L1Distance(rho, born)
```

Track:

```go
max_equivariance_l1_error
final_equivariance_l1_error
mean_equivariance_l1_error
max_norm_error
max_rho_sum_error
max_rho_negative_violation
max_current_antisymmetry_error
max_rate_negative_violation
mean_total_bell_activity
max_total_bell_activity
nan_or_inf_detected
probability_floor_hits
```

Pass condition:

```text
toy_goal_passed only if:
  Hamiltonian Hermitian error <= hermitian_tolerance
  Max ψ norm error <= norm_tolerance
  Max rho sum error <= norm_tolerance
  Bell current antisymmetry within tolerance
  Rates nonnegative within tolerance
  Max equivariance L1 error <= equivariance_tolerance
  No serious rho negativity
  No NaN/Inf
  Forbidden-language audit passes
```

Fail condition:

```text
toy_goal_failed if:
  Checks complete but one or more pass/fail criteria are violated.
```

Inconclusive condition:

```text
toy_goal_inconclusive if:
  Numerical instability prevents judgment,
  NaN/Inf appears,
  probability-floor events are significant,
  or rho negativity is severe enough that the numerical method cannot be trusted.
```

---

# 13. Report Schema

Write these files to `--out`:

```text
input.json
report.json
report.md
```

Implement:

```go
type Report struct {
	SchemaVersion          string                 `json:"schema_version"`
	ToyID                  string                 `json:"toy_id"`
	ToyAnalysisOnly         bool                   `json:"toy_analysis_only"`
	PhysicsClaim           string                 `json:"physics_claim"`
	Model                  string                 `json:"model"`
	Sites                  int                    `json:"sites"`
	HilbertDim             int                    `json:"hilbert_dim"`
	Boundary               string                 `json:"boundary"`
	Goal                   string                 `json:"goal"`
	GoalStatus             string                 `json:"goal_status"`
	Checks                 Checks                 `json:"checks"`
	Metrics                Metrics                `json:"metrics"`
	DebtStatus             map[string]string      `json:"debt_status"`
	Limitations            []string               `json:"limitations"`
	ForbiddenLanguageAudit ForbiddenLanguageAudit `json:"forbidden_language_audit"`
}

type Checks struct {
	HamiltonianHermitian              bool `json:"hamiltonian_hermitian"`
	StateNormPreserved                bool `json:"state_norm_preserved"`
	RhoSumPreserved                   bool `json:"rho_sum_preserved"`
	CurrentAntisymmetric              bool `json:"current_antisymmetric"`
	RatesNonnegative                  bool `json:"rates_nonnegative"`
	EquivarianceErrorWithinTolerance bool `json:"equivariance_error_within_tolerance"`
	NoNaNOrInf                        bool `json:"no_nan_or_inf"`
	ForbiddenLanguagePassed          bool `json:"forbidden_language_passed"`
}

type Metrics struct {
	MaxHermitianError           float64 `json:"max_hermitian_error"`
	MaxNormError                float64 `json:"max_norm_error"`
	MaxRhoSumError              float64 `json:"max_rho_sum_error"`
	MaxRhoNegativeViolation     float64 `json:"max_rho_negative_violation"`
	MaxCurrentAntisymmetryError float64 `json:"max_current_antisymmetry_error"`
	MaxRateNegativeViolation    float64 `json:"max_rate_negative_violation"`
	MaxEquivarianceL1Error      float64 `json:"max_equivariance_l1_error"`
	FinalEquivarianceL1Error    float64 `json:"final_equivariance_l1_error"`
	MeanEquivarianceL1Error     float64 `json:"mean_equivariance_l1_error"`
	MeanTotalBellActivity       float64 `json:"mean_total_bell_activity"`
	MaxTotalBellActivity        float64 `json:"max_total_bell_activity"`
	ProbabilityFloorHits        int     `json:"probability_floor_hits"`
}
```

Required constant report values:

```json
{
  "schema_version": "bell_mipt_report_v0",
  "toy_id": "BELL-MIPT-0001",
  "toy_analysis_only": true,
  "physics_claim": "none",
  "goal": "compute_bell_rates_and_verify_equivariance"
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

---

# 14. Markdown Report

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
| max_rho_sum_error | ... |
| max_current_antisymmetry_error | ... |
| max_equivariance_l1_error | ... |
| mean_total_bell_activity | ... |
| max_total_bell_activity | ... |

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

The terms “MIPT” and “holography” are allowed when used in limitations/non-goals. Do not ban them globally.

---

# 15. Forbidden Language Audit

Add a simple forbidden phrase scanner for JSON and Markdown report strings.

Forbidden phrases include:

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
Bell jumps equal measurements
explains black holes
explains holography
```

Allowed scope phrases include:

```text
toy_goal_passed
toy analysis
numerical check
equivariance audit
rate algebra
No MIPT claim
No holography claim
No physics promotion
```

Implement:

```go
type ForbiddenLanguageAudit struct {
	Passed bool     `json:"passed"`
	Hits   []string `json:"hits"`
}

func AuditForbiddenLanguage(text string) ForbiddenLanguageAudit
```

Tests:

```text
Reports contain no forbidden promotion language.
Scanner catches known bad phrases.
Scanner allows limitation phrases such as "No MIPT claim" and "No holography claim".
```

---

# 16. Main Execution Flow

Program flow:

```text
1. Parse --config and --out.
2. Load config or default config.
3. Validate config.
4. Create output directory.
5. Write normalized input.json to output directory.
6. Build basis.
7. Build Hamiltonian.
8. Audit Hermiticity.
9. Generate deterministic random normalized ψ0.
10. Set ρ0 = |ψ0|².
11. Loop for steps:
    a. Perform joint RK4 step for ψ and ρ.
    b. Recompute metrics after each step.
    c. Track max/mean diagnostics.
    d. Detect NaN/Inf.
12. Determine goal_status.
13. Build report object.
14. Render report.json.
15. Render report.md.
16. Audit forbidden language.
17. Rebuild report with final forbidden-language audit field if needed.
18. Write report.json and report.md.
19. Print concise console summary:
    - output path
    - goal_status
    - max_equivariance_l1_error
20. Exit code:
    - 0 for toy_goal_passed
    - 1 for toy_goal_failed
    - 2 for toy_goal_inconclusive
```

If the repository or existing test harness expects different exit-code behavior, document the choice in the implementation report.

---

# 17. Required Tests

Implement tests for the following.

## Config

```text
TestDefaultConfigValid
TestLoadConfig
TestRejectUnsupportedModel
TestRejectUnsupportedBoundary
TestRejectUnsupportedInitialState
TestRejectTooManySites
```

## Basis

```text
TestBasisEnumeration
TestOccupied
TestPopCount
TestCountBelow
```

## Fermion operators

```text
TestAnnihilationSigns
TestCreationSigns
TestApplyOpsRightToLeft
TestNilOnInvalidCreationOrAnnihilation
TestFermionAnticommutationSmallBasis
```

## Hamiltonian

```text
TestHamiltonianHermitianOpen
TestHamiltonianHermitianPeriodic
TestHamiltonianDimension
TestHamiltonianDiagonalWhenNoHoppingNoPairing
TestPeriodicBoundaryUsesGenericJWNoSpecialParity
```

## Bell current

```text
TestBellCurrentAntisymmetric
TestBellCurrentDiagonalZero
```

## Bell rates

```text
TestBellRatesNonnegative
TestBellRatesPositiveCurrentDirection
TestBellRatesProbabilityFloorSafe
```

## Evolution and audit

```text
TestEquivarianceAuditSmallSystem
TestNoNaNInfDuringEvolution
TestNormPreservedSmallSystem
TestRhoSumPreservedSmallSystem
TestRhoTracksBornDistribution
```

## Output

```text
TestDeterministicOutputFixedSeed
TestReportContainsRequiredDebtStatus
TestReportContainsRequiredLimitations
TestNoForbiddenPromotionLanguage
TestDefaultRunWritesInputReportJSONAndMarkdown
```

---

# 18. Validation Commands

Run:

```bash
go test ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
```

Optional stronger validation:

```bash
go test -race ./...
go run ./cmd/bellmipt --config bellmipt.json --out /tmp/bellmipt-custom
```

After the default run, verify:

```text
/tmp/bellmipt-default/input.json exists.
/tmp/bellmipt-default/report.json exists.
/tmp/bellmipt-default/report.md exists.
report.json has schema_version = bell_mipt_report_v0.
report.json has toy_id = BELL-MIPT-0001.
report.json has toy_analysis_only = true.
report.json has physics_claim = none.
goal_status is one of:
  toy_goal_passed
  toy_goal_failed
  toy_goal_inconclusive.
default run should ideally produce toy_goal_passed.
```

---

# 19. Final Implementation Report Required from Coding Agent

After implementing, return a concise report with:

```text
1. Summary
2. Files added/modified
3. Key implementation decisions
4. Validation commands run
5. Default run goal_status
6. Important metrics from default report:
   - max_hermitian_error
   - max_norm_error
   - max_rho_sum_error
   - max_current_antisymmetry_error
   - max_equivariance_l1_error
   - mean_total_bell_activity
   - max_total_bell_activity
7. Any failing tests or known limitations
8. EBP status:
   needMap = unpaid
   needInvariant = partially_paid_equivariance_only
   needToyCheck = partially_paid_rate_algebra_only
   needNullModel = unpaid
   needObstruction = bell_jumps_are_not_measurements
   needFaithfulnessReview = unpaid
9. Explicit statement:
   This implementation does not claim MIPT, holography, conditional-wave-function bridge, or physics promotion.
```

---

# 20. Hard Guardrails

Do not claim:

```text
Bell-MIPT bridge proved
Bell-MIPT bridge established
MIPT observed
holography explained
Bell jumps are measurements
Bell jumps equal measurements
black-hole information explained
Bohmian mechanics validated
physics promotion achieved
```

Allowed conclusion if default run passes:

```text
BELL-MIPT-0001 passed its limited toy goal: the finite Kitaev-chain Bell-rate algebra and numerical equivariance audit behaved within configured tolerances.
```

Allowed conclusion if default run fails:

```text
BELL-MIPT-0001 failed its limited toy goal under the tested configuration. The failure is numerical or implementation-level unless further review shows otherwise.
```

Allowed conclusion if inconclusive:

```text
BELL-MIPT-0001 was inconclusive due to numerical stability or audit limitations.
```

---

# 21. Priority Order

Implement in this order:

```text
1. Config/defaults/validation
2. Basis
3. Fermion operators + anticommutation tests
4. Dense matrix type
5. Hamiltonian construction + Hermiticity tests
6. Initial state generation
7. Bell current + antisymmetry tests
8. Bell rates + nonnegativity tests
9. Derivative and RK4 evolution
10. Equivariance audit
11. Report JSON/Markdown
12. Forbidden-language audit
13. cmd/bellmipt main
14. Integration/default-run test
15. Final validation commands
```

Do not proceed to MIPT, random trajectories, conditional wave functions, or agents in this ticket.

My recommendation is to hand that prompt to the coding agent exactly as written. It supersedes conflicting details in the uploaded plans, especially the periodic-boundary sign issue and the choice to use joint RK4 with rate recomputation at each stage.

