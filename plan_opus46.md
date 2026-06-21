Viewed prompt.md:1-247
Listed directory bft-mipt
Invoked subagent

Let me also look at the existing plans for additional context while the research completes.

Viewed bell-type-qft-mipt.md:1-100
Viewed bell-type-qft-mipt.md:100-300

I have a good understanding of the project. Now let me wait for the physics research subagent to complete and then write the implementation plan.

Used tool: schedule

I have enough understanding from the prompt and background document to begin drafting the implementation plan while waiting for the research agent. Let me write it now.

Edited implementation_plan.md

The implementation plan is ready for your review. It covers all 9 components needed to implement BELL-MIPT-0001 as a single Go command:

1. **Module & entry point** — `cmd/bellmipt/main.go` with `--config` and `--out` flags
2. **Config** — JSON parsing with built-in defaults
3. **Basis** — Fock state enumeration as uint64 bitstrings
4. **Fermion operators** — Jordan-Wigner signed creation/annihilation
5. **Hamiltonian** — Dense Kitaev chain matrix (chemical potential + hopping + pairing)
6. **Bell current & rates** — $J_{nm} = 2\,\text{Im}(\psi_n^* H_{nm} \psi_m)$, rates via positive-current formula
7. **Evolution** — RK4 for both Schrödinger (ψ) and Bell master equation (ρ)
8. **Audit** — Hermiticity, norm, nonnegativity, equivariance checks
9. **Report** — JSON + Markdown output with promotion language scanning

Two open questions for you in the plan:
- **Go module path** — `bellmipt` vs a fully-qualified path?
- **Dependencies** — zero external deps (hand-rolled dense LA) vs `gonum`?

Good — the physics research confirmed all formulas and flagged one important correction. Let me update the implementation plan to incorporate it.

Edited implementation_plan.md

The implementation plan has been updated with confirmed physics from the research:

**Key corrections applied:**
1. **Periodic boundary JW sign** — changed from WARNING to NOTE. No separate parity sign correction is needed in the full Fock-space approach; the `Create`/`Annihilate` operators handle it automatically.
2. **Operator application order** — clarified right-to-left application with composed signs for compound operators like $c_0^\dagger c_{L-1}$.
3. **v0 evolution coupling** — explicitly stated that rates are held fixed during RK4 sub-steps (O(dt²) splitting error, acceptable for dt=0.001).

The plan is ready for your review. Please approve it (or provide feedback) so I can proceed with implementation. The two open questions remain:
- **Module path**: `bellmipt` vs fully-qualified?
- **Dependencies**: zero-dep or `gonum`?



# BELL-MIPT-0001: One-Shot Go Toy Check

Compute Bell jump rates for a finite fermionic lattice model (Kitaev chain) and numerically verify equivariance — i.e., that the Bell master-equation distribution ρ(t) tracks |ψ(t)|² over time.

> [!IMPORTANT]
> **This is a toy analysis only.** No MIPT claim, no holography claim, no Bell-jumps-equal-measurements claim. The output report must explicitly state `physics_claim: "none"` and `toy_analysis_only: true`.

## User Review Required

> [!IMPORTANT]
> **Complex number library**: Go has no native complex matrix library. The plan uses `complex128` slices for dense matrices and hand-rolled linear algebra (matrix-vector multiply, RK4). For a 6-site chain the Hilbert space is 2⁶=64, so dense matrices are fine. Do you want to use any external library (e.g., `gonum/mat`) or keep it dependency-free?

> [!NOTE]
> **Periodic boundary Kitaev chain**: In the full Fock-space approach (building the complete 2^N × 2^N matrix), the Jordan-Wigner string automatically handles the periodic boundary bond (N-1, 0). No separate parity sign correction is needed — that subtlety only arises in the BdG single-particle or momentum-space formalisms. We simply add the wrap-around bond and let the `Create`/`Annihilate` operators compute the correct signs.

## Open Questions

1. **Go module path**: Should the module be `bellmipt` or something like `github.com/chaschel/bft-mipt`? The plan assumes `bellmipt` for simplicity.
2. **gonum dependency**: Should we keep this zero-dependency, or use `gonum` for matrix operations? The plan below assumes zero external dependencies since the matrices are tiny (64×64).

