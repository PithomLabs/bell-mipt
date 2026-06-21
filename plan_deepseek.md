# Implementation Plan: `BELL-MIPT-0001` – One‑Shot Go Toy Check

This document describes the step‑by‑step implementation of a minimal Go program that computes Bell jump rates for a finite fermionic lattice and verifies equivariance numerically. The final command is

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

The program must produce a directory containing `input.json`, `report.json`, and `report.md`. No MIPT, holography, or conditional‑wave‑function logic is implemented.

---

## 1. Project Structure

```
bellmipt/
├── cmd/
│   └── bellmipt/
│       └── main.go           // entry point, flag parsing, orchestration
├── internal/
│   ├── config/
│   │   └── config.go         // JSON schema, defaults, loading
│   ├── basis/
│   │   └── basis.go          // bitstring basis enumeration
│   ├── hamiltonian/
│   │   └── hamiltonian.go    // Kitaev‑chain construction, Hermiticity
│   ├── dynamics/
│   │   ├── schrodinger.go    // ψ RK4 integrator
│   │   └── bellmaster.go     // ρ master‑equation step
│   ├── audit/
│   │   └── audit.go          // equivariance, norm, rate checks
│   └── report/
│       ├── report.go         // JSON & Markdown writers
│       └── language.go       // forbidden‑phrase check
├── go.mod
└── go.sum
```

All core logic resides in `internal/`. The `cmd` package is the only one with `main`.

---

## 2. Configuration (`internal/config`)

### 2.1 JSON Schema (v0)

```go
type Config struct {
    SchemaVersion string `json:"schema_version"`
    Model         string `json:"model"`          // only "finite_kitaev_chain"
    Sites         int    `json:"sites"`
    Boundary      string `json:"boundary"`       // "open" or "periodic"
    Parameters    struct {
        Mu    float64 `json:"mu"`
        T     float64 `json:"t"`
        Delta float64 `json:"delta"`
    } `json:"parameters"`
    InitialState struct {
        Type string `json:"type"`   // only "random_normalized"
        Seed int64  `json:"seed"`
    } `json:"initial_state"`
    Time struct {
        DT    float64 `json:"dt"`
        Steps int     `json:"steps"`
    } `json:"time"`
    Audit struct {
        HermitianTolerance     float64 `json:"hermitian_tolerance"`
        NormTolerance          float64 `json:"norm_tolerance"`
        EquivarianceTolerance  float64 `json:"equivariance_tolerance"`
    } `json:"audit"`
}
```

### 2.2 Default Configuration

If `--config` is omitted, use the built‑in defaults shown in the prompt (6 sites, periodic, mu=1, t=1, delta=0.5, random_normalized with seed 12345, dt=0.001, steps=1000, tolerances).

Implementation:

```go
func DefaultConfig() Config {
    return Config{
        SchemaVersion: "bell_mipt_toy_v0",
        Model:         "finite_kitaev_chain",
        Sites:         6,
        Boundary:      "periodic",
        Parameters: struct{...}{Mu:1, T:1, Delta:0.5},
        ...
    }
}
```

---

## 3. Basis (`internal/basis`)

### 3.1 Bitstring Representation

The system has `N` sites, each can be occupied (1) or empty (0). A configuration is represented as a `uint64` with the lowest `N` bits significant.

```go
type Config uint64

func (c Config) Bit(i int) bool {
    // i from 0 to N-1
    return (c>>i)&1 == 1
}
```

### 3.2 Enumerate All Configurations

```go
func Enumerate(sites int) []Config {
    n := 1 << sites
    configs := make([]Config, n)
    for i := 0; i < n; i++ {
        configs[i] = Config(i)
    }
    return configs
}
```

Test: `Enumerate(3)` returns 8 configurations with bits matching the integer. Verify the length is `2^sites`.

---

## 4. Fermion Operators (`internal/hamiltonian/operators.go`)

### 4.1 Jordan‑Wigner String

