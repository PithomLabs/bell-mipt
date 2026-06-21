# BELL-MIPT-0001 Implementation Plan

Compute Bell jump rates for a finite Kitaev-chain fermion lattice and numerically verify equivariance — i.e., that the Bell master-equation distribution ρ(t) tracks |ψ(t)|² over time.

> [!IMPORTANT]
> **Toy analysis only.** No MIPT claim, no holography claim, no Bell-jumps-equal-measurements claim. Output states `physics_claim: "none"`, `toy_analysis_only: true`.

## Decisions from Clarification

| Question | Decision |
|---|---|
| Go module path | `github.com/PithomLabs/bell-mipt` |
| External deps | Zero — stdlib only (`complex128` dense LA on 64×64 matrices) |
| Evolution coupling | Joint RK4 with rate recomputation at every sub-stage (only v0 strategy) |
| Integration testing | `bellmipt.Run()` as library call + unit test for `ExitCodeForGoalStatus` |
| Pairing sign | `+Δ` exactly as prompt specifies |
| PBC sign | No extra boundary parity — JW string handles it in full Fock space |
| Max sites | 10 (v0 cap) |

---

## Proposed Changes

### File Layout

```
bft-mipt/
├── go.mod                           # github.com/PithomLabs/bell-mipt
├── bellmipt.json                    # Example config
├── cmd/bellmipt/
│   └── main.go                      # Thin CLI: flags → Run() → exit code
└── internal/bellmipt/
    ├── config.go                    # Config struct, defaults, validation
    ├── basis.go                     # Fock basis (uint64 bitstrings)
    ├── fermion.go                   # Create/Annihilate with JW signs, ApplyOps
    ├── hamiltonian.go               # Dense Kitaev chain H matrix
    ├── current.go                   # Bell probability current J_nm
    ├── rates.go                     # Bell jump rates σ(n←m)
    ├── evolve.go                    # Joint RK4 for (ψ, ρ), Derivative
    ├── audit.go                     # Per-step + final audit checks
    ├── report.go                    # Report struct, JSON generation
    ├── markdown.go                  # Markdown report rendering
    ├── forbidden.go                 # Forbidden promotion language scanner
    ├── run.go                       # Run() orchestrator
    ├── testdata/
    │   └── small_default.json       # Test config
    ├── config_test.go
    ├── basis_test.go
    ├── fermion_test.go
    ├── hamiltonian_test.go
    ├── current_test.go
    ├── rates_test.go
    ├── evolve_test.go
    ├── audit_test.go
    └── report_test.go
```

---

### 1. Config (`config.go`)

#### [NEW] [config.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/config.go)

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
    HermitianTolerance    float64 `json:"hermitian_tolerance"`
    NormTolerance         float64 `json:"norm_tolerance"`
    EquivarianceTolerance float64 `json:"equivariance_tolerance"`
}
```

Functions:
- `DefaultConfig() Config` — 6-site periodic, μ=1.0, t=1.0, Δ=0.5, seed 12345, dt=0.001, 1000 steps
- `LoadConfig(path string) (Config, error)` — reads JSON
- `(c Config) Validate() error` — rejects unsupported values

Validation rules:
- `schema_version` must be `"bell_mipt_toy_v0"`
- `model` must be `"finite_kitaev_chain"`
- `boundary` must be `"open"` or `"periodic"`
- `initial_state.type` must be `"random_normalized"`
- `sites` must be 2–10
- `dt` > 0, `steps` > 0, all tolerances > 0

#### [NEW] [config_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/config_test.go)

Tests: `TestDefaultConfigValid`, `TestLoadConfig`, `TestRejectUnsupportedModel`, `TestRejectUnsupportedBoundary`, `TestRejectUnsupportedInitialState`, `TestRejectTooManySites`

---

### 2. Basis (`basis.go`)

#### [NEW] [basis.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/basis.go)

Each Fock state is a `uint64` bitstring. Bit `j` set = site `j` occupied. Basis state index = bitstring value.

```go
type Basis struct {
    Sites int
    Dim   int  // 1 << Sites
}

