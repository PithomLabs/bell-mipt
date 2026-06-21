# Implementation Plan — BELL-MIPT-0002A in Go

This document outlines the implementation plan for `BELL-MIPT-0002A: Bell trajectory sampler + conditional subsystem state audit`. It extends the accepted `BELL-MIPT-0001` codebase.

---

## 1. Goals and Non-Goals

**Goals:**
- Sample discrete-time Bell configuration trajectories.
- Construct conditional subsystem wave functions based on environment configurations.
- Measure fidelity drops in the conditional state induced by environment jumps.
- Track empirical trajectory equivariance against $|\psi(t)|^2$.
- Extend reports with a `bridge` section while preserving `BELL-MIPT-0001` behavior when the bridge is disabled.

**Non-Goals (Strictly Forbidden):**
- MIPT phase diagrams, entanglement entropy, mutual information, purity.
- Monitored circuits, projective measurements, Lindblad dynamics.
- Holography, black holes, Bohhmian mechanics.
- Lean theorem proving, AI agents.
- Any physics promotion or claim that Bell jumps are measurements.

---

## 2. File Layout Changes

We extend the existing `internal/bellmipt` package rather than creating a new one.

**Modified files:**
- `internal/bellmipt/config.go`: Add `BridgeConfig` struct and validation.
- `internal/bellmipt/run.go`: Branch execution to call the bridge runner if enabled.
- `internal/bellmipt/report.go`: Add `Bridge` section to `Report` struct and JSON output.
- `internal/bellmipt/markdown.go`: Add bridge metrics and explicit limitations to Markdown.
- `internal/bellmipt/forbidden.go`: Add any new bridge-specific forbidden phrases if necessary (though existing list is sufficient).

**New files:**
- `internal/bellmipt/bridge.go`: Trajectory sampling logic, RNG management.
- `internal/bellmipt/conditional.go`: Basis splitting, conditional wave function extraction, fidelity calculation.
- `internal/bellmipt/bridge_metrics.go`: Accumulators for jump counts, fidelity drops, and empirical distributions.
- `internal/bellmipt/bridge_run.go`: Orchestrator for the trajectory loop and pre-computation of universal $\psi(t)$ history.
- `internal/bellmipt/bridge_test.go`: Tests for bridge config, sampling, conditional states, and metrics.

---

## 3. New Config Structs

In `internal/bellmipt/config.go`:

```go
type BridgeConfig struct {
    Enabled          bool    `json:"enabled"`
    SubsystemSites   []int   `json:"subsystem_sites"`
    EnvironmentSites []int   `json:"environment_sites"`
    Trajectories     int     `json:"trajectories"`
    Seed             uint64  `json:"seed"`
    SampleEverySteps int     `json:"sample_every_steps"`
}

type Config struct {
    // ... existing fields ...
    Bridge BridgeConfig `json:"bridge"`
}
```

**Validation Rules (in `Config.Validate()`):**
- If `Bridge.Enabled == false`, no further bridge validation is performed.
- `SubsystemSites` and `EnvironmentSites` must form a perfect partition of `0` to `sites-1`. No overlaps, no missing sites.
- `Trajectories > 0`.
- `SampleEverySteps >= 1`.

---

## 4. New Data Structures

In `internal/bellmipt/bridge_metrics.go`:

```go
type BridgeMetrics struct {
    TrajectoryCount                      int     `json:"trajectory_count"`
    MeanJumpCount                        float64 `json:"mean_jump_count"`
    MeanEnvironmentJumpCount             float64 `json:"mean_environment_jump_count"`
    MeanSubsystemJumpCount               float64 `json:"mean_subsystem_jump_count"`
    ConditionalNormFailures              int     `json:"conditional_norm_failures"`
    MeanFidelityDropAtEnvironmentJumps   float64 `json:"mean_fidelity_drop_at_environment_jumps"`
    MeanFidelityDropWithoutEnvJumps      float64 `json:"mean_fidelity_drop_without_environment_jumps"`
    ConditionalUpdateRatio               *float64 `json:"conditional_update_ratio"` // pointer for null
    MaxEmpiricalEquivarianceL1           float64 `json:"max_empirical_equivariance_l1"`
    FinalEmpiricalEquivarianceL1         float64 `json:"final_empirical_equivariance_l1"`
}

type BridgeInterpretation struct {
    MonitoringLikeSignal string `json:"monitoring_like_signal"` // "not_assessed" | "no_clear_signal" | ...
    Reason               string `json:"reason"`
}

type BridgeReport struct {
    Enabled        bool                  `json:"enabled"`
    BridgeGoal     string                `json:"bridge_goal"`
    BridgeStatus   string                `json:"bridge_status"`
    Trajectories   int                   `json:"trajectories"`
    SubsystemSites []int                 `json:"subsystem_sites"`
    EnvironmentSites []int               `json:"environment_sites"`
    SampleEverySteps int                 `json:"sample_every_steps"`
    Metrics        BridgeMetrics         `json:"metrics"`
    Interpretation BridgeInterpretation  `json:"interpretation"`
}
```