Given a configuration `c` and site `j`, the creation operator `c_j^†` flips the bit at `j` and multiplies by the Jordan‑Wigner phase `(-1)^{∑_{k<j} n_k}`.

```go
func JWSign(c Config, j int) complex128 {
    // count number of occupied sites with index < j
    var parity int
    for k := 0; k < j; k++ {
        if c.Bit(k) {
            parity ^= 1
        }
    }
    if parity == 1 {
        return -1
    }
    return 1
}
```

### 4.2 Action of `c_j^†` and `c_j`

- `c_j^† |c⟩`: if site `j` is empty, produce `sign * |c'⟩` where `c'` is `c` with bit `j` set; else zero.
- `c_j |c⟩`: if site `j` occupied, produce `sign * |c''⟩` where `c''` is `c` with bit `j` cleared; else zero.

Implementation returns `(newConfig, amplitude)`. The amplitude is `JWSign(c, j)`.

Test: on a 2‑site chain, verify anticommutation relations `{c_i, c_j^†}=δ_{i,j}` for all basis states (manually compute a few).

---

## 5. Hamiltonian (`internal/hamiltonian/kitaev.go`)

### 5.1 Kitaev Chain Terms

For `N` sites with boundary condition (open/periodic), construct the Hamiltonian as a sparse matrix of size `2^N × 2^N`.

Hamiltonian in second quantisation:

\[
H = -\mu \sum_{j=1}^N c_j^\dagger c_j \;-\; t \sum_{j=1}^{N-1} \big(c_j^\dagger c_{j+1} + c_{j+1}^\dagger c_j\big) \;+\; \Delta \sum_{j=1}^{N-1} \big(c_j c_{j+1} + c_{j+1}^\dagger c_j^\dagger\big)
\]

For periodic boundary (`N` sites), the sums run `j=1..N` with `j+1` wrapping to 1 (mod N).

### 5.2 Matrix Construction

- Allocate a dense `[][]complex128` of size `dim × dim` (dim = 1<<N). For 6 sites dim=64, dense is acceptable.
- For each basis configuration `c`, apply each term’s operators to find destination config and amplitude, then accumulate into `H[dest][c]`.

**Chemical potential**: for each `j`, `c_j^† c_j` contributes `-μ` if site j occupied, diagonal.

**Hopping**: for each `j`, `c_j^† c_{j+1} + h.c.` calculates `c_{j+1}|c⟩`, then `c_j^†` on result; amplitude from both hops.

**Pair creation/annihilation**: similar.

Implementation detail: function `applyString` that applies `c_j` or `c_j^†` and returns `(newConfig, amplitude)`. Multiply amplitudes.

### 5.3 Hermiticity Check

In the construction, ensure \(H_{mn} = \overline{H_{nm}}\). During testing, compute maximum |H_{mn} - conj(H_{nm})|. In program, perform this check after building H.

```go
func HermiticityError(H [][]complex128) float64 {
    maxErr := 0.0
    for i := range H {
        for j := range H {
            diff := cmplx.Abs(H[i][j] - cmplx.Conj(H[j][i]))
            if diff > maxErr {
                maxErr = diff
            }
        }
    }
    return maxErr
}
```

---

## 6. Bell Probability Current and Rates (`internal/dynamics/bellrates.go`)

### 6.1 Bell Current

Given state vector `ψ` (complex slice of length dim) and Hamiltonian matrix `H`, the Bell probability current from `m` to `n` is:

\[
J_{n←m} = 2\,\mathrm{Im}\big(\overline{\psi_n}\, H_{nm}\, \psi_m\big)
\]

Note: \(J\) is antisymmetric, \(J_{n←m} = -J_{m←n}\).

```go
func BellCurrent(psi []complex128, H [][]complex128, n, m int) float64 {
    return 2 * imag(cmplx.Conj(psi[n]) * H[n][m] * psi[m])
}
```

### 6.2 Bell Jump Rates