func NewBasis(sites int) (Basis, error)
func Occupied(state uint64, site int) bool
func CountBelow(state uint64, site int) int  // popcount of bits below site j
func PopCount(state uint64) int
func EnumerateStates(b Basis) []uint64       // [0, 1, ..., 2^N - 1]
```

`CountBelow` uses `math/bits.OnesCount64(state & ((1 << site) - 1))`.

#### [NEW] [basis_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/basis_test.go)

Tests: `TestBasisEnumeration` (sites=3 → 8 states), `TestOccupied`, `TestPopCount`, `TestCountBelow`

---

### 3. Fermion Operators (`fermion.go`)

#### [NEW] [fermion.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/fermion.go)

Jordan-Wigner convention:

$$c_j |n\rangle = \begin{cases} 0 & \text{if site } j \text{ unoccupied} \\ (-1)^{\text{CountBelow}(n, j)} \, |n \oplus 2^j\rangle & \text{otherwise} \end{cases}$$

$$c_j^\dagger |n\rangle = \begin{cases} 0 & \text{if site } j \text{ occupied} \\ (-1)^{\text{CountBelow}(n, j)} \, |n \oplus 2^j\rangle & \text{otherwise} \end{cases}$$

```go
type OpKind int
const (
    Create     OpKind = iota
    Annihilate
)

type FermionOp struct {
    Kind OpKind
    Site int
}

type OpResult struct {
    OK    bool
    State uint64
    Sign  int  // +1 or -1
}

func ApplyAnnihilate(state uint64, site int) OpResult
func ApplyCreate(state uint64, site int) OpResult
func ApplyOps(state uint64, ops []FermionOp) OpResult
```

> [!IMPORTANT]
> `ApplyOps` takes operators in **physics notation order** (left-to-right) but executes them **right-to-left**. For `c†_i c_j`, pass `[]FermionOp{{Create, i}, {Annihilate, j}}` — it applies `c_j` first, then `c†_i`. Signs compose multiplicatively.

> [!NOTE]
> **Periodic boundary**: No special parity sign. The bond (N-1, 0) uses the same generic `ApplyOps`. For `c†_0 c_{N-1}`: first `Annihilate(state, N-1)` counts bits 0..N-2, then `Create(intermediate, 0)` counts no bits below 0. Correct by construction.

#### [NEW] [fermion_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/fermion_test.go)

Tests: `TestAnnihilationSigns`, `TestCreationSigns`, `TestApplyOpsRightToLeft`, `TestNilOnInvalidCreationOrAnnihilation`, `TestFermionAnticommutationSmallBasis` (verify {c_i, c†_j} = δ_ij on all basis states for sites=2 and sites=3)

---

### 4. Dense Matrix + Hamiltonian (`hamiltonian.go`)

#### [NEW] [hamiltonian.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/hamiltonian.go)

Dense flat row-major matrix:

```go
type Matrix struct {
    Dim  int
    Data []complex128  // length Dim*Dim, row-major
}

