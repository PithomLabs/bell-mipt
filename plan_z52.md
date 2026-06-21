# Implementation Plan — BELL-MIPT-0001 in Go

Below is a complete, file-by-file implementation plan that satisfies the prompt. It is deliberately non-bloated: one module, one command, one report. No MIPT, no holography, no CWF, no trajectories.

---

## 1. Goals recap (locked scope)

| Item | Status |
|---|---|
| Compute Bell jump rates on a finite Kitaev chain | ✅ in scope |
| Evolve ψ by Schrödinger and ρ by Bell master equation | ✅ in scope |
| Audit ρ(t) vs \|ψ(t)\|² (equivariance) | ✅ in scope |
| Emit JSON + Markdown report with explicit goal_status | ✅ in scope |
| Trajectory sampling / MIPT / holography / CWF | ❌ forbidden |

The single exit decision is one of three strings: `toy_goal_passed`, `toy_goal_failed`, `toy_goal_inconclusive`.

---

## 2. Repository layout

```
bell-mipt/
  go.mod                       # module github.com/<you>/bell-mipt
  cmd/
    bellmipt/
      main.go                  # flags: --config, --out; orchestrates run
  internal/
    config/
      config.go                # JSON schema + default config
      config_test.go
    basis/
      basis.go                 # uint64 bitstring basis + enumeration
      basis_test.go
    fermion/
      ops.go                   # Jordan–Wigner c, c† matrix elements
      ops_test.go
    hamiltonian/
      kitaev.go                # build sparse H for finite Kitaev chain
      kitaev_test.go
    bell/
      current.go               # J_nm = 2 Im(ψ_n* H_nm ψ_m)
      rates.go                 # rate(n<-m) = max(J_nm,0)/|ψ_m|^2
      master.go                # dρ/dt for Bell master equation
      bell_test.go
    evolve/
      schrodinger.go           # RK4 for ψ:  dψ/dt = -i H ψ
      master.go                # RK4 for ρ:  dρ/dt = L(ρ)
      evolve_test.go
    audit/
      audit.go                 # hermiticity, norm, rates≥0, equivariance L1
      audit_test.go
    report/
      report.go                # write input.json, report.json, report.md
      report_test.go
  go.sum
```

No `internal/cli`, no `internal/logging`, no plugin system. Just direct packages.

---

## 3. Module dependencies

Only stdlib plus one tiny dependency for complex128 linear algebra helpers if needed — but the entire pipeline can be written with `math/cmplx` and hand-rolled sparse matrix ops on a `map[[2]uint64]complex128`. **Prefer zero external deps.** Use:

- `math`, `math/cmplx`
- `encoding/json`
- `flag`, `os`, `path/filepath`
- `fmt`, `sort`, `strings`
- `math/rand/v2` (only for `random_normalized` initial state)
- `testing`

---

## 4. Core data structures

### 4.1 Config (`internal/config/config.go`)

```go
type Config struct {
    SchemaVersion string         `json:"schema_version"`
    Model         string         `json:"model"`            // "finite_kitaev_chain"
    Sites         int            `json:"sites"`
    Boundary      string         `json:"boundary"`         // "open" | "periodic"
    Parameters    Parameters     `json:"parameters"`
    InitialState  InitialStateSpec `json:"initial_state"`
    Time          TimeSpec       `json:"time"`
    Audit         AuditSpec      `json:"audit"`
}

type Parameters struct {
    Mu    float64 `json:"mu"`
    T     float64 `json:"t"`
    Delta float64 `json:"delta"`
}

type InitialStateSpec struct {
    Type string `json:"type"`   // "random_normalized"
    Seed uint64 `json:"seed"`
}

type TimeSpec struct {
    Dt    float64 `json:"dt"`
    Steps int     `json:"steps"`
}

type AuditSpec struct {
    HermitianTolerance      float64 `json:"hermitian_tolerance"`
    NormTolerance           float64 `json:"norm_tolerance"`
    EquivarianceTolerance   float64 `json:"equivariance_tolerance"`
}

func Default() Config { /* built-in config from prompt */ }
func Load(path string) (Config, error)
func (c Config) Validate() error   // sites in [1, 20], boundary ∈ {open,periodic}, dt > 0, steps > 0, ...
```