---

## 5. Trajectory Sampling Algorithm

To ensure determinism and save compute, the universal wavefunction $\psi(t)$ is evolved once using the existing RK4 integrator, and the history $\psi(t)$ is cached.

For each trajectory $k \in [0, \text{Trajectories})$:
1. **Initialize RNG**: Use `math/rand/v2` with seed = `BridgeConfig.Seed + k`.
2. **Sample Initial State $Q_0$**: Draw from distribution $P(q) = |\psi_0(q)|^2$.
3. **Iterate Steps** $s = 1 \dots \text{Steps}$:
   - Retrieve cached $\psi(t_s)$.
   - Compute outgoing Bell rates $\sigma(n \leftarrow Q_{s-1})$ from $\psi(t_s)$.
   - Calculate total rate $\lambda = \sum_n \sigma(n \leftarrow Q_{s-1})$.
   - Calculate jump probability $p_{\text{jump}} = 1 - \exp(-\lambda \cdot dt)$.
   - Roll RNG $r \in [0, 1)$.
   - If $r < p_{\text{jump}}$:
     - Sample destination $n$ with probability $\sigma(n \leftarrow Q_{s-1}) / \lambda$.
     - Set $Q_s = n$. Record jump type (Environment, Subsystem, or Both).
   - Else: $Q_s = Q_{s-1}$ (No jump).
   - If $s \mod \text{SampleEverySteps} == 0$:
     - Construct conditional state $\psi_A(t_s)$.
     - Compute fidelity drop against previous sampled conditional state.
     - Update metric accumulators based on jump type.
     - Update empirical distribution histogram at time $s$.

---

## 6. Conditional-Wave-Function Construction

In `internal/bellmipt/conditional.go`:

1. **Basis Splitting**: Given a full configuration $q$, mask the bits corresponding to `SubsystemSites` to get $a$, and `EnvironmentSites` to get $b$.
2. **Extraction**: For a fixed environment $b$, iterate over all subsystem configurations $a$. The unnormalized conditional amplitude is $\tilde{\psi}_A(a, t) = \Psi(a, b, t)$.
3. **Normalization**: $N = \sqrt{\sum_a |\tilde{\psi}_A(a, t)|^2}$.
   - If $N < 10^{-12}$, increment `ConditionalNormFailures` and return a zero vector.
   - Else, $\psi_A(a, t) = \tilde{\psi}_A(a, t) / N$.
4. **Fidelity Drop**: Between $t$ and $t-dt$:
   - $F = |\sum_a \psi_A^*(a, t-dt) \psi_A(a, t)|^2$.
   - $\text{drop} = 1 - F$.

---

## 7. Bridge Metrics & Interpretation Logic

- **Conditional Update Ratio**: `MeanFidelityDropAtEnvironmentJumps / MeanFidelityDropWithoutEnvJumps`. If denominator $< 10^{-12}$, set `ConditionalUpdateRatio` to `null` (nil pointer).
- **Empirical Equivariance L1**: At sampled steps, $\sum_q |\text{freq}(q)/\text{Trajectories} - |\psi_q(t)|^2|$.
- **Monitoring Signal Classification**:
  - `no_clear_signal`: Ratio $\in [0.8, 1.2]$.
  - `weak_signal`: Ratio $> 1.2$ or $< 0.8$.
  - `candidate_signal`: Ratio $> 2.0$.
  - `not_assessed`: Insufficient data.
  - *Reason text must explicitly state this is descriptive only and not a measurement claim.*

