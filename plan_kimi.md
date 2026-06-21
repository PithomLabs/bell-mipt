I'll read the uploaded file and then create an implementation plan for this Go project
Here is the implementation plan for **BELL-MIPT-0001** in Go.

---

## BELL-MIPT-0001: Go Implementation Plan

### 1. Project Structure

```
bellmipt/
├── go.mod
├── cmd/
│   └── bellmipt/
│       └── main.go          # CLI entry point
├── internal/
│   ├── config/
│   │   └── config.go        # JSON input schema + defaults
│   ├── basis/
│   │   └── basis.go         # uint64 bitstring basis enumeration
│   ├── fermion/
│   │   └── fermion.go       # c, c† with Jordan-Wigner signs
│   ├── hamiltonian/
│   │   └── hamiltonian.go   # Kitaev chain H matrix
│   ├── bell/
│   │   └── bell.go          # J_nm, rates, master equation
│   ├── evolve/
│   │   └── evolve.go        # RK4 for ψ, ρ evolution
│   ├── audit/
│   │   └── audit.go         # equivariance checks
│   └── report/
│       └── report.go        # JSON + Markdown output
└── internal/
    └── ..._test.go          # tests alongside packages
```

---

### 2. Package-by-Package Plan

#### 2.1 `config` — Input & Defaults

```go
// config.go
package config

type Config struct {
    SchemaVersion string      `json:"schema_version"`
    Model         string      `json:"model"`        // "finite_kitaev_chain"
    Sites         int         `json:"sites"`
    Boundary      string      `json:"boundary"`     // "open" | "periodic"
    Parameters    Parameters  `json:"parameters"`
    InitialState  InitialState `json:"initial_state"`
    Time          TimeParams  `json:"time"`
    Audit         AuditParams `json:"audit"`
}

type Parameters struct {
    Mu    float64 `json:"mu"`
    T     float64 `json:"t"`      // hopping
    Delta float64 `json:"delta"`  // pairing
}

type InitialState struct {
    Type string `json:"type"`    // "random_normalized"
    Seed int64  `json:"seed"`
}

type TimeParams struct {
    Dt     float64 `json:"dt"`
    Steps  int     `json:"steps"`
}

type AuditParams struct {
    HermitianTolerance     float64 `json:"hermitian_tolerance"`
    NormTolerance          float64 `json:"norm_tolerance"`
    EquivarianceTolerance  float64 `json:"equivariance_tolerance"`
}

func DefaultConfig() Config { ... }
func Load(path string) (Config, error) { ... }
func Validate(c Config) error { ... }
```

**Tests:**
- `DefaultConfig` returns valid struct
- `Load` parses sample JSON correctly
- `Validate` rejects unknown model/boundary/initial_state

---

#### 2.2 `basis` — Bitstring Enumeration

```go
// basis.go
package basis

// Dim returns 2^sites
func Dim(sites int) int { return 1 << sites }

// State is a uint64 bitstring; bit i = occupation at site i
type State uint64

// Occupied checks if site i is occupied
func (s State) Occupied(i int) bool { return (s>>i)&1 == 1 }

// AllStates returns slice of all 2^sites configurations
func AllStates(sites int) []State { ... }

// CountParticles returns number of 1-bits
func (s State) CountParticles() int { ... }
```

**Tests:**
- `Dim(6) == 64`
- `AllStates(3)` returns 8 states in order 0..7
- `CountParticles` matches known values

---

#### 2.3 `fermion` — Jordan-Wigner Operators

