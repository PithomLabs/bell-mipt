# BELL-MIPT-0001 — Go Implementation Plan

Target: `cmd/bellmipt`, one command, finite Kitaev-chain Bell-rate equivariance toy check.
Scope lock from the ticket: no MIPT, no holography, no Bell-jumps-equal-measurements, no
conditional-wave-function bridge, no random trajectories. `physics_claim: none` is a hard
invariant of every run, not a hope.

This document is the **plan**, not the code. It fixes package layout, types, function
signatures, the two physics-correctness traps that are easy to get wrong, the audit/decision
logic, and a milestone order with explicit tests per milestone.

---

## 1. Repository layout

```
cmd/bellmipt/
    main.go                # flags, load config, call run.Run, write outputs

internal/bellmipt/
    config.go               # Config struct, JSON tags, defaults, validation
    basis.go                 # occupation-number basis, popcount helpers
    fermion.go               # generic Jordan-Wigner operator algebra
    hamiltonian.go            # finite Kitaev chain builder + Hermiticity check
    belldynamics.go           # Bell current J, Bell rates R
    evolution.go              # RK4 for ψ, frozen-generator step for ρ
    audit.go                  # per-step diagnostics, tolerance checks
    report.go                 # Report struct, goal_status logic, JSON/MD writers
    forbidden.go               # promotion-language denylist + scanner
    run.go                    # Run(cfg) orchestrates everything above

internal/bellmipt/testdata/    # small fixed configs for golden/determinism tests
```

`internal/` keeps everything non-exported outside `cmd/bellmipt`, which matches the "one
simple Go command" requirement — `main.go` should be thin, `run.go` should be the thing
unit tests call directly without shelling out.

**No external dependencies needed.** Standard library only: `encoding/json`, `flag`, `math`,
`math/cmplx`, `math/rand`, `os`, `path/filepath`, `testing`. Worth confirming since this repo
won't need a Go module proxy in the sandbox or CI.

---

## 2. Config — `config.go`

```go
type Config struct {
    SchemaVersion string       `json:"schema_version"`
    Model         string       `json:"model"`
    Sites         int          `json:"sites"`
    Boundary      string       `json:"boundary"` // "open" | "periodic"
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
    Type string `json:"type"` // only "random_normalized" supported
    Seed int64  `json:"seed"`
}

type TimeConfig struct {
    Dt    float64 `json:"dt"`
    Steps int     `json:"steps"`
}

type AuditConfig struct {
    HermitianTolerance     float64 `json:"hermitian_tolerance"`
    NormTolerance          float64 `json:"norm_tolerance"`
    EquivarianceTolerance  float64 `json:"equivariance_tolerance"`
}

func DefaultConfig() Config        // literal values from the ticket's example JSON
func LoadConfig(path string) (Config, error) // os.Open + json.Decode, falls back to DefaultConfig if path == ""
func (c Config) Validate() error   // rejects anything outside the locked-down scope below
```

`Validate()` is where "keep it flexible but small" gets enforced in code, not just in prose:

```text
Model != "finite_kitaev_chain"        -> error
Boundary not in {"open","periodic"}    -> error
InitialState.Type != "random_normalized" -> error
Sites < 2                              -> error
Time.Steps < 1 or Time.Dt <= 0         -> error
```

Any unsupported value fails loudly at startup rather than silently doing something
unspecified — this is the Go-level analogue of "ideas enter free, promotion costs debt":
config values enter free, but only validated values get to drive a run.

---

## 3. Basis — `basis.go`

`Sites = N`, Hilbert dimension `D = 1 << N`. A basis state is just the integer `0..D-1` read
as a bitstring, bit `i` = occupation of site `i`.

```go
func HilbertDim(n int) int               // 1 << n, with an overflow/size guard (see §9)
func PopcountBelow(state uint64, i int) int // popcount(state & ((1<<i) - 1))
```

`PopcountBelow` is the one function the entire JW sign convention rests on — see §4.

Test: enumerate `N=2,3` by hand, assert `D` states, no duplicates, bit patterns match the
expected `0..D-1` sequence.

---

## 4. Fermion operators — `fermion.go`

**Convention**, stated explicitly so it's a documented choice rather than an implicit guess:

```text
c_i^† |n⟩ = (1 - n_i) · (-1)^(Σ_{j<i} n_j) |n with bit i set⟩
c_i   |n⟩ =     n_i   · (-1)^(Σ_{j<i} n_j) |n with bit i cleared⟩
```