---

## 8. Goal and Bridge Status Logic

- `goal_status`: Unchanged from `BELL-MIPT-0001`. Evaluates Hermiticity, Norm, Rates, and Master-Equation Equivariance.
- `bridge_status`:
  - `bridge_disabled`: If `Bridge.Enabled == false`.
  - `bridge_toy_passed`: Trajectories sampled, metrics emitted, norm failures $< 10\%$ of samples, empirical L1 reasonable.
  - `bridge_toy_failed`: Implementation error (e.g., invalid partition, NaN in fidelity).
  - `bridge_toy_inconclusive`: Too many norm failures, or zero environment jumps recorded (preventing ratio calculation).

**Debt Status Update:**
If `bridge_toy_passed`:
```json
{
  "needMap": "partially_paid_conditional_state_toy_only",
  "needInvariant": "partially_paid_equivariance_plus_empirical_trajectory_check",
  "needToyCheck": "partially_paid_rate_algebra_and_conditional_state_toy",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_reviewed_for_0001_only"
}
```
Else, retain `BELL-MIPT-0001` debt status.

---

## 9. Required Tests (`bridge_test.go`)

1. **Config Tests**:
   - Omitted bridge defaults to disabled.
   - Disabled bridge preserves old report structure exactly.
   - Partition validation rejects overlaps/missing sites.
   - Trajectories and SampleEverySteps must be positive.
2. **Trajectory Sampling Tests**:
   - Initial config sampling matches $|\psi_0|^2$ within statistical bounds.
   - Zero total rate guarantees no jump.
   - Destination sampling proportional to rates (mock rates, assert frequencies).
   - Deterministic output for fixed seed.
3. **Conditional Wave Function Tests**:
   - Correct basis splitting (mask logic).
   - Dimension is $2^{|A|}$.
   - Normalization yields unit norm.
   - Norm failure threshold triggers correctly.
   - Fidelity of identical states is 1.0; drop is in $[0, 1]$.
4. **Bridge Metrics Tests**:
   - Environment vs Subsystem jump classification.
   - `conditional_update_ratio` handles zero denominator (returns null).
   - Empirical L1 computation.
   - Status logic maps to `passed`/`failed`/`inconclusive` correctly.
5. **Report Tests**:
   - Old fields backward compatible.
   - Forbidden language audit passes on new markdown.

---

## 10. Required Limitations (Markdown Output)

The `report.md` must explicitly include:
```text
## Limitations
- This samples Bell configuration histories in a finite toy model only.
- This conditional-state audit is not a monitored quantum trajectory simulation.
- This does not implement MIPT.
- This does not show Bell jumps are measurements.
- This does not establish a Bell-MIPT bridge.
- This does not support any holography or black-hole claim.
- This is not a physics promotion.
```

---

## 11. Validation Commands

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge
```

---

## 12. Known Risks

1. **Compute Time**: Running 200 trajectories over 1000 steps with sparse rate calculations per step could be slow.
   - *Mitigation*: Pre-compute $\psi(t)$ history once. Pre-compute outgoing rates for all states at each time step once per time step, then reuse across trajectories.
2. **Categorical Sampling Zero-Division**: If $p_{\text{jump}}$ is non-zero due to float rounding but $\lambda$ evaluates to 0.
   - *Mitigation*: Guard against $\lambda < 10^{-15}$ before sampling destination.
3. **RNG Concurrency**: If trajectories are parallelized, RNG states must be isolated.
   - *Mitigation*: Use `rand.New(rand.NewPCG(seed, k))` per trajectory. Run sequentially in v0 for strict determinism.

---

## 13. Recommendation: Direct Extension vs. Separate Package

**Recommendation: Direct extension of the existing `internal/bellmipt` package.**

**Rationale**: The bridge functionality heavily relies on the internal details of `BELL-MIPT-0001` (specifically, basis bitstring manipulations, sparse Hamiltonian lookups, and Bell rate calculations). Moving this to a separate package (e.g., `internal/bellbridge`) would require either exporting low-level internal functions (violating encapsulation) or duplicating logic. Adding the files directly into `internal/bellmipt` keeps the dependency graph flat and aligns with the v0 architecture directive to "extend this code, not rewrite it."
