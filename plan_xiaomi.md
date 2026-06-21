# BELL-MIPT-0001: Go Implementation Plan

## 1. Project Structure

```
bellmipt/
├── cmd/
│   └── bellmipt/
│       └── main.go              # CLI entrypoint
├── internal/
│   ├── config/
│   │   └── config.go            # JSON schema parsing + default config
│   ├── basis/
│   │   └── basis.go             # uint64 bitstring enumeration
│   ├── fermion/
│   │   └── fermion.go           # Jordan-Wigner creation/annihilation operators
│   ├── hamiltonian/
│   │   └── hamiltonian.go       # Finite Kitaev chain construction
│   ├── bell/
│   │   └── bell.go              # Bell probability currents and jump rates
│   ├── evolve/
│   │   └── evolve.go            # RK4 Schrödinger + Bell master equation
│   ├── audit/
│   │   └── audit.go             # Equivariance comparison ρ(t) vs |ψ(t)|²
│   └── report/
│       └── report.go            # JSON + Markdown report generation
├── go.mod
└── bellmipt_test.go             # (or split across packages)
```

---

## 2. Implementation Steps (In Order)

### Step 1 — Basis (`internal/basis`)

**Purpose:** Represent each fermion Fock-space configuration as a `uint64` bitstring.

- Define `type BasisState = uint64`.
- Implement `func Enumerate(nSites int) []BasisState` returning all `2^nSites` states (bit `i` set = site `i` occupied).
- Hilbert dimension = `1 << nSites`; validate `nSites <= 63`.
- The number of set bits gives particle count; provide a helper `func Popcount(s BasisState) int`.

---

### Step 2 — Fermion Operators (`internal/fermion`)

**Purpose:** Implement creation `c†_i` and annihilation `c_i` with Jordan-Wigner sign.

- For a given basis state `|n⟩`, `c_i |n⟩`:
  - If site `i` is unoccupied → result is 0.
  - Otherwise → clear bit `i`, multiply by `(-1)^{sum of occupied bits below i}` (the JW string).
- `c†_i` is the adjoint: if site `i` is occupied → 0; otherwise set bit `i` with the same sign factor.
- Both operators return `(newState BasisState, amplitude complex128, ok bool)`.

---

### Step 3 — Hamiltonian (`internal/hamiltonian`)

**Purpose:** Build the finite Kitaev-chain Hamiltonian matrix.

The Kitaev chain Hamiltonian is:

```
H = -μ Σ_i n_i - t Σ_{<i,j>} (c†_i c_j + h.c.) + Δ Σ_{<i,j>} (c†_i c†_j + h.c.)
```

- **Input:** `nSites`, `mu`, `t`, `delta`, `boundary` (open or periodic).
- **Output:** `[][]complex128` Hermitian matrix of dimension `2^nSites`.
- **Construction:** For each basis state, apply each term to get column entries:
  - Chemical potential: diagonal, `-μ` per occupied site.
  - Hopping: `c†_i c_j` and its conjugate, for nearest-neighbor pairs.
  - Pair creation/annihilation: `c†_i c†_j` and `c_i c_j`, for nearest-neighbor pairs.
- **Neighbor pairs:** For open boundary, `i = 0..nSites-2`, pair `(i, i+1)`. For periodic, add `(nSites-1, 0)`.
- **Hermiticity audit:** After construction, verify `|H[i][j] - conj(H[j][i])| < hermitian_tolerance`.

---

### Step 4 — Bell Current & Rates (`internal/bell`)

**Purpose:** Compute Bell probability currents and positive-current jump rates.

- **Bell probability current** between configurations `n` and `m`:

  ```
  J_nm = 2 * Im(conj(ψ_n) * H_nm * ψ_m)
  ```

  This is antisymmetric: `J_nm = -J_mn`.

- **Bell jump rate** from `m` to `n`:

  ```
  rate(n←m) = max(J_nm, 0) / |ψ_m|²
  ```

  Guard against division by zero when `|ψ_m|² = 0` (set rate to 0).

- **Output:** Rate matrix `rate[n][m]` (size `dim × dim`).
- **Audit:** Verify all rates ≥ 0 (within `norm_tolerance`).

---

### Step 5 — Evolution (`internal/evolve`)

#### 5a. Schrödinger Evolution for ψ

- **Equation:** `dψ/dt = -i H ψ` (ℏ = 1).
- **Method:** 4th-order Runge-Kutta (RK4).
- **Norm check:** After each step, verify `|ψ|²` sums to 1 within `norm_tolerance`.

#### 5b. Bell Master Equation for ρ

- **Equation:**

  ```
  dρ_n/dt = Σ_m [ rate(n←m) * ρ_m  -  rate(m←n) * ρ_n ]
  ```