i.e. the JW string is the parity of occupied sites *strictly below* index `i`, evaluated on
the state immediately before that operator's own flip.

```go
type OpKind int
const ( Create OpKind = iota; Annihilate )

type OpTerm struct { Kind OpKind; Site int }

// ApplyOperatorString applies ops in physics left-to-right notation
// (ops[0] is leftmost), but executes them rightmost-first, as QM requires.
func ApplyOperatorString(state uint64, ops []OpTerm) (result uint64, sign int, ok bool)
```

Sketch (pseudocode-level, not final):

```go
func ApplyOperatorString(state uint64, ops []OpTerm) (uint64, int, bool) {
    s, sign := state, 1
    for k := len(ops) - 1; k >= 0; k-- {
        op := ops[k]
        bit := (s >> op.Site) & 1
        switch op.Kind {
        case Annihilate:
            if bit == 0 { return 0, 0, false }
            s &^= 1 << op.Site
        case Create:
            if bit == 1 { return 0, 0, false }
            s |= 1 << op.Site
        }
        if PopcountBelow(s_before_this_flip, op.Site)%2 == 1 { sign = -sign }
    }
    return s, sign, true
}
```

### The one trap worth naming up front: periodic boundary signs

Do **not** write a hand-special-cased "nearest neighbor hopping sign" function and then bolt
on a separate sign rule for the wraparound bond `(N-1, 0)`. That's the classic Kitaev-ring
bug: the wrap bond is adjacent *physically* but far apart in the *linear JW ordering*, and a
hand-rolled "adjacent sites always get sign +1" shortcut will silently get it wrong.

The fix is architectural, not a patch: implement `ApplyOperatorString` generically for
*any* pair of site indices, never assuming adjacency. Then both `c_i^† c_{i+1}` (open/bulk
bonds) and `c_{N-1}^† c_0` (the periodic wrap bond) go through the exact same code path with
no special case. Correctness of the wrap bond becomes a corollary of correctness of the
general two-operator composition, which is covered by the Hermiticity test in §5 and the
small-system equivariance test in §7 — if the wrap sign were wrong, `H` would still come out
Hermitian (sign errors of this kind are self-adjoint-preserving), so Hermiticity alone won't
catch this bug. The small-system equivariance test against a hand-checked 2–3 site periodic
chain is the test that actually catches it. **Flag this test as load-bearing, not optional.**

Test: `{c_i, c_j^†} = δ_ij` on a small (N=2 or 3) system, checked by direct composition —
apply `c_i` then `c_i^†` to an occupied site, confirm identity with sign +1; apply to an
empty site, confirm annihilation to zero.

---

## 5. Hamiltonian — `hamiltonian.go`

```go
type Matrix struct {
    Dim  int
    Data []complex128 // row-major, Dim*Dim
}
func (m *Matrix) At(r, c int) complex128
func (m *Matrix) Set(r, c int, v complex128)

func BuildHamiltonian(cfg Config) (*Matrix, error)
func HermitianError(h *Matrix) float64 // max_{n,m} |H_nm - conj(H_mn)|
```

Terms, built by iterating basis states and calling `ApplyOperatorString`:

```text
H = -μ Σ_i  n_i
    -t Σ_⟨i,j⟩ (c_i^† c_j + c_j^† c_i)
    +Δ Σ_⟨i,j⟩ (c_i^† c_j^† + c_j c_i)
```

where `⟨i,j⟩` ranges over `(0,1), (1,2), …, (N-2,N-1)` for open boundary, plus `(N-1,0)` for
periodic. `n_i` is just `ApplyOperatorString(state, []OpTerm{{Create,i},{Annihilate,i}})`,
which falls out of the same generic machinery for free (sign always cancels to +1 when valid).

Sign convention for μ, t, Δ is a free implementation choice — the ticket doesn't pin one, and
none of the pass/fail checks (Hermiticity, norm, nonnegative rates, equivariance) depend on
which overall sign is used, as long as it's applied consistently. Document the chosen
convention in a doc-comment so a future reader doesn't have to reverse-engineer it.

Test: for `N ∈ {2,4,6}` × `boundary ∈ {open,periodic}` × a few seeds, `HermitianError(H) <
1e-10`. This is necessary but, per §4, not sufficient on its own for catching wrap-sign bugs.

---