func NewMatrix(dim int) Matrix
func (m Matrix) At(row, col int) complex128
func (m Matrix) Add(row, col int, v complex128)
func (m Matrix) Set(row, col int, v complex128)
```

Convention: `H[n][m] = ⟨n|H|m⟩`, so row = destination, col = source. Index: `row*Dim + col`.

Kitaev chain Hamiltonian:

$$H = -\mu \sum_i n_i - t \sum_{\langle i,j \rangle} (c_i^\dagger c_j + c_j^\dagger c_i) + \Delta \sum_{\langle i,j \rangle} (c_i^\dagger c_j^\dagger + c_j c_i)$$

```go
func BuildKitaevHamiltonian(cfg Config, basis Basis) (Matrix, error)
func HermitianError(H Matrix) float64  // max |H[n,m] - conj(H[m,n])|
```

**Construction algorithm** (per basis ket |m⟩):

1. **Chemical potential**: for each site i, if occupied, add `-μ` to `H[m][m]`
2. **For each bond (i, j)**:
   - Hopping: apply `ApplyOps(m, [{Create, i}, {Annihilate, j}])` → if OK, add `-t * sign` to `H[n][m]`. Same for `[{Create, j}, {Annihilate, i}]`.
   - Pairing: apply `ApplyOps(m, [{Create, i}, {Create, j}])` → if OK, add `+Δ * sign` to `H[n][m]`. Same for `[{Annihilate, j}, {Annihilate, i}]`.

**Bonds**: open = (0,1)...(N-2,N-1); periodic = open + (N-1,0).

> [!WARNING]
> Do **not** symmetrize H after construction to hide sign errors. Build all conjugate terms explicitly, then verify Hermiticity as a real audit.

#### [NEW] [hamiltonian_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/hamiltonian_test.go)

Tests: `TestHamiltonianHermitianOpen`, `TestHamiltonianHermitianPeriodic`, `TestHamiltonianDimension`, `TestHamiltonianDiagonalWhenNoHoppingNoPairing`, `TestPeriodicBoundaryUsesGenericJWNoSpecialParity`

For sites=2 with simple parameters, verify matrix entries match hand calculation.

---

### 5. Initial State (in `run.go`)

```go
func RandomNormalizedState(dim int, seed int64) []complex128
func Probabilities(psi []complex128) []float64  // |ψ_i|²
func Norm(psi []complex128) float64             // √(Σ|ψ_i|²)
```

Generate each component as `complex(rng.NormFloat64(), rng.NormFloat64())` with deterministic `math/rand.New(math/rand.NewSource(seed))`, then normalize to unit norm.

Initialize ρ as `rho[i] = |psi[i]|²`.

---

### 6. Bell Probability Current (`current.go`)

#### [NEW] [current.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/current.go)

$$J_{nm} = 2 \, \text{Im}(\psi_n^* \, H_{nm} \, \psi_m)$$

```go
func BellCurrent(H Matrix, psi []complex128) [][]float64
```

Returns `dim × dim` matrix. Properties: `J[n][m] = -J[m][n]` (antisymmetric), `J[n][n] ≈ 0`.

#### [NEW] [current_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/current_test.go)

Tests: `TestBellCurrentAntisymmetric`, `TestBellCurrentDiagonalZero`

---

### 7. Bell Jump Rates (`rates.go`)

#### [NEW] [rates.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/rates.go)

$$\sigma(n \leftarrow m) = \frac{\max(J_{nm},\, 0)}{|\psi_m|^2}$$

```go
const ProbabilityFloor = 1e-14

type RateStats struct {
    MaxNegativeRateViolation float64 `json:"max_negative_rate_violation"`
    MeanTotalActivity        float64 `json:"mean_total_bell_activity"`
    MaxTotalActivity         float64 `json:"max_total_bell_activity"`
    ProbabilityFloorHits     int     `json:"probability_floor_hits"`
}