```go
// fermion.go
package fermion

import "bellmipt/internal/basis"

// Apply c_i (annihilation) to |s⟩.
// Returns (newState, amplitude, ok).
// ok=false if site i is empty.
func Annihilate(s basis.State, i int) (basis.State, complex128, bool) {
    if !s.Occupied(i) {
        return 0, 0, false
    }
    // Jordan-Wigner sign: (-1)^{number of occupied sites j < i}
    sign := jwSign(s, i)
    newState := s &^ (1 << i) // clear bit i
    return newState, complex(float64(sign), 0), true
}

// Apply c†_i (creation)
func Create(s basis.State, i int) (basis.State, complex128, bool) {
    if s.Occupied(i) {
        return 0, 0, false
    }
    sign := jwSign(s, i)
    newState := s | (1 << i)
    return newState, complex(float64(sign), 0), true
}

// jwSign computes (-1)^{#occupied sites < i}
func jwSign(s basis.State, i int) int { ... }
```

**Tests:**
- `Annihilate` on empty site → `ok=false`
- `Create` on occupied site → `ok=false`
- `Create` then `Annihilate` on same site returns original state with amplitude 1
- JW sign alternates correctly for adjacent sites (anticommutation)
- `c_i c_j + c_j c_i = 0` on test states (numerically)

---

#### 2.4 `hamiltonian` — Kitaev Chain Matrix

```go
// hamiltonian.go
package hamiltonian

import (
    "bellmipt/internal/basis"
    "bellmipt/internal/config"
    "bellmipt/internal/fermion"
)

// Matrix is H in the occupation-number basis
type Matrix struct {
    N     int              // Hilbert dim = 2^sites
    Sites int
    H     [][]complex128   // dense for toy; N ≤ ~1024 (sites ≤ 10)
}

// Build constructs H from config
func Build(cfg config.Config) Matrix { ... }

// HermitianError returns max |H_ij - conj(H_ji)|
func (m Matrix) HermitianError() float64 { ... }

// internals:
//   -mu * sum_i c†_i c_i          (chemical potential)
//   -t * sum_i (c†_i c_{i+1} + h.c.)   (hopping)
//   +delta * sum_i (c_i c_{i+1} + h.c.) (pairing)
```

**Implementation notes:**
- Map each `basis.State` to integer index `idx = int(state)`
- For each term, iterate all basis states, apply fermion operators, accumulate into `H`
- Periodic: site `sites` wraps to `0`. Open: last bond omitted.

**Tests:**
- `HermitianError` < tolerance for random parameters
- Trace matches expected `-mu * <N>` for known states
- Known case: `sites=2, mu=0, t=0, delta=0` → `H=0`

---

#### 2.5 `bell` — Currents, Rates, Master Equation

```go
// bell.go
package bell

import (
    "bellmipt/internal/hamiltonian"
)

// Current J_nm = 2 * Im(conj(psi_n) * H_nm * psi_m)
func Current(psi []complex128, H hamiltonian.Matrix, n, m int) float64 {
    return 2 * imag(conj(psi[n]) * H.H[n][m] * psi[m])
}

// Rate n <- m = max(J_nm, 0) / |psi_m|^2
// Returns 0 if J_nm ≤ 0 or |psi_m|^2 == 0
func Rate(psi []complex128, H hamiltonian.Matrix, n, m int) float64 { ... }

// RateMatrix returns full W[n][m] = rate n <- m
func RateMatrix(psi []complex128, H hamiltonian.Matrix) [][]float64 { ... }

// MasterEquationRHS computes dρ/dt = sum_m [W_nm ρ_m - W_mn ρ_n]
func MasterEquationRHS(rho []float64, W [][]float64) []float64 { ... }
```

**Tests:**
- `Current(n,m) == -Current(m,n)` (antisymmetry)
- All rates ≥ 0
- `Rate` = 0 when `J_nm < 0`
- Master equation preserves sum(rho) ≈ 1

---

#### 2.6 `evolve` — RK4 Integration

```go
// evolve.go
package evolve

// StepPsi advances ψ by dt using RK4 for dψ/dt = -i H ψ
func StepPsi(psi []complex128, H [][]complex128, dt float64) []complex128 { ... }

// StepRho advances ρ by dt using RK4 for master equation
func StepRho(rho []float64, W [][]float64, dt float64) []float64 { ... }

// Norm returns sum |psi_i|^2
func Norm(psi []complex128) float64 { ... }

// helpers: mat-vec multiply for ψ, vector ops for ρ
```