---

## Proposed Changes

The project structure will be:

```
bft-mipt/
├── go.mod
├── bellmipt.json                    # Example config
├── cmd/
│   └── bellmipt/
│       └── main.go                  # Entry point
├── internal/
│   ├── config/
│   │   └── config.go                # JSON config parsing + defaults
│   ├── basis/
│   │   └── basis.go                 # Fock basis enumeration (uint64 bitstrings)
│   │   └── basis_test.go
│   ├── fermion/
│   │   └── operators.go             # Creation/annihilation with Jordan-Wigner signs
│   │   └── operators_test.go
│   ├── hamiltonian/
│   │   └── kitaev.go                # Build finite Kitaev chain Hamiltonian matrix
│   │   └── kitaev_test.go
│   ├── bell/
│   │   └── current.go               # Bell probability current J_nm
│   │   └── rates.go                 # Bell jump rates
│   │   └── master.go                # Bell master equation dρ/dt
│   │   └── bell_test.go
│   ├── evolve/
│   │   └── schrodinger.go           # RK4 for dψ/dt = -iHψ
│   │   └── rk4.go                   # Generic complex RK4
│   │   └── evolve_test.go
│   ├── audit/
│   │   └── audit.go                 # Hermiticity, norm, equivariance checks
│   │   └── audit_test.go
│   └── report/
│       └── report.go                # JSON + Markdown report generation
│       └── language.go              # Forbidden promotion language checker
│       └── report_test.go
└── out/                             # Default output directory (gitignored)
```

---

### Component 1: Module & Entry Point

#### [NEW] [go.mod](file:///home/chaschel/Documents/ibm/go/bft-mipt/go.mod)
- Module `bellmipt`, Go 1.22+
- Zero external dependencies

#### [NEW] [main.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/cmd/bellmipt/main.go)
- Parse `--config` (optional) and `--out` (required) flags
- If `--config` omitted, use built-in default config (6-site periodic Kitaev chain)
- Orchestrate: config → basis → Hamiltonian → evolve → audit → report
- Exit 0 on pass/inconclusive, exit 1 on internal error only
- Copy input config to `out/input.json`

---

### Component 2: Config

#### [NEW] [config.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/config/config.go)

```go
type Config struct {
    SchemaVersion string       `json:"schema_version"`
    Model         string       `json:"model"`           // "finite_kitaev_chain"
    Sites         int          `json:"sites"`
    Boundary      string       `json:"boundary"`        // "open" or "periodic"
    Parameters    Parameters   `json:"parameters"`
    InitialState  InitialState `json:"initial_state"`
    Time          TimeConfig   `json:"time"`
    Audit         AuditConfig  `json:"audit"`
}

type Parameters struct {
    Mu    float64 `json:"mu"`    // chemical potential
    T     float64 `json:"t"`     // hopping amplitude
    Delta float64 `json:"delta"` // pairing amplitude
}

type InitialState struct {
    Type string `json:"type"` // "random_normalized"
    Seed int64  `json:"seed"`
}

type TimeConfig struct {
    Dt    float64 `json:"dt"`
    Steps int     `json:"steps"`
}

type AuditConfig struct {
    HermitianTolerance   float64 `json:"hermitian_tolerance"`
    NormTolerance        float64 `json:"norm_tolerance"`
    EquivarianceTolerance float64 `json:"equivariance_tolerance"`
}
```

- `DefaultConfig()` returns the 6-site example from the prompt
- `LoadConfig(path string)` reads JSON file, validates fields

---

### Component 3: Basis

#### [NEW] [basis.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/basis/basis.go)

- Represent each Fock state as a `uint64` bitstring. Bit `j` = 1 means site `j` is occupied.
- `EnumerateBasis(sites int) []uint64` — returns all 2^sites states in canonical order (0, 1, 2, …, 2^sites-1)
- `Occupation(state uint64, site int) bool` — check if site is occupied
- `PopCount(state uint64) int` — number of occupied sites
- `PopCountBelow(state uint64, site int) int` — number of occupied sites with index < site (for Jordan-Wigner sign)

