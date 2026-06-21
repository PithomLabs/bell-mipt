# Implementation Plan: BELL-MIPT-0001 (Go Toy Check)

This document provides a complete, step-by-step implementation plan for the `bellmipt` Go program. It is designed to be handed directly to a coding agent. The plan strictly enforces the epistemic boundaries: **no MIPT claims, no holography, no conditional-wave-function bridges.** The sole objective is to compute Bell jump rates for a finite Kitaev chain and verify numerical equivariance.

---

## 1. Project Structure

Create the following directory and file structure:

```text
bellmipt/
├── cmd/
│   └── bellmipt/
│       └── main.go          # CLI entry point, flag parsing, orchestration
├── internal/
│   ├── config/
│   │   └── config.go        # JSON schema definitions, loading, defaults
│   ├── basis/
│   │   ├── basis.go         # Bitstring operations, Hilbert space mapping
│   │   └── basis_test.go
│   ├── operators/
│   │   ├── fermion.go       # Creation/annihilation with Jordan-Wigner signs
│   │   └── fermion_test.go
│   ├── hamiltonian/
│   │   ├── kitaev.go        # Kitaev chain Hamiltonian construction
│   │   └── kitaev_test.go
│   ├── bell/
│   │   ├── rates.go         # Probability currents and Bell jump rates
│   │   └── rates_test.go
│   ├── evolution/
│   │   ├── integrator.go    # RK4 for coupled ψ and ρ evolution
│   │   └── integrator_test.go
│   └── report/
│       ├── report.go        # JSON and Markdown generation
│       └── report_test.go   # Checks for forbidden promotion language
├── go.mod
└── README.md
```

---

## 2. Module Specifications

### 2.1. `config` (Configuration & I/O)
- Define Go structs matching the exact JSON schema provided in the prompt.
- Implement `LoadConfig(path string) (*Config, error)`.
- Implement `DefaultConfig() *Config` returning a 6-site open Kitaev chain with `mu=1.0, t=1.0, delta=0.5`, `dt=0.001`, `steps=1000`.
- **Constraint:** Validate that `model` is `finite_kitaev_chain` and `initial_state` is `random_normalized`.

### 2.2. `basis` (Fock Space & Bitstrings)
- Represent states as `uint64`. The $i$-th bit represents occupation at site $i$.
- `Dim(sites int) int`: Returns $2^{\text{sites}}$.
- `Occupation(state uint64, site int) int`: Returns 0 or 1.
- `JWSign(state uint64, site int) complex128`: Computes the Jordan-Wigner sign $(-1)^{\sum_{j<i} n_j}$. Use `bits.OnesCount64(state & ((1 << site) - 1))`.

### 2.3. `operators` (Fermion Algebra)
- `ApplyAnnihilation(state uint64, site int) (newState uint64, sign complex128)`:
  - If site is empty, return 0, 0.
  - Else, flip bit, compute JW sign, return newState, sign.
- `ApplyCreation(state uint64, site int) (newState uint64, sign complex128)`:
  - If site is occupied, return 0, 0.
  - Else, flip bit, compute JW sign, return newState, sign.
- **Periodic Boundary Trap:** For periodic boundaries, the boundary term $c^\dagger_{N-1} c_0$ requires an extra sign factor of $(-1)^{\hat{N}}$, where $\hat{N}$ is the total fermion number. Multiply the sign by `(-1)^bits.OnesCount64(state)` for these specific boundary terms.

### 2.4. `hamiltonian` (Kitaev Chain)
- `BuildKitaev(cfg *config.Config) [][]complex128`:
  - Initialize a $D \times D$ complex matrix ($D = 2^{\text{sites}}$).
  - Loop over sites $i$. Add terms:
    1. **Chemical potential:** $-\mu c^\dagger_i c_i$
    2. **Hopping:** $-t c^\dagger_i c_{i+1} - t c^\dagger_{i+1} c_i$
    3. **Pairing:** $\Delta c_i c_{i+1} + \Delta^* c^\dagger_{i+1} c^\dagger_i$ (Assume $\Delta$ is real for simplicity, or handle complex conjugate properly).
  - If `boundary == "periodic"`, wrap $i+1$ to $0$ and apply the periodic JW sign fix.
- **Audit:** Check Hermiticity: $|H_{nm} - H_{mn}^*| < \text{hermitian\_tolerance}$.

### 2.5. `bell` (Currents and Rates)
- `ComputeCurrents(psi []complex128, H [][]complex128) [][]float64`:
  - $J_{nm} = 2 \text{Im}(\psi_n^* H_{nm} \psi_m)$.
  - **Audit:** Check antisymmetry: $|J_{nm} + J_{mn}| < \epsilon$.