## 6. Bell current and rates — `belldynamics.go`

```go
func BellCurrent(h *Matrix, psi []complex128) []float64 // J_nm flattened, row-major
func BellRates(j []float64, psi []complex128, dim int) []float64 // R_nm flattened
```

```text
J_nm = 2 · Im( conj(ψ_n) · H_nm · ψ_m )
R(n←m) = max(J_nm, 0) / |ψ_m|²        for n ≠ m
```

`J` is antisymmetric (`J_nm = -J_mn`) whenever `H` is Hermitian — this is a property of the
formula, not an independent assumption, so test it directly rather than just trusting it.

**Near-zero-population guard.** `R(n←m)` divides by `|ψ_m|²`. A basis state can transiently
have near-zero population during unitary evolution without ever being exactly zero, so a
naive division risks huge or `Inf`/`NaN` rates feeding into the ρ evolution. Guard at the
point of use, not by clamping the rate itself:

```go
const popFloor = 1e-12
contribution := 0.0
if rho_m > popFloor { contribution = rate_n_given_m * rho_m } // else treat as 0
```

i.e. compute `R(n←m)·ρ_m` directly with a floor check on `ρ_m`/`|ψ_m|²` rather than computing
`R(n←m)` as a free-standing huge number and hoping multiplication by a tiny `ρ_m` saves you —
floating point doesn't guarantee that cancellation is clean. This guard should be implemented
once in `evolution.go` where the master equation is actually evaluated (§7), not duplicated.

Tests:
- antisymmetry: random `H`-consistent `ψ`, assert `|J_nm + J_mn| < 1e-12` for all pairs.
- nonnegativity: assert `R_nm ≥ -1e-12` everywhere after the `max(·,0)` clamp — this is a
  regression test against a flipped `max`/`min` typo, not a deep physics claim.

---

## 7. Evolution — `evolution.go`

### ψ: standard RK4 for `dψ/dt = -iHψ`

```go
func SchrodingerRK4Step(h *Matrix, psi []complex128, dt float64) []complex128
```

Four matrix-vector products per step, `O(D²)` each. For the default `D=64` this is trivial.

### ρ: frozen-generator step, per the ticket's "RK4 or Euler-small-step" allowance

The honest complication: rates depend on the *instantaneous* `ψ(t)`, since `J` and `R` are
built from `ψ`. The master equation is therefore a **time-dependent** linear ODE
`dρ/dt = R(t)·ρ`, not a fixed-generator one. Two implementation options:

- (a) Recompute `R` at every RK4 sub-stage of the ρ step (`t`, `t+dt/2`, `t+dt`), requiring
  `ψ` at those same sub-stage times.
- (b) **Freeze `R` at `R(t_k)`** (computed from `ψ(t_k)` at the start of the step) and use
  that fixed `R` across a single RK4 (or Euler) step to advance `ρ`.

**Plan picks (b).** It's simpler, it's explicitly licensed by the ticket's "RK4 or
Euler-small-step" wording, and at the default `dt = 0.001` the per-step variation in `R` is a
second-order effect on top of an already-first-order time discretization of the *whole*
toy check. Document this choice in a comment; (a) is a legitimate future refinement if the
equivariance error ever turns out to be dominated by rate staleness rather than by `dt` itself
— that's a diagnosable, falsifiable thing to check later, not something to guess about now.

```go
func MasterEqStep(rates []float64, rho []float64, dim int, dt float64) []float64
// builds the dim x dim generator R_nm = rate(n<-m) off-diagonal,
// R_nn = -sum_{m != n} rate(m<-n), then does one RK4 (or Euler) step
// of drho/dt = R . rho, applying the popFloor guard from §6 at point of use.
```

The master equation is probability-conserving by construction (`Σ_n` of the RHS is zero), so
don't renormalize `ρ` after each step — if it drifts, that drift is itself useful diagnostic
information, not something to paper over. Track it as an internal sanity metric even though
it isn't in the locked report schema (see §8).

Driving loop in `run.go`:

```text
for k in 0..steps-1:
    R_k          = BellRates(BellCurrent(H, psi), psi)   // from psi at START of step
    psi_next     = SchrodingerRK4Step(H, psi, dt)
    rho_next     = MasterEqStep(R_k, rho, dim, dt)
    record audit point at (psi_next, rho_next)            // see §8
    psi, rho = psi_next, rho_next
```

---