Rate from `m` to `n` is \(\max(J_{n←m}, 0)\,/\,|\psi_m|^2\), with safeguard for zero probability:

```go
func JumpRates(psi []complex128, H [][]complex128) [][]float64 {
    dim := len(psi)
    rates := make([][]float64, dim)
    for n := 0; n < dim; n++ {
        rates[n] = make([]float64, dim)
    }
    for m := 0; m < dim; m++ {
        prob := real(psi[m])*real(psi[m]) + imag(psi[m])*imag(psi[m])
        if prob < 1e-15 {
            // set all outgoing rates from m to zero
            continue
        }
        for n := 0; n < dim; n++ {
            if n == m {
                continue
            }
            J := BellCurrent(psi, H, n, m)
            if J > 0 {
                rates[n][m] = J / prob
            } else {
                rates[n][m] = 0
            }
        }
    }
    return rates
}
```

Check non‑negativity: all `rates >= 0` by construction.

---

## 7. Evolution

### 7.1 Schrödinger Dynamics (RK4)

Equation: \( \frac{d\psi}{dt} = -i H \psi \).

```go
func schrodingerDeriv(psi []complex128, H [][]complex128) []complex128 {
    dim := len(psi)
    out := make([]complex128, dim)
    for i := 0; i < dim; i++ {
        var sum complex128
        for j := 0; j < dim; j++ {
            sum += H[i][j] * psi[j]
        }
        out[i] = -1i * sum
    }
    return out
}
```

Standard RK4 step of size `dt`:

```go
func rk4Step(psi []complex128, H [][]complex128, dt float64) []complex128 {
    k1 := schrodingerDeriv(psi, H)
    // k2, k3, k4 with temporary slices ...
    psiNew := make([]complex128, len(psi))
    for i := range psi {
        psiNew[i] = psi[i] + dt/6*(k1[i] + 2*k2[i] + 2*k3[i] + k4[i])
    }
    return psiNew
}
```

Maintain norm: after each step, optionally renormalise if norm deviates beyond `norm_tolerance`. For the audit we will record the deviation.

### 7.2 Bell Master Equation for ρ

ρ is a classical probability distribution (slice of float64 length dim). Equation:

\[
\frac{d\rho_n}{dt} = \sum_{m \neq n} \big( \text{rate}_{n←m}\, \rho_m - \text{rate}_{m←n}\, \rho_n \big)
\]

The rates depend on the current ψ, so they are computed at the same time step as ψ. We can use a simple forward Euler step with the same `dt`. For better stability one could use smaller sub‑steps, but for this toy we keep it simple.

```go
func masterStep(rho []float64, rates [][]float64, dt float64) []float64 {
    newRho := make([]float64, len(rho))
    copy(newRho, rho)
    for n := range rho {
        var influx, outflux float64
        for m := range rho {
            if m == n { continue }
            influx += rates[n][m] * rho[m]
            outflux += rates[m][n] * rho[n]
        }
        newRho[n] += dt * (influx - outflux)
    }
    return newRho
}
```

**Initial ρ**: set ρ = |ψ₀|².

---

## 8. Audit (`internal/audit`)

At each time step (or every few steps) compute:

- **Norm of ψ**: `||ψ||² = Σ |ψ_i|²`. Error = |1 - norm|.
- **ρ non‑negativity**: trivial if we clamp.
- **Rate non‑negativity**: computed rates are non‑negative by construction, but we check `max_negative_rate_violation` = 0.
- **Equivariance L1 error**: `Σ |ρ_i - |ψ_i|²|`. Maximum over all steps.

Track maximum errors across all steps.

After evolution, determine goal status:

- If any of:
  - Hamiltonian not Hermitian (max error > tolerance)
  - Max norm error > norm_tolerance
  - Any negative rate (should be 0, but check)
  - Max equivariance error > equivariance_tolerance
  - State becomes NaN or Inf