- `ComputeRates(psi []complex128, J [][]float64) [][]float64`:
  - $\sigma_{n \leftarrow m} = \frac{\max(J_{nm}, 0)}{\max(|\psi_m|^2, 10^{-15})}$.
  - **Audit:** Ensure all rates are $\ge 0$.

### 2.6. `evolution` (Integrators)
- Implement a **Joint RK4** integrator for the coupled system $(\psi, \rho)$.
- State vector: $Y = [\psi, \rho]$.
- Derivative function $F(Y)$:
  1. Compute $d\psi/dt = -i H \psi$.
  2. Compute $J$ and $\sigma$ from current $\psi$.
  3. Compute $d\rho_n/dt = \sum_m (\sigma_{n \leftarrow m} \rho_m - \sigma_{m \leftarrow n} \rho_n)$.
- **Audit:** At each step, compute L1 equivariance error: $E(t) = \sum_n |\rho_n(t) - |\psi_n(t)|^2|$. Track the maximum error. Ensure $\sum \rho_n = 1$ and $\sum |\psi_n|^2 = 1$.

### 2.7. `report` (Output Generation)
- Generate `report.json` strictly matching the schema.
- Generate `report.md` with a human-readable summary.
- **CRITICAL CONSTRAINT:** Implement a `SanitizeText(text string) string` function that panics or strips forbidden words.
  - **Blacklist:** "MIPT", "measurement-induced", "holography", "black hole", "collapse", "observer", "measurement rate", "Zeno", "entanglement transition".
  - **Allowed:** "Bell jump", "equivariance", "Kitaev", "fermion", "stochastic", "master equation", "Schrödinger", "probability current".

---

## 3. CLI & Orchestration (`cmd/bellmipt/main.go`)

1. Parse flags: `--config` (string), `--out` (string).
2. Load config (or use default).
3. Create output directory.
4. Echo `input.json` to the output directory.
5. Initialize basis, build Hamiltonian, generate random normalized initial state (using config seed).
6. Initialize $\rho(0) = |\psi(0)|^2$.
7. Run Joint RK4 evolution.
8. Evaluate pass/fail criteria:
   - `hamiltonian_hermitian`: max Hermitian error < tolerance.
   - `state_norm_preserved`: max norm deviation < tolerance.
   - `rates_nonnegative`: no negative rates found.
   - `equivariance_error_within_tolerance`: max L1 error < tolerance.
9. Write `report.json` and `report.md`.
10. Print final status to stdout.

---

## 4. Testing Strategy

Write unit tests for every internal package. The coding agent must ensure `go test ./...` passes.

- **`basis_test.go`**: Verify bit flipping, occupation counting, and JW signs for known states.
- **`fermion_test.go`**: Verify $c_i |1\rangle = |0\rangle$, $c^\dagger_i |0\rangle = |1\rangle$, and nil for invalid operations.
- **`kitaev_test.go`**: Build a 2-site chain. Verify matrix is Hermitian. Verify trace and eigenvalues match analytical free-fermion results.
- **`rates_test.go`**: Verify $J_{nm} = -J_{mn}$. Verify rates are $\ge 0$. Verify rate is 0 when $J_{nm} < 0$.
- **`integrator_test.go`**: Evolve a 2-site system for 10 steps. Verify $\rho(t)$ matches $|\psi(t)|^2$ within $10^{-5}$.
- **`report_test.go`**: Pass a string containing "MIPT" and "holography" to the sanitizer and verify it panics or removes them.

---

## 5. Execution and Validation Commands

The coding agent must validate the implementation using these exact commands:

```bash
# 1. Run all unit tests
go test ./...

# 2. Run with default config
go run ./cmd/bellmipt --out /tmp/bellmipt-default

# 3. Verify outputs exist
ls -l /tmp/bellmipt-default/input.json
ls -l /tmp/bellmipt-default/report.json
ls -l /tmp/bellmipt-default/report.md

# 4. Check report status (should say "toy_goal_passed")
cat /tmp/bellmipt-default/report.json | grep goal_status
```

---

## 6. Epistemic Guardrails (For the Coding Agent)

**DO NOT:**
- Add any code that calculates entanglement entropy, mutual information, or purity.
- Add any variables or comments containing the words "measurement", "MIPT", "observer", or "collapse" (except in the blacklist checker).
- Attempt to simulate stochastic quantum trajectories (Monte Carlo jumps). The master equation is sufficient for this ticket.
- Optimize for large $N$. This is a toy check for $N=6$. Dense matrices are fine.

**DO:**
- Keep the code simple, readable, and strictly focused on the mathematical verification of Bell's rate law.
- Ensure the JSON output exactly matches the requested schema, including the `debt_status` fields.
- Make the random number generator deterministic based on the `seed` in the config.