#### [NEW] [basis_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/basis/basis_test.go)
- Test: `EnumerateBasis(3)` returns 8 states
- Test: occupation queries on known states
- Test: `PopCountBelow` for Jordan-Wigner

---

### Component 4: Fermion Operators

#### [NEW] [operators.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/fermion/operators.go)

Jordan-Wigner convention: creation operator $c_j^\dagger$ acting on a Fock state $|n\rangle$:

$$c_j^\dagger |n\rangle = \begin{cases} 0 & \text{if site } j \text{ is occupied in } n \\ (-1)^{\sum_{k<j} n_k} \cdot |n \oplus 2^j\rangle & \text{otherwise} \end{cases}$$

Annihilation operator $c_j$ is the adjoint.

Functions:
- `Create(state uint64, site int) (newState uint64, sign int, ok bool)` — returns new state, ±1 sign, and whether it succeeded
- `Annihilate(state uint64, site int) (newState uint64, sign int, ok bool)` — same

#### [NEW] [operators_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/fermion/operators_test.go)
- Test: creating on empty site gives correct sign
- Test: creating on occupied site returns `ok=false`
- Test: annihilating occupied site gives correct sign
- Test: anticommutation: $\{c_i, c_j^\dagger\} = \delta_{ij}$ verified on small basis

---

### Component 5: Hamiltonian (Kitaev Chain)

#### [NEW] [kitaev.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/hamiltonian/kitaev.go)

The Kitaev chain Hamiltonian in second quantization:

$$H = -\mu \sum_{j=0}^{L-1} c_j^\dagger c_j - t \sum_{j=0}^{L'-1} (c_j^\dagger c_{j+1} + c_{j+1}^\dagger c_j) + \Delta \sum_{j=0}^{L'-1} (c_j c_{j+1} + c_{j+1}^\dagger c_j^\dagger)$$

Where $L' = L-2$ for open boundary, $L' = L-1$ for periodic boundary (with site indices mod L).

> [!NOTE]
> For periodic boundary, we simply add the bond (L-1, 0) alongside the other nearest-neighbor bonds. The Jordan-Wigner signs are automatically correct because the `Create`/`Annihilate` operators compute signs based on occupied sites below the target site in our canonical ordering. No separate boundary sign correction is needed in the full Fock-space approach.

Functions:
- `BuildMatrix(cfg config.Config, basisStates []uint64) []complex128` — returns a dense `dim × dim` Hamiltonian stored as a flat row-major `[]complex128`
- Helper: `applyHopping`, `applyChemicalPotential`, `applyPairing`
- The matrix is built by iterating over all basis states and applying each term
- For each bond (i, j), applies operators right-to-left: first `c_j` then `c†_i` to get the intermediate and final states with composed signs

#### [NEW] [kitaev_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/hamiltonian/kitaev_test.go)
- Test: 2-site Hamiltonian matches hand calculation
- Test: Hermiticity check: $H_{nm} = H_{mn}^*$ for all n, m
- Test: dimension is correct (2^sites × 2^sites)

---

### Component 6: Bell Current and Rates

#### [NEW] [current.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bell/current.go)

Bell probability current between configurations n and m:

$$J_{nm} = 2 \, \text{Im}(\psi_n^* \, H_{nm} \, \psi_m)$$

- `ComputeCurrentMatrix(H []complex128, psi []complex128, dim int) []float64` — returns the full J matrix (antisymmetric: $J_{nm} = -J_{mn}$)

#### [NEW] [rates.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bell/rates.go)

Bell jump rate for transition $n \leftarrow m$:

$$\sigma_{n \leftarrow m} = \frac{\max(J_{nm}, 0)}{|\psi_m|^2}$$

When $|\psi_m|^2 < \varepsilon$ (a small cutoff, e.g. $10^{-15}$), set the rate to 0 to avoid division by zero.

- `ComputeRateMatrix(J []float64, psi []complex128, dim int) []float64` — returns the rate matrix

#### [NEW] [master.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bell/master.go)

Bell master equation:

$$\frac{d\rho_n}{dt} = \sum_m \left[ \sigma_{n \leftarrow m} \, \rho_m - \sigma_{m \leftarrow n} \, \rho_n \right]$$