- Then: **toy_goal_failed** (if errors are clearly beyond tolerance) or **toy_goal_inconclusive** if instability is suspected (e.g., errors blow up by factor >100 × tolerance, or NaN appears without obvious code bug). A simple rule: if any check fails but the error is within 10× tolerance and looks like round‑off, we could mark inconclusive. For the first version, if any check fails, mark fail; but add a flag `inconclusive` if numerical instability detected (like NaN in ψ). Better: define instability flag when `math.IsNaN(equivErr)` or `equivErr > 1e3*tolerance` and growing.

We will produce a metric `max_equivariance_l1_error`.

---

## 9. Report (`internal/report`)

### 9.1 Output Directory

Create `--out` directory (e.g., `out/bellmipt-run`). Inside write:

- `input.json` – exact copy of the configuration used.
- `report.json` – the JSON report following schema `bell_mipt_report_v0`.
- `report.md` – a Markdown summary containing goal status, metrics, and a plain statement that no physical claim is made.

### 9.2 JSON Report Structure

```go
type Report struct {
    SchemaVersion string `json:"schema_version"`
    ToyID         string `json:"toy_id"`
    ToyAnalysisOnly bool `json:"toy_analysis_only"`
    PhysicsClaim    string `json:"physics_claim"`
    Model           string `json:"model"`
    Sites           int    `json:"sites"`
    HilbertDim      int    `json:"hilbert_dim"`
    Goal            string `json:"goal"`
    GoalStatus      string `json:"goal_status"` // toy_goal_passed/failed/inconclusive
    Checks          struct {
        HamiltonianHermitian             bool `json:"hamiltonian_hermitian"`
        StateNormPreserved               bool `json:"state_norm_preserved"`
        RatesNonnegative                 bool `json:"rates_nonnegative"`
        EquivarianceErrorWithinTolerance bool `json:"equivariance_error_within_tolerance"`
    } `json:"checks"`
    Metrics struct {
        MaxHermitianError        float64 `json:"max_hermitian_error"`
        MaxNormError             float64 `json:"max_norm_error"`
        MaxNegativeRateViolation float64 `json:"max_negative_rate_violation"`
        MaxEquivarianceL1Error   float64 `json:"max_equivariance_l1_error"`
        MeanTotalBellActivity    float64 `json:"mean_total_bell_activity"`
        MaxTotalBellActivity     float64 `json:"max_total_bell_activity"`
    } `json:"metrics"`
    DebtStatus map[string]string `json:"debt_status"`
}
```

Debt status hard‑coded as required:

```go
DebtStatus: map[string]string{
    "needMap":                  "unpaid",
    "needInvariant":            "partially_paid_equivariance_only",
    "needToyCheck":             "partially_paid_rate_algebra_only",
    "needNullModel":            "unpaid",
    "needObstruction":         "bell_jumps_are_not_measurements",
    "needFaithfulnessReview":  "unpaid",
}
```

### 9.3 Bell Activity (optional metric)

Compute total Bell activity per step: sum of all positive rates (or sum of all J_nm for J>0). Track mean and max.

---

## 10. Markdown Summary

A short, honest text without promotional language. Example:

```
# BELL-MIPT-0001 Toy Report

- **Goal**: Compute Bell jump rates and verify equivariance.
- **Status**: toy_goal_passed
- **Model**: finite Kitaev chain, 6 sites, periodic
- **Hermiticity**: OK (max error 0.0)
- **Norm**: preserved (max error 1.2e-15)
- **Non‑negative rates**: OK
- **Equivariance**: max L1 deviation 8.7e-7 within tolerance 1e-5
- **Physics claim**: none. This is a numerical toy analysis only.
```

The Markdown must not contain any blacklisted phrases like “MIPT”, “holography”, “black hole”, “measurement equals jump”, etc. The `language.go` module will scan the generated text and return an error if any forbidden phrase appears.

---

## 11. Main Orchestration (`cmd/bellmipt/main.go`)