## 8. Audit — `audit.go`

```go
type AuditAccumulator struct {
    MaxHermitianError       float64
    MaxNormError             float64
    MaxNegativeRateViolation float64
    MaxEquivarianceL1Error   float64
    MeanTotalBellActivity    float64
    MaxTotalBellActivity     float64
    SawNaNOrInf              bool
}

func (a *AuditAccumulator) RecordStep(psi []complex128, rho, rates []float64)
func (a *AuditAccumulator) Finalize(hermitianErr float64) // folds in the one-time H check
```

Per-step work:

```text
born_n        = |psi_n|^2
norm_error    = |Σ_n born_n - 1|
l1_error      = Σ_n |rho_n - born_n|
total_activity = Σ_{n,m} R_nm        // diagnostic, "how much Bell activity is happening"
NaN/Inf check on psi, rho, rates -> sets SawNaNOrInf
```

`MaxNegativeRateViolation` is effectively a regression assertion (§6) — under correct code it
should always read `0.0`; a nonzero value here means the clamp logic is broken, not that
something interesting was discovered about the physics.

---

## 9. Report and decision logic — `report.go`

```go
type Report struct {
    SchemaVersion string            `json:"schema_version"`
    ToyID         string            `json:"toy_id"`
    ToyAnalysisOnly bool            `json:"toy_analysis_only"` // always true
    PhysicsClaim  string            `json:"physics_claim"`     // always "none"
    Model         string            `json:"model"`
    Sites         int               `json:"sites"`
    HilbertDim    int               `json:"hilbert_dim"`
    Goal          string            `json:"goal"`
    GoalStatus    string            `json:"goal_status"`
    Checks        Checks            `json:"checks"`
    Metrics       Metrics           `json:"metrics"`
    DebtStatus    DebtStatus        `json:"debt_status"`
}
```

`DebtStatus` is a **fixed literal**, not computed from the run — it's EBP bookkeeping about
what this ticket does and doesn't establish, not a numeric result:

```go
func DefaultDebtStatus() DebtStatus {
    return DebtStatus{
        NeedMap:                "unpaid",
        NeedInvariant:          "partially_paid_equivariance_only",
        NeedToyCheck:           "partially_paid_rate_algebra_only",
        NeedNullModel:          "unpaid",
        NeedObstruction:        "bell_jumps_are_not_measurements",
        NeedFaithfulnessReview: "unpaid",
    }
}
```

### `goal_status` decision tree

```text
1. SawNaNOrInf                                          -> "toy_goal_inconclusive"
2. HermitianError  > audit.hermitian_tolerance           -> "toy_goal_failed"
   NormError       > audit.norm_tolerance                -> "toy_goal_failed"
   NegativeRateViolation > 0                              -> "toy_goal_failed"
   (these three indicate implementation bugs, not subtle physics — hard fail, not "inconclusive")
3. EquivarianceL1Error > audit.equivariance_tolerance     -> "toy_goal_failed"
   (this is the actual hypothesis under test)
4. otherwise                                              -> "toy_goal_passed"
```

Keeping (2) and (3) as separate tiers matters: a Hermiticity or norm failure means "something
is broken," while an equivariance failure on a Hermitian, norm-preserving system means
"the thing we set out to check didn't check out" — those are different findings and the
report should let a reader tell them apart at a glance, not just see one flat `failed`.

### `report.md`

Plain prose rendering of the same fields. Opens with the three non-claim lines verbatim
(`physics_claim: none`, `toy_analysis_only: true`, the goal statement) before anything else,
so the disclaimer can't be missed by a reader who only looks at the top of the file.

---

## 10. Forbidden-language guard — `forbidden.go`

```go
var forbiddenTerms = []string{
    "proves", "proven", "confirms", "confirmed", "establishes", "demonstrates",
    "QED", "holography", "holographic", "MIPT", "measurement-induced",
    "is equivalent to measurement", "therefore true",
}

func ScanForbiddenLanguage(text string) []string // returns matches found, empty = clean
```

Run against the generated `report.md` prose and against any free-text fields. The fixed
`debt_status` strings (e.g. `"bell_jumps_are_not_measurements"`) are intentionally exempt —
they're disclaiming, not promoting — so the scanner should run on rendered prose, not on the
literal JSON keys/values of `debt_status`.

Test: generate the default-config report, assert `ScanForbiddenLanguage` returns empty on
both `report.md` and the prose-bearing string fields of `report.json`.