- `MasterEquationRHS(rates []float64, rho []float64, dim int) []float64` — computes dρ/dt given current rate matrix and ρ

#### [NEW] [bell_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bell/bell_test.go)
- Test: current antisymmetry $J_{nm} = -J_{mn}$
- Test: all rates are nonneg
- Test: for ρ = |ψ|², the master-equation RHS should match d|ψ|²/dt from Schrödinger evolution (equivariance at one step)

---

### Component 7: Evolution

#### [NEW] [rk4.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/evolve/rk4.go)

Generic RK4 stepper for complex-valued and real-valued ODEs:

```go
// RK4StepComplex advances y by one step: y' = f(y)
func RK4StepComplex(y []complex128, dt float64, f func([]complex128) []complex128) []complex128

// RK4StepReal advances y by one step: y' = f(y)
func RK4StepReal(y []float64, dt float64, f func([]float64) []float64) []float64
```

#### [NEW] [schrodinger.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/evolve/schrodinger.go)

Schrödinger equation: $d\psi/dt = -i H \psi$

- `SchrodingerRHS(H []complex128, psi []complex128, dim int) []complex128` — returns $-iH\psi$
- `MatVecComplex(H []complex128, v []complex128, dim int) []complex128` — dense matrix-vector multiply

Main evolution loop (in `main.go` or a dedicated runner):
```
for step := 0; step < cfg.Time.Steps; step++ {
    1. Compute Bell current J from (H, ψ)
    2. Compute Bell rate matrix σ from (J, ψ)
    3. RK4 step ψ via Schrödinger
    4. RK4 step ρ via Bell master equation using σ(ψ_current)
       (Note: rates depend on ψ, which changes during the step.
        For simplicity, use rates computed at the start of each step.
        This is first-order accurate for ρ, which is fine for the audit.)
    5. Audit: compare ρ vs |ψ|²
    6. Record metrics
}
```

> [!NOTE]
> **v0 coupling strategy**: The rate matrix σ depends on ψ, which changes during the time step. For v0, we use the rate matrix computed at the **start** of each step and hold it fixed during the RK4 sub-steps for ρ. This introduces a splitting error of O(dt²), which is acceptable for dt=0.001. The equivariance audit will confirm this is sufficient. If needed in a future version, we can recompute rates at each RK4 sub-step for higher accuracy.

#### [NEW] [evolve_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/evolve/evolve_test.go)
- Test: RK4 on a simple ODE (e.g., dy/dt = iy) matches analytic solution
- Test: Schrödinger evolution preserves norm to within tolerance
- Test: known 2-site system matches analytic time evolution

---

### Component 8: Audit

#### [NEW] [audit.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/audit/audit.go)

```go
type AuditResult struct {
    HamiltonianHermitian           bool
    StateNormPreserved             bool
    RatesNonnegative               bool
    EquivarianceErrorWithinTol     bool
    MaxHermitianError              float64
    MaxNormError                   float64
    MaxNegativeRateViolation       float64
    MaxEquivarianceL1Error         float64
    MeanTotalBellActivity          float64
    MaxTotalBellActivity           float64
}
```

Functions:
- `CheckHermiticity(H []complex128, dim int, tol float64) (bool, float64)` — max |H_nm - conj(H_mn)|
- `CheckNorm(psi []complex128, tol float64) (bool, float64)` — |‖ψ‖² - 1|
- `CheckRatesNonneg(rates []float64, dim int) (bool, float64)` — min rate, should be ≥ 0
- `CheckEquivariance(rho []float64, psi []complex128, tol float64) (bool, float64)` — L1 norm of ρ - |ψ|²
- `TotalBellActivity(rates []float64, rho []float64, dim int) float64` — Σ_{n≠m} σ_{n←m} ρ_m

#### [NEW] [audit_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/audit/audit_test.go)
- Test: known Hermitian matrix passes
- Test: known non-Hermitian matrix fails
- Test: equivariance check on trivial (diagonal H) system

---

### Component 9: Report

#### [NEW] [report.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/report/report.go)

Generates both `report.json` and `report.md`:

```go
type Report struct {
    SchemaVersion   string            `json:"schema_version"`
    ToyID           string            `json:"toy_id"`
    ToyAnalysisOnly bool              `json:"toy_analysis_only"`
    PhysicsClaim    string            `json:"physics_claim"`
    Model           string            `json:"model"`
    Sites           int               `json:"sites"`
    HilbertDim      int               `json:"hilbert_dim"`
    Goal            string            `json:"goal"`
    GoalStatus      string            `json:"goal_status"`
    Checks          Checks            `json:"checks"`
    Metrics         Metrics           `json:"metrics"`
    DebtStatus      DebtStatus        `json:"debt_status"`
}
```

- `GoalStatus` is one of: `"toy_goal_passed"`, `"toy_goal_failed"`, `"toy_goal_inconclusive"`
- The goal passes only if ALL checks pass
- The goal is inconclusive if NaN/Inf is detected in metrics
- Otherwise the goal fails

`DebtStatus` always contains the fixed values from the prompt:
```go
DebtStatus{
    NeedMap:                "unpaid",
    NeedInvariant:          "partially_paid_equivariance_only",
    NeedToyCheck:           "partially_paid_rate_algebra_only",
    NeedNullModel:          "unpaid",
    NeedObstruction:        "bell_jumps_are_not_measurements",
    NeedFaithfulnessReview: "unpaid",
}
```

#### [NEW] [language.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/report/language.go)

Checks that report text does not contain forbidden promotion language:
- Scans the generated Markdown for words like "proves", "demonstrates", "confirms", "establishes", "shows that Bell jumps are measurements", etc.
- Returns a list of violations (if any)

#### [NEW] [report_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/report/report_test.go)
- Test: report contains required fields
- Test: no forbidden promotion language in generated markdown
- Test: deterministic output for fixed config/seed

---

### Example Config File

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

## Key Algorithmic Details

### Jordan-Wigner Sign Computation

For site $j$ and state $|n\rangle$ represented as a bitstring:

$$\text{sign} = (-1)^{\text{popcount}(n \,\&\, ((1 \ll j) - 1))}$$

This counts the number of set bits below position $j$ using bit masking.

### Evolution Coupling

The ψ evolution is independent (pure Schrödinger), but the ρ evolution depends on ψ because the rates σ depend on ψ. The coupling strategy for v0:

1. At each time step, compute rates from current ψ
2. RK4-advance ψ (independent of ρ)
3. RK4-advance ρ using rates computed at start of step (held fixed during the RK4 sub-steps)
4. This introduces a splitting error of O(dt²), acceptable for dt=0.001

### Periodic Boundary Jordan-Wigner

For periodic boundary, we add the bond (L-1, 0). The `Create`/`Annihilate` operators handle signs correctly because they count occupied sites below the target site in canonical ordering. For compound operators like $c_0^\dagger c_{L-1}$, we apply right-to-left: first `Annihilate(state, L-1)` (which counts bits 0..L-2), then `Create(intermediate, 0)` (which counts no bits below 0). The signs compose multiplicatively. No separate boundary parity correction is needed in the full Fock-space approach.

---

## Verification Plan

### Automated Tests

```bash
go test ./... -v -count=1
```

Tests cover:
1. **Basis**: correct enumeration, occupation queries, popcount
2. **Fermion operators**: Jordan-Wigner signs, anticommutation relations
3. **Hamiltonian**: Hermiticity, 2-site hand calculation match, correct dimension
4. **Bell current**: antisymmetry $J_{nm} = -J_{mn}$
5. **Bell rates**: nonnegativity
6. **Evolution**: norm preservation, analytic match for simple systems
7. **Equivariance**: ρ tracks |ψ|² for small system
8. **Report**: deterministic output for fixed seed, no forbidden language
9. **Language check**: detects promotion language in test strings

### Manual Verification

```bash
# Run with default config
go run ./cmd/bellmipt --out out/bellmipt-default

# Run with explicit config
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run

# Inspect output
cat out/bellmipt-default/report.json | python3 -m json.tool
cat out/bellmipt-default/report.md
```

Verify that:
- `goal_status` is `"toy_goal_passed"`
- All checks are `true`
- Equivariance L1 error is below tolerance
- `physics_claim` is `"none"`
- `toy_analysis_only` is `true`
- No promotion language in report.md

