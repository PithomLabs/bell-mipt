# BELL-MIPT-0001 Implementation Walkthrough

We have successfully implemented and verified the `BELL-MIPT-0001` one-shot Go toy check, which computes Bell jump rates for a finite fermionic lattice model and numerically verifies equivariance.

## Changes Made

- **Report Markdown Generator** (`internal/bellmipt/markdown.go`): Formats the audit report as a clean Markdown document covering status, scope, verification checks, quantitative metrics, debt status, and limitations.
- **Run Orchestrator** (`internal/bellmipt/run.go`): Manages the full execution flow, validating configuration, generating inputs, propagating states, auditing equivariance, scanning for forbidden promotional language, and exporting report files.
- **Main Entry Point** (`cmd/bellmipt/main.go`): A simple CLI wrapping the orchestrator, accepting `--config` and `--out` flags, returning an exit code matching the goal status (0 for passed, 1 for failed, 2 for inconclusive).
- **Report & Integration Tests** (`internal/bellmipt/report_test.go`): Comprehensive tests verifying deterministic execution, correct output structure (debt status and limitations), custom forbidden language filtering, and end-to-end integration.

---

## Verification Results

### 1. Automated Tests

All tests passed successfully (including under the Go race detector):
```bash
/usr/local/go/bin/go test -race ./...
```
```text
ok      github.com/PithomLabs/bell-mipt/internal/bellmipt       1.103s
```

All 37 test cases cover:
- Config loading, validation, and error reporting.
- Fermion basis enumerations and Jordan-Wigner sign computation.
- Hamiltonian construction and Hermiticity tests.
- Antisymmetry of the probability current.
- Nonnegativity of Bell jump rates.
- Norm and rho sum conservation.
- Equivariance (Born distribution matching) on small systems.
- Scan for forbidden promotional language.
- Deterministic simulation outputs with a fixed seed.

### 2. Manual Verification Run

Executing the CLI binary:
```bash
./bin/bellmipt --out out/bellmipt-default
```
```text
Simulation run completed.
Output directory: out/bellmipt-default
Goal Status:      toy_goal_passed
Max Equivariance L1 Error: 3.134990e-11
```

The resulting `report.json` and `report.md` files verify that:
- The simulation completed with `toy_goal_passed`.
- All checks (including forbidden language scanning) are `true`.
- The maximum equivariance L1 error across the 1000 steps is extremely low (~3.13e-11).
- No promotion language was detected, and all required limitations are included.