- **Method:** RK4 or small-step Euler (document recommends RK4).
- **Initialization:** `ρ_n(0) = |ψ_n(0)|²`.
- **Norm preservation:** `Σ_n ρ_n` should remain 1 within tolerance.

---

### Step 6 — Audit (`internal/audit`)

**Purpose:** At every time step, compare `ρ_n(t)` against `|ψ_n(t)|²`.

- Compute **L1 error**: `Σ_n |ρ_n(t) - |ψ_n(t)|²|`.
- Track `max_equivariance_l1_error` across all steps.
- **Pass condition:** `max_equivariance_l1_error < equivariance_tolerance`.

---

### Step 7 — Report (`internal/report`)

**Purpose:** Write `report.json` and `report.md`.

#### report.json

Follow the schema from the document exactly:

```json
{
  "schema_version": "bell_mipt_report_v0",
  "toy_id": "BELL-MIPT-0001",
  "toy_analysis_only": true,
  "physics_claim": "none",
  "model": "finite_kitaev_chain",
  "sites": <int>,
  "hilbert_dim": <int>,
  "goal": "compute_bell_rates_and_verify_equivariance",
  "goal_status": "toy_goal_passed" | "toy_goal_failed" | "toy_goal_inconclusive",
  "checks": {
    "hamiltonian_hermitian": <bool>,
    "state_norm_preserved": <bool>,
    "rates_nonnegative": <bool>,
    "equivariance_error_within_tolerance": <bool>
  },
  "metrics": {
    "max_hermitian_error": <float>,
    "max_norm_error": <float>,
    "max_negative_rate_violation": <float>,
    "max_equivariance_l1_error": <float>,
    "mean_total_bell_activity": <float>,
    "max_total_bell_activity": <float>
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

#### report.md

A human-readable summary covering: config used, check results, metrics, and `goal_status`.

#### Output Files

```
out/bellmipt-run/
  input.json      ← copy of the config used
  report.json     ← structured results
  report.md       ← human-readable summary
```

---

### Step 8 — CLI Entrypoint (`cmd/bellmipt/main.go`)

- Flags: `--config` (path to JSON), `--out` (output directory).
- If `--config` is omitted, use the built-in default config from the document.
- Sequence: parse config → build basis → build Hamiltonian → audit Hermiticity → evolve ψ and ρ in lockstep → compute Bell rates at each step → audit equivariance → write reports → exit with status code.

---

## 3. Pass / Fail / Inconclusive Logic

| Status | Condition |
|---|---|
| **toy_goal_passed** | All checks true |
| **toy_goal_failed** | Any check false |
| **toy_goal_inconclusive** | Numerical instability prevents judgment (NaN/Inf detected) |

---

## 4. Test Plan

| Test | Package | What It Verifies |
|---|---|---|
| Basis enumeration | `basis` | All `2^n` states generated, no duplicates, no overflow for valid `nSites` |
| Fermion operator signs | `fermion` | JW signs correct for known 2- and 3-site cases; `c†c` gives occupation |
| Hamiltonian Hermiticity | `hamiltonian` | `|H[i][j] - conj(H[j][i])| < tolerance` for both open/periodic boundary |
| Bell current antisymmetry | `bell` | `J_nm + J_mn = 0` for all pairs on a known state |
| Nonnegative rates | `bell` | All `rate(n←m) ≥ 0` for random ψ |
| Equivariance audit | `audit` | For a small system (4 sites), `ρ(t)` tracks `|ψ(t)|²` within `1e-5` L1 error over 1000 steps |
| Deterministic output | integration | Same config + seed → byte-identical `report.json` |
| No promotion language | `report` | Scan output for forbidden strings: "MIPT", "holography", "black hole", "measurement", etc. — only the literal strings `"bell_jumps_are_not_measurements"` and `"physics_claim": "none"` are permitted |

---

## 5. Key Constraints (from the document)

- **No subcommands.** Single `go run ./cmd/bellmipt` command.
- **No random trajectories** in this ticket (deferred to `BELL-MIPT-0001.1`).
- **No MIPT, holography, conditional wave functions, Lean, or agents.**
- Report must state `physics_claim: "none"` and `toy_analysis_only: true`.
- Debt status fields are hardcoded as specified — they are honest accounting, not results.

---

## 6. Suggested Build Order

```
Week 1:  basis → fermion → hamiltonian → unit tests for all three
Week 2:  bell (currents + rates) → evolve (Schrödinger + master equation)
Week 3:  audit → report → CLI integration → integration tests
```

Each step is independently testable before the next is built. The Hamiltonian module is the critical dependency; once it passes Hermiticity checks, everything downstream can proceed with confidence.