### 4.2 Basis (`internal/basis/basis.go`)

```go
// State is a fermion occupation bitstring. Bit i = occupancy of site i.
type State uint64

func NumStates(sites int) uint64         // 1 << sites
func Enumerate(sites int) []State        // 0..(1<<sites)-1
func Occupation(s State, i int) int      // (s >> i) & 1
func ParityBelow(s State, i int) int     // (-1)^(Σ_{j<i} n_j); returns ±1
func FlipBit(s State, i int) State       // s ^ (1<<i)
```

Hilbert dim is `1 << sites` (full Fock space — no symmetry projection in v0).

### 4.3 Fermion ops (`internal/fermion/ops.go`)

Implement matrix elements directly on bitstrings. No matrix storage — just two functions returning `(target State, amplitude complex128, ok bool)`:

```go
// c_i† |s⟩  → (1-n_i) * (-1)^{Σ_{j<i} n_j} |s ⊕ 2^i⟩
func Create(s basis.State, i int) (basis.State, complex128, bool)

// c_i |s⟩   → n_i * (-1)^{Σ_{j<i} n_j} |s ⊕ 2^i⟩
func Annihilate(s basis.State, i int) (basis.State, complex128, bool)
```

`ok=false` when the operator kills the state (annihilation on empty, creation on filled).

### 4.4 Hamiltonian (`internal/hamiltonian/kitaev.go`)

Sparse matrix as `map[[2]uint64]complex128` keyed by `(row, col)` = `(n, m)`. Equivalently `map[State]map[State]complex128`. The latter is friendlier.

```go
type SparseH struct {
    Sites int
    Entries map[basis.State]map[basis.State]complex128
}

func BuildKitaev(cfg config.Config) *SparseH
```

Terms (one convention; lock it in a doc comment):

```
H = -μ Σ_i (n_i - 1/2)
    - t Σ_<i,j> (c_i† c_j + c_j† c_i)
    + Δ Σ_<i,j> (c_i† c_j† + c_j c_i)
```

where `<i,j>` runs over nearest-neighbor pairs `(i, i+1)`, with `(sites-1, 0)` added when `boundary == "periodic"`.