```
1. Parse flags `--config` and `--out`.
2. Load config (or use default).
3. Validate config: model must be "finite_kitaev_chain", initial_state.type "random_normalized", boundary "open"/"periodic".
4. Build basis (dim = 2^sites).
5. Generate initial state ψ: for random_normalized, use math/rand with seed to produce complex amplitudes, then normalize.
6. Construct Hamiltonian H (as dense matrix).
7. Compute Hermiticity error; store max.
8. Set ρ_0 = |ψ_0|^2.
9. Prepare slices for ψ, ρ, metrics.
10. Loop for `steps`:
    a. Compute current ψ, then rates.
    b. Step ψ forward using RK4 with dt.
    c. Step ρ forward using master equation (Euler with same dt).
    d. Every step (or every 10 steps, adjustable) record:
        - norm error
        - equivariance L1 error
        - rates non‑negativity (should be always true)
        - total Bell activity
    e. Check for NaN/Inf; if found, set instability flag.
11. After loop, determine goal_status:
    - If hermitianErr <= tol && maxNormErr <= tol && no negative rates && maxL1Err <= tol && no instability -> passed.
    - Else if instability (NaN/Inf or errors > 100*tol) -> inconclusive.
    - Else -> failed.
12. Build report, write JSON and Markdown to out dir.
13. Copy config to output as input.json.
14. Run language check on Markdown; if fails, log error (this is a test, but the program can still exit 0, or fail in test suite).
```

---

## 12. Testing Plan (`*_test.go` files)

### 12.1 Basis Tests
- `TestEnumerateSize`: sites=3 gives 8 configs.
- `TestConfigBits`: verify bit manipulation.

### 12.2 Operator Tests
- `TestJWSign`: for a few configs, manually compute phase.
- `TestAnticommutation`: for 2 sites, verify {c_i, c_j^†} = δ_{i,j} on all basis states (using matrix representation).

### 12.3 Hamiltonian Tests
- `TestHermiticity`: compute H for small chain (N=2,4) and verify max |H - H^†| < 1e-12.
- `TestKnownSpectrum`: for small N (2) with zero Δ, compare eigenvalues to exact free‑fermion energies.

### 12.4 Bell Current Tests
- `TestCurrentAntisymmetry`: for random ψ, check J_nm = -J_mn.
- `TestRateNonnegative`: all computed rates >= 0.

### 12.5 Equivariance Audit
- Use a tiny system (2 sites) with known initial state and short evolution. Compare final ρ and |ψ|²; they should match within tight tolerance (1e-12) for small dt. Run with a few steps and verify L1 error is tiny.
- For a larger system (4 sites) check that equivariance holds over several steps.

### 12.6 Deterministic Output
- Run with fixed seed and config; capture report.json. Run again, ensure identical results (bitwise comparison).

### 12.7 Forbidden Language Test
- `TestNoPromotionLanguage`: Run the program (or just the markdown generation) and scan output for terms like “MIPT”, “holography”, “black hole”, “measurement equals jump”. Fail if found.

### 12.8 Integration Test
- `TestDefaultRun`: execute `go run ./cmd/bellmipt --out /tmp/test-out` and check that report.json exists and contains `"toy_analysis_only": true` and `"physics_claim": "none"`.

---

## 13. Performance Considerations

For N=6, dim=64, dense matrix operations are fast. For larger sizes (N up to 12) the program would become slow due to O(dim^3) RK4 if implemented naively, but that’s out of scope.

The master equation uses `dim × dim` rates, also O(dim^2) per step, fine for 6 sites.

---

## 14. Known Limitations / Debt Tracking

The `debt_status` map in the report explicitly lists the unpaid debts (needMap, needNullModel, etc.) and the paid ones (equivariance, rate algebra). This is part of the report to acknowledge what was not done.

---

This plan provides all the necessary details to implement the toy program faithfully, without over‑engineering. Each step is small and testable. The output structure guarantees that the program will produce the exact required report format and clearly state its limited scope.