func BellRates(J [][]float64, psi []complex128) (rates [][]float64, stats RateStats)
```

If `|ψ_m|² < ProbabilityFloor`, set all outgoing rates from m to zero and record a floor hit. Significant floor hits → `toy_goal_inconclusive`.

#### [NEW] [rates_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/rates_test.go)

Tests: `TestBellRatesNonnegative`, `TestBellRatesPositiveCurrentDirection`, `TestBellRatesProbabilityFloorSafe`

---

### 8. Evolution (`evolve.go`)

#### [NEW] [evolve.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/evolve.go)

Schrödinger: $d\psi/dt = -iH\psi$

Bell master equation: $d\rho_n/dt = \sum_m [\sigma(n \leftarrow m) \rho_m - \sigma(m \leftarrow n) \rho_n]$

**Joint RK4** evolves (ψ, ρ) together. At each RK4 sub-stage, rates are recomputed from the stage ψ:

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

**Derivative algorithm**:
1. `dpsi = -i H ψ` (dense matrix-vector multiply)
2. `J = BellCurrent(H, ψ)`
3. `rates, rateStats = BellRates(J, ψ)`
4. `drho[n] = Σ_m [rates[n][m] * rho[m] - rates[m][n] * rho[n]]`

**RK4 algorithm**:
```
k1 = Derivative(H, ψ, ρ)
k2 = Derivative(H, ψ + dt*k1_ψ/2, ρ + dt*k1_ρ/2)
k3 = Derivative(H, ψ + dt*k2_ψ/2, ρ + dt*k2_ρ/2)
k4 = Derivative(H, ψ + dt*k3_ψ, ρ + dt*k3_ρ)
ψ_next = ψ + dt*(k1_ψ + 2*k2_ψ + 2*k3_ψ + k4_ψ)/6
ρ_next = ρ + dt*(k1_ρ + 2*k2_ρ + 2*k3_ρ + k4_ρ)/6
```

> [!NOTE]
> Do **not** renormalize ψ. Measure norm drift honestly. For ρ, allow tiny negatives within tolerance but report `max_rho_negative_violation`.

#### [NEW] [evolve_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/evolve_test.go)

Tests: `TestEquivarianceAuditSmallSystem`, `TestNoNaNInfDuringEvolution`, `TestNormPreservedSmallSystem`, `TestRhoSumPreservedSmallSystem`, `TestRhoTracksBornDistribution`

---

### 9. Audit (`audit.go`)

#### [NEW] [audit.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/audit.go)

Per-step tracking and final judgment:

```go
type StepAudit struct {
    NormError              float64
    RhoSumError            float64
    EquivarianceL1Error    float64
    RhoNegativeViolation   float64
    CurrentAntisymError    float64
    NaNOrInfDetected       bool
    TotalBellActivity      float64
}
```

Accumulated metrics:

```go
type AuditAccumulator struct {
    MaxHermitianError           float64
    MaxNormError                float64
    MaxRhoSumError              float64
    MaxRhoNegativeViolation     float64
    MaxCurrentAntisymmetryError float64
    MaxRateNegativeViolation    float64
    MaxEquivarianceL1Error      float64
    FinalEquivarianceL1Error    float64
    MeanEquivarianceL1Error     float64
    MeanTotalBellActivity       float64
    MaxTotalBellActivity        float64
    NaNOrInfDetected            bool
    ProbabilityFloorHits        int
}
```

Functions:
- `L1Distance(a, b []float64) float64`
- `CheckNaNInf(psi []complex128, rho []float64) bool`
- `(a *AuditAccumulator) RecordStep(step StepAudit)`
- `(a *AuditAccumulator) DetermineGoalStatus(cfg AuditConfig) string`

**Goal status logic**:
- `toy_goal_inconclusive` if NaN/Inf detected or significant probability-floor hits
- `toy_goal_passed` if ALL: Hermitian error ≤ tol, norm error ≤ tol, rho sum error ≤ tol, current antisymmetric, rates nonneg, equivariance L1 ≤ tol, no serious rho negativity, forbidden language passes
- `toy_goal_failed` otherwise

#### [NEW] [audit_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/audit_test.go)

Tests: verify pass/fail/inconclusive logic on synthetic audit data.

---

### 10. Report (`report.go`, `markdown.go`)

#### [NEW] [report.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/report.go)

```go
type Report struct {
    SchemaVersion          string                 `json:"schema_version"`  // "bell_mipt_report_v0"
    ToyID                  string                 `json:"toy_id"`          // "BELL-MIPT-0001"
    ToyAnalysisOnly        bool                   `json:"toy_analysis_only"` // true
    PhysicsClaim           string                 `json:"physics_claim"`   // "none"
    Model                  string                 `json:"model"`
    Sites                  int                    `json:"sites"`
    HilbertDim             int                    `json:"hilbert_dim"`
    Boundary               string                 `json:"boundary"`
    Goal                   string                 `json:"goal"` // "compute_bell_rates_and_verify_equivariance"
    GoalStatus             string                 `json:"goal_status"`
    Checks                 Checks                 `json:"checks"`
    Metrics                Metrics                `json:"metrics"`
    DebtStatus             map[string]string      `json:"debt_status"`
    Limitations            []string               `json:"limitations"`
    ForbiddenLanguageAudit ForbiddenLanguageAudit `json:"forbidden_language_audit"`
}