Build procedure for each pair `(i, j)`:
1. For each basis state `s`:
   - Hopping `c_i† c_j`: apply Annihilate(s, j) → (s', a1, ok1); if ok1, apply Create(s', i) → (s'', a2, ok2); if ok2, add `-t * a1 * a2` to `H[s''][s]`.
   - Hopping `c_j† c_i`: symmetric.
   - Pair creation `c_i† c_j†`: apply Create(s, j) then Create(s', i); add `+Δ * a1 * a2`.
   - Pair annihilation `c_j c_i`: apply Annihilate(s, i) then Annihilate(s', j); add `+Δ * a1 * a2` (Hermitian conjugate, real coefficient).
2. Chemical potential: diagonal `-μ * (Σ_i n_i - sites/2)` added to `H[s][s]`.

After construction, symmetrize: ensure `H[m][n] = conj(H[n][m])` and merge both directions into the map. (This is automatic if all four terms above are added; the symmetrize step is a safety net.)

### 4.5 Bell current and rates (`internal/bell/`)

```go
// J[n][m] = 2 * Im(conj(psi[n]) * H[n][m] * psi[m])
// For sparse H, iterate only nonzero H[n][m].
func Currents(H *hamiltonian.SparseH, psi map[basis.State]complex128) map[basis.State]map[basis.State]float64

// rate(n<-m) = max(J[n][m], 0) / |psi[m]|^2   (with regularization)
// If |psi[m]|^2 < eps, rate = 0 (and J[n][m] is also ~0 by construction).
func Rates(J CurrentMap, psi map[basis.State]complex128, eps float64) RateMap
```

**Regularization rule** (critical for equivariance): when `|ψ_m|² < eps`, set `rate(n←m) = 0` for all `n`. Choose `eps = 1e-14` (well below `norm_tolerance = 1e-8`). Verify in tests that this does not perturb equivariance beyond tolerance, because `J_nm ∝ ψ_m` so the numerator vanishes at the same rate.

### 4.6 Master equation (`internal/bell/master.go`)

```go
// dρ_n/dt = Σ_m [ rate(n<-m) ρ_m - rate(m<-n) ρ_n ]
func Deriv(rho []float64, rates RateMap, states []basis.State) []float64
```

Implementation detail: precompute, for each `m`, the list of outgoing rates `rate(n←m)` and incoming rates `rate(m←n)`. The derivative is then `O(nnz per state)` rather than `O(dim^2)`.

---

## 5. Time evolution (`internal/evolve/`)

### 5.1 Schrödinger (`schrodinger.go`)

```go
// dψ/dt = -i H ψ   →   RK4
func StepSchrodinger(H *hamiltonian.SparseH, psi []complex128, dt float64) []complex128
```

Use fixed classical RK4 with complex state. `k1 = -i H ψ`, `k2 = -i H (ψ + dt/2 k1)`, etc. No adaptive stepping in v0.

### 5.2 Master (`master.go`)

```go
// RK4 on dρ/dt = L(ρ)
func StepMaster(rates RateMap, rho []float64, states []basis.State, dt float64) []float64
```

Same RK4 pattern, real-valued.

### 5.3 Time-stepping driver

```go
func Run(cfg config.Config) (*audit.Report, error) {
    H := hamiltonian.BuildKitaev(cfg)
    states := basis.Enumerate(cfg.Sites)
    psi := initialState(cfg, states)       // random_normalized
    rho := probabilities(psi)              // |ψ_n|^2 at t=0

    var rep audit.Report
    rep.HermitianError = audit.HermitianError(H)
    for step := 0; step < cfg.Time.Steps; step++ {
        psi = evolve.StepSchrodinger(H, psi, cfg.Time.Dt)
        rates := bell.Rates(bell.Currents(H, psi), psi, eps)
        rho = evolve.StepMaster(rates, rho, states, cfg.Time.Dt)

        rep.Update(audit.Snapshot{
            Step: step,
            NormError:       audit.NormError(psi),
            RatesNegative:   audit.MaxNegativeRate(rates),
            EquivarianceL1:  audit.EquivarianceL1(psi, rho),
            TotalActivity:   audit.TotalActivity(rates, rho),
        })
    }
    rep.Finalize(cfg)
    return &rep, nil
}
```

Note: rates are recomputed every step from the current ψ. The master equation then uses those rates for one RK4 substep. This is the cleanest equivariance audit.

---

## 6. Audit (`internal/audit/audit.go`)

```go
type Snapshot struct {
    Step            int
    NormError       float64  // |‖ψ‖² − 1|
    RatesNegative   float64  // max(0, −min rate)
    EquivarianceL1  float64  // Σ_n |ρ_n − |ψ_n|²|
    TotalActivity   float64  // Σ_{n,m} rate(n←m) ρ_m
}

type Report struct {
    SchemaVersion string
    ToyID         string  // "BELL-MIPT-0001"
    ToyAnalysisOnly bool
    PhysicsClaim  string  // "none"
    Model         string
    Sites         int
    HilbertDim    int
    Goal          string
    GoalStatus    string  // toy_goal_passed | _failed | _inconclusive
    Checks        Checks
    Metrics       Metrics
    DebtStatus    DebtStatus
}

type Checks struct {
    HamiltonianHermitian                   bool
    StateNormPreserved                     bool
    RatesNonnegative                       bool
    EquivarianceErrorWithinTolerance       bool
}

type Metrics struct {
    MaxHermitianError          float64
    MaxNormError               float64
    MaxNegativeRateViolation   float64
    MaxEquivarianceL1Error     float64
    MeanTotalBellActivity      float64
    MaxTotalBellActivity       float64
}

type DebtStatus struct {
    NeedMap                string  // "unpaid"
    NeedInvariant          string  // "partially_paid_equivariance_only"
    NeedToyCheck           string  // "partially_paid_rate_algebra_only"
    NeedNullModel          string  // "unpaid"
    NeedObstruction        string  // "bell_jumps_are_not_measurements"
    NeedFaithfulnessReview string  // "unpaid"
}
```

`Report.Finalize` decides `GoalStatus`:

```
if MaxHermitianError > HermitianTolerance       → fail
if MaxNormError      > NormTolerance            → fail
if MaxNegativeRateViolation > 0 (beyond ~1e-12) → fail
if MaxEquivarianceL1Error  > EquivarianceTolerance → fail
if any NaN/Inf encountered                     → inconclusive
else                                           → passed
```

Equivariance L1 is `Σ_n |ρ_n(t) − |ψ_n(t)|²|`, summed across all steps and tracked as the per-step max.

---

## 7. Report writing (`internal/report/report.go`)

Three files in `--out`:

- `input.json` — the resolved config (default or loaded), pretty-printed, schema_version `bell_mipt_toy_v0`.
- `report.json` — the `audit.Report` struct marshaled with schema_version `bell_mipt_report_v0`, sorted keys off (use struct tags).
- `report.md` — a short Markdown summary.

Markdown template (literal strings, no promotional language):

```markdown
# BELL-MIPT-0001 Toy Report

- toy_id: BELL-MIPT-0001
- toy_analysis_only: true
- physics_claim: none
- model: {{.Model}}
- sites: {{.Sites}}
- hilbert_dim: {{.HilbertDim}}
- goal: compute_bell_rates_and_verify_equivariance
- goal_status: {{.GoalStatus}}

## Checks

| Check | Result |
|---|---|
| Hamiltonian Hermitian | {{.Checks.HamiltonianHermitian}} |
| State norm preserved  | {{.Checks.StateNormPreserved}} |
| Rates nonnegative     | {{.Checks.RatesNonnegative}} |
| Equivariance within tol | {{.Checks.EquivarianceErrorWithinTolerance}} |

## Metrics

| Metric | Value |
|---|---|
| max_hermitian_error | {{.Metrics.MaxHermitianError}} |
| max_norm_error | {{.Metrics.MaxNormError}} |
| max_negative_rate_violation | {{.Metrics.MaxNegativeRateViolation}} |
| max_equivariance_l1_error | {{.Metrics.MaxEquivarianceL1Error}} |
| mean_total_bell_activity | {{.Metrics.MeanTotalBellActivity}} |
| max_total_bell_activity | {{.Metrics.MaxTotalBellActivity}} |

## Debt status

- needMap: unpaid
- needInvariant: partially_paid_equivariance_only
- needToyCheck: partially_paid_rate_algebra_only
- needNullModel: unpaid
- needObstruction: bell_jumps_are_not_measurements
- needFaithfulnessReview: unpaid

## Obstruction note

Bell jumps are not measurements. No MIPT, holography, or conditional-wave-function claim is made or implied by this report.
```

A `ForbiddenPhrases` list is checked in `report_test.go`:

```go
var forbidden = []string{
    "MIPT", "measurement-induced", "holograph", "holographic",
    "black hole", "conditional wave function", "trajectory",
    "spacetime", "emergent geometry", "AdS",
}
```

The test fails if any forbidden phrase appears (case-insensitive substring) in either output file. The strings "bell_jumps_are_not_measurements" and the obstruction note are explicitly allow-listed.

---

## 8. `cmd/bellmipt/main.go`

```go
func main() {
    configPath := flag.String("config", "", "path to JSON config (default if omitted)")
    outDir     := flag.String("out", "out/bellmipt-run", "output directory")
    flag.Parse()

    cfg := config.Default()
    if *configPath != "" {
        c, err := config.Load(*configPath)
        if err != nil { fatal(err) }
        cfg = c
    }
    if err := cfg.Validate(); err != nil { fatal(err) }

    rep, err := evolve.Run(cfg)
    if err != nil { fatal(err) }

    if err := report.WriteAll(*outDir, cfg, rep); err != nil { fatal(err) }

    fmt.Printf("BELL-MIPT-0001: %s\n", rep.GoalStatus)
    fmt.Printf("equivariance L1 (max): %.3e\n", rep.Metrics.MaxEquivarianceL1Error)
    os.Exit(exitCode(rep.GoalStatus))
}
```

Exit code: `0` on `toy_goal_passed`, `1` on `toy_goal_failed`, `2` on `toy_goal_inconclusive`. This makes the toy usable as a CI gate.

---

## 9. Build order (mapped to prompt steps 1–8)

| # | Prompt step | Package(s) | Deliverable | Gate |
|---|---|---|---|---|
| 1 | Basis | `internal/basis` | `State`, `Enumerate`, `ParityBelow` | unit test: enumeration count = `1<<sites`, parity sign for canonical states |
| 2 | Fermion ops | `internal/fermion` | `Create`, `Annihilate` | test: `c c† + c† c = 1` on every site for every state; `c² = (c†)² = 0`; sign on first/last site |
| 3 | Hamiltonian | `internal/hamiltonian` | `BuildKitaev` | test: `H = H†` (Hermiticity L∞ < 1e-14); known two-site analytic spectrum |
| 4 | Bell current | `internal/bell/current.go` | `Currents` | test: `J[n][m] == -J[m][n]` for random ψ; `Σ_nm J[n][m] == 0` |
| 5 | Bell rates | `internal/bell/rates.go` | `Rates` | test: every rate ≥ 0 (within 1e-15); division-by-zero guard fires on a hand-crafted ψ with a zero component |
| 6 | Schrödinger + master | `internal/evolve` | `StepSchrodinger`, `StepMaster` | test: norm preservation over 100 steps < 1e-10; ρ stays nonnegative (clip tiny negatives from RK4 to 0 with audit) |
| 7 | Audit | `internal/audit` | `Snapshot`, `Report`, `Finalize` | test: synthetic passing and failing reports produce correct `GoalStatus` |
| 8 | Report + main | `internal/report`, `cmd/bellmipt` | `input.json`, `report.json`, `report.md` | test: forbidden phrases absent; deterministic output for fixed seed |

Each row is independently testable; commit after each gate.

---

## 10. Sign-convention and edge-case checklist

1. **Bit ordering**: bit `i` ↔ site `i`. Document this in `basis.go`.
2. **Jordan–Wigner string**: `(-1)^{Σ_{j<i} n_j}`. Test on state `|1010⟩` for `i=2`: parity below = `(-1)^1 = -1`.
3. **Hopping pair**: apply `c_j` first (annihilate source), then `c_i†` (create target). Order matters for the sign.
4. **Pair creation**: apply `c_j†` first, then `c_i†`. Use `i<j` to avoid double counting.
5. **Boundary**: when `boundary == "periodic"` and `sites == 2`, the pair `(1,0)` appears twice (once as `(0,1)`, once as `(1,0)` wrap). For `sites == 2` periodic, decide explicitly: count each bond once per direction or once total. Document the choice. Recommend: **count each unordered bond once**, including the wrap bond `(sites-1, 0)`. For `sites == 2`, the wrap bond coincides with the bulk bond — disallow `sites < 3` for `periodic` to avoid ambiguity, or document that `sites == 2` periodic == `sites == 2` open with double coupling. Pick the cleaner rule (disallow) in `Validate`.
6. **Zero components in ψ**: rates guarded by `eps = 1e-14`. Test with a state where one amplitude is exactly zero.
7. **RK4 intermediate negativity in ρ**: clip values in `[-tiny, 0)` to `0` post-step with `tiny = 1e-15`; record the clip magnitude in audit if non-trivial (in v0: just clip silently — RK4 errors are well below tolerance).
8. **NaN/Inf detection**: any NaN/Inf in ψ, ρ, or rates sets `GoalStatus = toy_goal_inconclusive` and aborts further steps.
9. **Determinism**: `rand/v2` with explicit seed. Marshal config with sorted keys to make `input.json` byte-identical across runs.

---

## 11. Test plan (`go test ./...`)

| Package | Test | Property |
|---|---|---|
| `basis` | `TestEnumerateCount` | `len(Enumerate(L)) == 1<<L` for L=1..10 |
| `basis` | `TestParityBelow` | spot-checks on canonical states |
| `fermion` | `TestAnticommutator` | `c_i c_i† + c_i† c_i = I` on full basis, every site |
| `fermion` | `TestNilpotency` | `c_i² = (c_i†)² = 0` |
| `fermion` | `TestJWSign` | sign of `c_i†|0...1_i...0⟩` matches formula |
| `hamiltonian` | `TestHermiticity` | `‖H − H†‖∞ < 1e-13` for open & periodic, L=4,6 |
| `hamiltonian` | `TestTwoSiteSpectrum` | analytic eigenvalues of 2-site chain |
| `bell` | `TestCurrentAntisymmetry` | `J[n][m] + J[m][n] == 0` for random ψ |
| `bell` | `TestCurrentContinuity` | `Σ_m J[n][m]` matches finite-diff of `\|ψ_n\|²` over one Euler step |
| `bell` | `TestRatesNonnegative` | all rates ≥ `-1e-15` |
| `bell` | `TestRatesRegularization` | zero-amplitude component produces zero rates, no NaN |
| `evolve` | `TestSchrodingerNorm` | `|‖ψ‖² − 1| < 1e-10` over 1000 steps, L=4 |
| `evolve` | `TestEquivarianceSmall` | L=3, 200 steps, equivariance L1 < 1e-7 |
| `audit` | `TestFinalizePass` | synthetic clean report → `toy_goal_passed` |
| `audit` | `TestFinalizeFailHermitian` | injected Hermitian error → `toy_goal_failed` |
| `audit` | `TestFinalizeInconclusiveNaN` | injected NaN → `toy_goal_inconclusive` |
| `report` | `TestNoForbiddenPhrases` | grep report.json and report.md |
| `report` | `TestDeterministic` | two runs with same config+seed → byte-identical `report.json` |
| `cmd/bellmipt` | `TestDefaultRun` | `go run ./cmd/bellmipt --out /tmp/x` returns exit 0; report file exists; `goal_status == "toy_goal_passed"` |

---

## 12. Validation gates

Run in this exact order before declaring the ticket done:

```bash
gofmt -l .                         # must be empty
go vet ./...                       # must be clean
go test ./...                      # all green
go run ./cmd/bellmipt --out /tmp/bellmipt-default
# inspect:
#   /tmp/bellmipt-default/input.json
#   /tmp/bellmipt-default/report.json   (goal_status == "toy_goal_passed")
#   /tmp/bellmipt-default/report.md
go run ./cmd/bellmipt --config examples/kitaev6.json --out /tmp/bellmipt-6
```

A second run with the same config must produce a byte-identical `report.json` (modulo absolute timestamps if any — there should be none in v0).

---

## 13. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Equivariance drifts over long runs due to RK4 truncation error compounding differently in ψ and ρ | Keep `dt ≤ 0.001` and `steps ≤ 2000` for v0; document that longer runs need a symplectic or Crank–Nicolson integrator in a follow-up ticket |
| Division by `|ψ_m|²` blows up near nodes | `eps = 1e-14` guard; test explicitly |
| Periodic boundary double-counts the wrap bond for `sites == 2` | `Validate` rejects `sites < 3` when `boundary == "periodic"` |
| User-set tolerances too loose to catch real bugs | `Validate` enforces `HermitianTolerance ≥ 1e-14`, `NormTolerance ≥ 1e-12`, `EquivarianceTolerance ≥ 1e-10` (lower bounds on strictness, not upper) — i.e. user may set stricter, never laxer |
| Markdown accidentally contains a forbidden phrase via metric values | Forbidden-phrase test scans the rendered files, not the template |
| Future scope creep (trajectories, MIPT) sneaks in | Code review checklist item: any PR touching `cmd/bellmipt` must not introduce `trajectory`, `MIPT`, `CWF`, `holograph` identifiers; CI grep enforces |

---

## 14. Out-of-scope guards (explicit non-implementations)

The following are **forbidden** in this ticket and should be rejected at code review:

- Trajectory sampling (Poisson process on Bell rates, wave-function jumps).
- Conditional wave functions, pilot waves, collapse models.
- Holographic dictionaries, tensor-network geometry, black-hole analogs.
- Subcommands, plugin systems, telemetry.
- Symmetry projection (parity, particle number) on the Hilbert space — full Fock space only.
- Adaptive time stepping.
- Lean formalization, agent loops, auto-prompts.

Any of these belongs in `BELL-MIPT-0001.1` or later.

---

## 15. Definition of done

The ticket is closed when **all** of the following hold:

1. `go test ./...` is green on a clean checkout.
2. `go run ./cmd/bellmipt --out /tmp/bellmipt-default` exits `0` and `report.json` has `"goal_status": "toy_goal_passed"`.
3. `report.json` and `report.md` contain `physics_claim: none`, `toy_analysis_only: true`, and the exact `debt_status` block from the prompt.
4. No forbidden phrase appears in any output file.
5. A second identical run produces byte-identical `report.json`.
6. The obstruction note `bell_jumps_are_not_measurements` is present in `report.md`.

That is one input, one run, one honest report.