---

## 11. CLI — `cmd/bellmipt/main.go`

```go
func main() {
    configPath := flag.String("config", "", "path to JSON config (optional)")
    outDir     := flag.String("out", "out/bellmipt-run", "output directory")
    flag.Parse()

    cfg, err := bellmipt.LoadConfig(*configPath) // empty path -> DefaultConfig()
    // Validate, Run, write input.json/report.json/report.md, os.Exit(1) on hard error
}
```

No subcommands, no extra flags — matches "do not add subcommands yet."

---

## 12. Test plan, mapped 1:1 to the ticket's required list

| Ticket requirement | Test | Where |
|---|---|---|
| basis enumeration | `TestBasisEnumeration` | `basis_test.go` |
| fermion creation/annihilation signs | `TestFermionAnticommutation` | `fermion_test.go` |
| Hamiltonian Hermiticity | `TestHamiltonianHermitian` | `hamiltonian_test.go` |
| Bell current antisymmetry | `TestBellCurrentAntisymmetric` | `belldynamics_test.go` |
| nonnegative rates | `TestRatesNonnegative` | `belldynamics_test.go` |
| equivariance audit, small system | `TestEquivarianceSmallSystem` (N=2,3; catches §4's wrap-sign trap) | `run_test.go` |
| deterministic output, fixed config/seed | `TestDeterministicOutput` (run twice, byte-identical report) | `run_test.go` |
| no forbidden promotion language | `TestNoForbiddenLanguage` | `forbidden_test.go` |
| `go test ./...` / `go run ./cmd/bellmipt --out /tmp/bellmipt-default` | end-to-end smoke test | `cmd/bellmipt/main_test.go` |

---

## 13. Milestones

```text
M0  Scaffolding: go.mod, package dirs, empty main.go that compiles.
M1  basis.go + fermion.go + their tests.
M2  hamiltonian.go + Hermiticity test (open + periodic).
M3  belldynamics.go (current, rates, popFloor guard) + antisymmetry/nonnegativity tests.
M4  evolution.go: psi RK4 + norm-preservation sanity check.
M5  evolution.go: rho frozen-generator step + small-system equivariance test.
M6  audit.go: full-trajectory accumulation + goal_status decision tree.
M7  config.go (load/default/validate) + report.go + forbidden.go + their tests.
M8  cmd/bellmipt/main.go wiring + end-to-end test + default-config smoke run.
```

Each milestone should leave `go test ./...` green before moving to the next — no milestone
depends on a later one, so this order is also a valid bisection order if something regresses.

---

## 14. Performance note (so no one over-engineers this)

Default config: `sites=6` → `D=64`, `steps=1000`. Per step: 4 dense `D×D` mat-vecs for ψ
(`O(D²)` each) plus one `D×D` rate-matrix build and apply for ρ. Total ≈ `1000 × O(D²)` ≈
tens of thousands of flops — sub-second in Go with the naive dense loops above. Dense
`complex128` matrices are fine up to roughly `sites ≲ 10–12` (`D ≲ 4096`); beyond that a
sparse representation would be needed, but that's out of scope for this ticket and isn't
worth pre-building.

---

## 15. Decisions flagged for confirmation (software-level, not physics-level debt)

These are implementation choices this plan made on the ticket's behalf, listed explicitly
rather than buried, in case any should be pinned differently before M2:

1. Sign convention for `μ, t, Δ` terms in `hamiltonian.go` (§5) — free choice, documented in
   code, doesn't affect any pass/fail check.
2. Frozen-generator (option (b)) vs. sub-stage-recomputed (option (a)) rate matrix for the ρ
   RK4 step (§7) — picked (b) for simplicity; (a) is a defensible later refinement if
   equivariance error analysis ever calls for it.
3. `popFloor = 1e-12` threshold for the near-zero-population guard (§6) — arbitrary but
   should be well below `audit.norm_tolerance` and `audit.equivariance_tolerance` defaults so
   it never masks a real signal.
4. Forbidden-term denylist contents (§10) — a starting list, not exhaustive; easy to extend
   without touching any other file.

None of these are `needMap`/`needInvariant`/EBP-style physics debt — they're ordinary
software design choices on a ticket whose own debt ledger (`physics_claim: none`,
`needMap: unpaid`, etc.) is already fixed by the spec and shouldn't be touched by the Go
implementation.