type Checks struct {
    HamiltonianHermitian             bool `json:"hamiltonian_hermitian"`
    StateNormPreserved               bool `json:"state_norm_preserved"`
    RhoSumPreserved                  bool `json:"rho_sum_preserved"`
    CurrentAntisymmetric             bool `json:"current_antisymmetric"`
    RatesNonnegative                 bool `json:"rates_nonnegative"`
    EquivarianceErrorWithinTolerance bool `json:"equivariance_error_within_tolerance"`
    NoNaNOrInf                       bool `json:"no_nan_or_inf"`
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

Fixed debt status:
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

Fixed limitations:
```
- This checks Bell-rate algebra in a finite toy model only.
- This does not implement MIPT.
- This does not show Bell jumps are measurements.
- This does not construct a conditional-wave-function bridge.
- This does not support any holography or black-hole claim.
- This is not a physics promotion.
```

#### [NEW] [markdown.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/markdown.go)

Renders `report.md` with sections: Status, Scope, What Was Checked, Metrics (table), EBP Debt Status (table), Limitations.

---

### 11. Forbidden Language (`forbidden.go`)

#### [NEW] [forbidden.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/forbidden.go)

```go
type ForbiddenLanguageAudit struct {
    Passed bool     `json:"passed"`
    Hits   []string `json:"hits"`
}

func AuditForbiddenLanguage(text string) ForbiddenLanguageAudit
```

Forbidden phrases (case-insensitive):
```
proves the Bell-MIPT bridge, proven bridge, establishes MIPT, confirms holography,
solves, breakthrough, validated theory, physics promotion, Bell jumps are measurements,
Bell jumps equal measurements, explains black holes, explains holography
```

Allowed context phrases (must not trigger):
```
toy_goal_passed, toy analysis, numerical check, equivariance audit, rate algebra,
No MIPT claim, No holography claim, No physics promotion
```

---

### 12. Run Orchestrator (`run.go`)

#### [NEW] [run.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/run.go)

```go
type RunResult struct {
    Report Report
    Error  error
}

func Run(cfg Config, outDir string) RunResult
func ExitCodeForGoalStatus(status string) int  // 0=passed, 1=failed, 2=inconclusive
```

**Execution flow**:
1. Validate config
2. Create output directory
3. Write `input.json`
4. Build basis (2^N states)
5. Build Hamiltonian matrix
6. Audit Hermiticity
7. Generate deterministic random normalized ψ₀
8. Set ρ₀ = |ψ₀|²
9. **Evolution loop** (steps iterations):
   - Joint RK4 step for (ψ, ρ) — rates recomputed at each sub-stage
   - Per-step audit: norm, rho sum, equivariance L1, NaN/Inf, current antisymmetry
   - Accumulate max/mean diagnostics
10. Determine `goal_status`
11. Build Report struct
12. Render `report.md`
13. Audit forbidden language on rendered markdown
14. Finalize report with forbidden-language result
15. Write `report.json` and `report.md`
16. Print console summary: output path, goal_status, max equivariance L1 error

---

### 13. CLI Entry Point (`cmd/bellmipt/main.go`)

#### [NEW] [main.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/cmd/bellmipt/main.go)

Thin wrapper:
```go
func main() {
    configPath := flag.String("config", "", "path to JSON config (default: built-in)")
    outDir := flag.String("out", "out/bellmipt-run", "output directory")
    flag.Parse()

    cfg := loadConfigOrDefault(*configPath)
    result := bellmipt.Run(cfg, *outDir)

    if result.Error != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", result.Error)
        os.Exit(1)
    }
    os.Exit(bellmipt.ExitCodeForGoalStatus(result.Report.GoalStatus))
}
```

---

### 14. Example Config

#### [NEW] [bellmipt.json](file:///home/chaschel/Documents/ibm/go/bft-mipt/bellmipt.json)

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

---

## Implementation Priority Order

```
 1. Config / defaults / validation           → config.go, config_test.go
 2. Basis                                    → basis.go, basis_test.go
 3. Fermion operators + anticommutation      → fermion.go, fermion_test.go
 4. Dense matrix type                        → hamiltonian.go (Matrix type)
 5. Hamiltonian construction + Hermiticity   → hamiltonian.go, hamiltonian_test.go
 6. Initial state generation                 → run.go (RandomNormalizedState, etc.)
 7. Bell current + antisymmetry              → current.go, current_test.go
 8. Bell rates + nonnegativity               → rates.go, rates_test.go
 9. Derivative and joint RK4 evolution       → evolve.go, evolve_test.go
10. Equivariance audit                       → audit.go, audit_test.go
11. Report JSON                              → report.go
12. Report Markdown                          → markdown.go
13. Forbidden-language audit                 → forbidden.go, report_test.go
14. Run orchestrator                         → run.go
15. cmd/bellmipt main                        → main.go
16. Integration / default-run test           → report_test.go (TestDefaultRunWritesInputReportJSONAndMarkdown)
17. Final validation commands
```

---

## Verification Plan

### Automated Tests

```bash
go test ./... -v -count=1
go test -race ./...
```

37 named tests across 9 test files (per prompt_plan.md §17):

| Area | Tests |
|---|---|
| Config | `TestDefaultConfigValid`, `TestLoadConfig`, `TestRejectUnsupportedModel`, `TestRejectUnsupportedBoundary`, `TestRejectUnsupportedInitialState`, `TestRejectTooManySites` |
| Basis | `TestBasisEnumeration`, `TestOccupied`, `TestPopCount`, `TestCountBelow` |
| Fermion | `TestAnnihilationSigns`, `TestCreationSigns`, `TestApplyOpsRightToLeft`, `TestNilOnInvalidCreationOrAnnihilation`, `TestFermionAnticommutationSmallBasis` |
| Hamiltonian | `TestHamiltonianHermitianOpen`, `TestHamiltonianHermitianPeriodic`, `TestHamiltonianDimension`, `TestHamiltonianDiagonalWhenNoHoppingNoPairing`, `TestPeriodicBoundaryUsesGenericJWNoSpecialParity` |
| Current | `TestBellCurrentAntisymmetric`, `TestBellCurrentDiagonalZero` |
| Rates | `TestBellRatesNonnegative`, `TestBellRatesPositiveCurrentDirection`, `TestBellRatesProbabilityFloorSafe` |
| Evolution | `TestEquivarianceAuditSmallSystem`, `TestNoNaNInfDuringEvolution`, `TestNormPreservedSmallSystem`, `TestRhoSumPreservedSmallSystem`, `TestRhoTracksBornDistribution` |
| Report | `TestDeterministicOutputFixedSeed`, `TestReportContainsRequiredDebtStatus`, `TestReportContainsRequiredLimitations`, `TestNoForbiddenPromotionLanguage`, `TestDefaultRunWritesInputReportJSONAndMarkdown` |

### Manual Validation

```bash
go run ./cmd/bellmipt --out out/bellmipt-default
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-custom
cat out/bellmipt-default/report.json | python3 -m json.tool
cat out/bellmipt-default/report.md
```

Verify:
- `goal_status` = `"toy_goal_passed"`
- All checks = `true`
- `physics_claim` = `"none"`, `toy_analysis_only` = `true`
- No forbidden language in report.md
- Exit code 0