**Tests:**
- Free particle (`H=0`): ψ unchanged
- Norm preserved within tolerance over many steps
- For small dt, RK4 matches analytic result for 2-level system

---

#### 2.7 `audit` — Equivariance Check

```go
// audit.go
package audit

type Result struct {
    MaxHermitianError        float64
    MaxNormError             float64
    MaxNegativeRateViolation float64
    MaxEquivarianceL1Error   float64
    MeanTotalBellActivity    float64
    MaxTotalBellActivity     float64
    Checks                   Checks
}

type Checks struct {
    HamiltonianHermitian           bool
    StateNormPreserved             bool
    RatesNonnegative               bool
    EquivarianceErrorWithinTolerance bool
}

// Run performs full evolution and comparison
func Run(cfg config.Config, H hamiltonian.Matrix, psi0 []complex128) Result { ... }
```

**Algorithm:**
1. Initialize `psi = psi0`, `rho = |psi0|^2`
2. For each time step:
   a. Compute `W` from current `psi`
   b. `psi = StepPsi(psi, H, dt)`
   c. `rho = StepRho(rho, W, dt)`
   d. Record `L1 = sum |rho_i - |psi_i|^2|`
   e. Record total Bell activity `sum_{n≠m} W_nm rho_m`
3. Return aggregated metrics

**Tests:**
- On `sites=2`, equivariance error near machine precision for small dt
- Deterministic for fixed seed

---

#### 2.8 `report` — Output Generation

```go
// report.go
package report

type Report struct {
    SchemaVersion      string            `json:"schema_version"`
    ToyID              string            `json:"toy_id"`
    ToyAnalysisOnly    bool              `json:"toy_analysis_only"`
    PhysicsClaim       string            `json:"physics_claim"`
    Model              string            `json:"model"`
    Sites              int               `json:"sites"`
    HilbertDim         int               `json:"hilbert_dim"`
    Goal               string            `json:"goal"`
    GoalStatus         string            `json:"goal_status"` // passed/failed/inconclusive
    Checks             audit.Checks      `json:"checks"`
    Metrics            audit.Result      `json:"metrics"`
    DebtStatus         DebtStatus        `json:"debt_status"`
}

type DebtStatus struct {
    NeedMap               string `json:"needMap"`
    NeedInvariant         string `json:"needInvariant"`
    NeedToyCheck          string `json:"needToyCheck"`
    NeedNullModel         string `json:"needNullModel"`
    NeedObstruction       string `json:"needObstruction"`
    NeedFaithfulnessReview string `json:"needFaithfulnessReview"`
}

func WriteJSON(r Report, path string) error { ... }
func WriteMD(r Report, path string) error { ... }
func WriteInput(cfg config.Config, path string) error { ... }
```

**Tests:**
- Output contains no forbidden words ("MIPT claim", "holography", etc.)
- JSON round-trips correctly
- Markdown contains all required fields

---

### 3. `cmd/bellmipt/main.go` — CLI

```go
package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"

    "bellmipt/internal/audit"
    "bellmipt/internal/basis"
    "bellmipt/internal/config"
    "bellmipt/internal/hamiltonian"
    "bellmipt/internal/report"
)

func main() {
    var configPath, outDir string
    flag.StringVar(&configPath, "config", "", "path to JSON config (optional, uses default)")
    flag.StringVar(&outDir, "out", "out/bellmipt-run", "output directory")
    flag.Parse()

    // 1. Load config
    cfg := config.DefaultConfig()
    if configPath != "" {
        var err error
        cfg, err = config.Load(configPath)
        if err != nil { ... }
    }
    if err := config.Validate(cfg); err != nil { ... }

    // 2. Build basis & Hamiltonian
    dim := basis.Dim(cfg.Sites)
    H := hamiltonian.Build(cfg)

    // 3. Check Hermiticity
    hermErr := H.HermitianError()

    // 4. Initialize state
    psi0 := initializeState(cfg, dim)

    // 5. Run audit / evolution
    result := audit.Run(cfg, H, psi0)

    // 6. Determine goal status
    goalStatus := determineGoalStatus(cfg, result)

    // 7. Build report
    rep := report.Report{
        SchemaVersion:   "bell_mipt_report_v0",
        ToyID:           "BELL-MIPT-0001",
        ToyAnalysisOnly: true,
        PhysicsClaim:    "none",
        Model:           cfg.Model,
        Sites:           cfg.Sites,
        HilbertDim:      dim,
        Goal:            "compute_bell_rates_and_verify_equivariance",
        GoalStatus:      goalStatus,
        Checks:          result.Checks,
        Metrics:         result,
        DebtStatus: report.DebtStatus{
            NeedMap:               "unpaid",
            NeedInvariant:         "partially_paid_equivariance_only",
            NeedToyCheck:          "partially_paid_rate_algebra_only",
            NeedNullModel:         "unpaid",
            NeedObstruction:         "bell_jumps_are_not_measurements",
            NeedFaithfulnessReview:  "unpaid",
        },
    }

    // 8. Write outputs
    os.MkdirAll(outDir, 0755)
    report.WriteInput(cfg, filepath.Join(outDir, "input.json"))
    report.WriteJSON(rep, filepath.Join(outDir, "report.json"))
    report.WriteMD(rep, filepath.Join(outDir, "report.md"))

    fmt.Printf("Done. Output written to %s\n", outDir)
    if goalStatus == "toy_goal_failed" {
        os.Exit(1)
    }
}
```

---

### 4. Test Matrix

| Test | Package | What it checks |
|---|---|---|
| `TestBasisEnumeration` | `basis` | All states generated, correct count |
| `TestFermionSigns` | `fermion` | JW signs correct, anticommutation |
| `TestHamiltonianHermiticity` | `hamiltonian` | H = H† for all parameter sets |
| `TestBellCurrentAntisymmetry` | `bell` | J_nm = -J_mn |
| `TestNonnegativeRates` | `bell` | All W_nm ≥ 0 |
| `TestEquivarianceSmallSystem` | `audit` | ρ tracks \|ψ\|² on sites=2,3 |
| `TestDeterministicOutput` | `audit` | Same seed → same result |
| `TestNoPromotionLanguage` | `report` | Output string scan for forbidden words |

---

### 5. Validation Commands

```bash
# Run all tests
go test ./...

# Run with default config
go run ./cmd/bellmipt --out /tmp/bellmipt-default

# Run with custom config
go run ./cmd/bellmipt --config bellmipt.json --out /tmp/bellmipt-custom

# Check outputs
cat /tmp/bellmipt-default/report.md
cat /tmp/bellmipt-default/report.json | jq .
```

---

### 6. Key Design Decisions

| Decision | Rationale |
|---|---|
| Dense matrices | `sites ≤ 10` → `dim ≤ 1024`; dense is simple and fast enough |
| No random trajectories | Master-equation check is cleaner for verifying rate algebra |
| No subcommands | Single command, single purpose — no CLI bloat |
| Built-in default config | `--config` is optional; always runnable |
| `uint64` bitstrings | Supports up to 64 sites; sufficient for toy |
| RK4 for both ψ and ρ | Consistent, well-understood, accurate enough |
| Exit code 1 on failure | Enables CI/script integration |

---

### 7. Forbidden Scope (Explicitly Excluded)

- MIPT mapping
- Holography / black holes
- Conditional wave functions
- Lean proofs
- Agent orchestration
- Random trajectory sampling
- Subcommands / CLI frameworks
- Any language implying Bell jumps *are* measurements

---

This plan gives you a **minimal, testable, honest** implementation of BELL-MIPT-0001. Each package has a single responsibility, the evolution is numerically sound, and the output is strictly scoped to toy analysis with no unwarranted claims.
