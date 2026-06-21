## Implementation Plan: BELL-MIPT-0002A — Bell Trajectory + Conditional Subsystem State Audit

### Overview
This ticket extends the existing `BELL-MIPT-0001` Go toy to sample actual Bell configuration histories and audit the conditional subsystem wavefunction induced by the environment configuration. It introduces a new optional `bridge` config section. When enabled, the program:
- samples many trajectories of the Bell configuration `Q(t)` according to Bell jump rates,
- at each sampled time step, splits the full wavefunction into subsystem A and environment B,
- constructs the normalized conditional wavefunction `ψ_A(a,t) ∝ Ψ(a, Q_B(t), t)`,
- computes fidelity drops between consecutive conditional states,
- classifies drops by whether an environment jump occurred,
- reports descriptive metrics without claiming a physics breakthrough.

**Key design decisions:**
- Keep the existing command shape (`go run ./cmd/bellmipt --config ... --out ...`).
- Extend `internal/bellmipt` with new files; do not create new subcommands or packages.
- Use a simple **discrete-time jump sampling** (small fixed `dt`) consistent with the existing time stepping; avoid continuous-time Gillespie unless absolutely required.
- Preserve old behavior when `bridge.enabled` is `false` or omitted.
- Produce a separate `bridge_status` alongside `goal_status`; `goal_status` remains unchanged.

---

### File Layout Changes
- **`internal/bellmipt/config.go`**  
  - Add `BridgeConfig` struct with fields: `Enabled`, `SubsystemSites`, `EnvironmentSites`, `Trajectories`, `Seed`, `SampleEverySteps`.
  - Extend `Config` to contain `Bridge *BridgeConfig`.
  - Add validation functions for bridge configuration.

- **`internal/bellmipt/trajectory.go`** (new)  
  - Functions for Bell trajectory sampling:
    - `SampleInitialConfiguration(psi []complex128, rng *rand.Rand) int` (returns index of basis state).
    - `SampleJump(currentIdx int, psi []complex128, rates []float64, totalRate float64, rng *rand.Rand) (newIdx int, jumped bool)`.
    - `EvolveConfiguration(currentIdx int, psi []complex128, dt float64, rng *rand.Rand) (newIdx int, jumped bool)`.
    - `RunSingleTrajectory(...) -> TrajectoryData` containing configuration history and jump flags at sampled times.

- **`internal/bellmipt/conditional.go`** (new)  
  - Subsystem/environment site mapping utilities.
  - `SplitConfiguration(idx int, subsystemSites, envSites []int) (aIdx, bIdx int)`.
  - `ConditionalState(psi []complex128, bIdx int, subsystemSites []int, nSubsystem int) ([]complex128, float64)` returns the conditional vector and its norm.
  - `NormalizeConditional(vec []complex128, norm float64) ([]complex128, bool)` returns normalized vector and success flag (norm > epsilon).
  - `Fidelity(vec1, vec2 []complex128) float64`.

- **`internal/bellmipt/bridge_metrics.go`** (new)  
  - Structures for aggregated metrics per trajectory and across all trajectories.
  - Functions to compute fidelity drops, classify transitions, update running aggregates.
  - `ComputeEmpiricalEquivariance(trajectories []TrajectoryData, psiAtTimes [][]complex128) []float64` for L1 distance.
  - Final metrics aggregation and status determination.

- **`internal/bellmipt/run.go`**  
  - Modify `Run()` or add `RunWithBridge()` that branches based on `cfg.Bridge.Enabled`.
  - Old path unchanged; new path calls trajectory sampling, conditional state audit, and produces combined report.

- **`internal/bellmipt/report.go`**  
  - Extend `Report` struct with `BridgeReport *BridgeReport`.
  - Define `BridgeReport` with all required fields (metrics, interpretation, status).
  - Update JSON marshaling and Markdown generation to include bridge section when present.
  - Ensure backward compatibility: omit bridge fields if disabled.

- **`internal/bellmipt/forbidden.go`**  
  - Add new banned phrases (if any) and ensure bridge outputs are audited.

- **Test files**  
  - `config_test.go`, `trajectory_test.go`, `conditional_test.go`, `bridge_metrics_test.go`, `report_test.go`.
  - Integration test with a small test case to verify old behavior and bridge results.

---

### New Data Structures

```go
// Config additions
type BridgeConfig struct {
    Enabled          bool   `json:"enabled"`
    SubsystemSites   []int  `json:"subsystem_sites"`
    EnvironmentSites []int  `json:"environment_sites"`
    Trajectories     int    `json:"trajectories"`
    Seed             int64  `json:"seed"`
    SampleEverySteps int    `json:"sample_every_steps"`
}

// TrajectoryData: stores history at each sampled step (or full history if sample_every_steps=1)
type TrajectoryData struct {
    ConfigHistory      []int     // configuration index at each sampled time
    JumpHistory        []bool    // whether a jump occurred at that step (global)
    EnvJumpHistory     []bool    // whether an environment jump occurred
    SubsystemJumpHistory []bool  // whether a subsystem jump occurred
    TotalJumpCount     int
    EnvJumpCount       int
    SubsystemJumpCount int
}

// BridgeMetrics aggregates across all trajectories
type BridgeMetrics struct {
    TrajectoryCount                        int
    MeanJumpCount                          float64
    MeanEnvJumpCount                       float64
    MeanSubsystemJumpCount                 float64
    ConditionalNormFailures                int
    MeanFidelityDropAtEnvJumps             float64
    MeanFidelityDropWithoutEnvJumps        float64
    ConditionalUpdateRatio                 float64   // may be NaN or Inf
    MaxEmpiricalEquivarianceL1             float64
    FinalEmpiricalEquivarianceL1           float64
    // Also store per-step aggregate counts for fidelity drop classification
}

// BridgeReport final output
type BridgeReport struct {
    Enabled       bool          `json:"enabled"`
    BridgeGoal    string        `json:"bridge_goal"` // fixed string
    BridgeStatus  string        `json:"bridge_status"`
    Trajectories  int           `json:"trajectories"`
    SubsystemSites []int        `json:"subsystem_sites"`
    EnvironmentSites []int      `json:"environment_sites"`
    SampleEverySteps int        `json:"sample_every_steps"`
    Metrics       BridgeMetrics `json:"metrics"`
    Interpretation struct {
        MonitoringLikeSignal string `json:"monitoring_like_signal"` // "not_assessed", "no_clear_signal", "weak_signal", "candidate_signal"
        Reason               string `json:"reason"`
    } `json:"interpretation"`
}
```

---

### Algorithms (Detailed Pseudocode)

#### 1. Bell Trajectory Sampling
Given:
- `psi`: current wavefunction vector (complex128 slice) of length `N=2^L`
- `dt`: time step size (existing from config)
- `currentIdx`: current basis configuration index
- `rng`: seeded random source

At each time step:
1. Compute Bell rates `sigma(n <- currentIdx)` for all destination configurations `n`. This uses the existing function from `rates.go` (likely `BellJumpRates(psi, currentIdx)` returning a slice of rates).
2. Compute total outgoing rate `lambda = sum(sigma)`.
3. Decide jump:
   - `p_jump = 1 - exp(-lambda * dt)` (for small dt this is approx `lambda*dt`, but exact exponential is safe).
   - Draw `u = rng.Float64()`; if `u < p_jump`, a jump occurs.
4. If jump:
   - Choose destination `n` with probability proportional to `sigma[n] / lambda`. Use `rng` to sample.
   - Set `newIdx = n`, `jumped = true`.
5. Else:
   - `newIdx = currentIdx`, `jumped = false`.
6. Record configuration and jump classification (environment/subsystem jumps based on site partitions).

#### 2. Initial Configuration Sampling
- At `t=0`, draw `N` samples (one per trajectory) from `P(Q_0=q) = |psi_0[q]|^2`.
- Use inverse transform sampling or a standard categorical sampler.

#### 3. Conditional Wavefunction Construction
Given:
- Full wavefunction `psi` of length `2^L`.
- Subsystem sites `A` of size `nA`, environment sites `B` of size `nB = L - nA`.
- A specific environment configuration index `bIdx` (0..2^nB - 1).

Procedure:
- Precompute mapping from full configuration index to subsystem index and environment index (or compute on the fly using bit operations, but precomputing for each trajectory run is fine for toy sizes).
- For each subsystem configuration index `aIdx` (0..2^nA - 1):
   - Compute full index `fullIdx = combine(aIdx, bIdx)` using the site ordering.
   - Get `psi[fullIdx]`.
- The conditional vector `cond` has length `2^nA`.
- Compute norm squared: `norm2 = sum(|cond[i]|^2)`.
- If `norm2 > eps` (e.g., 1e-12), normalize by `1/sqrt(norm2)`; else mark as norm failure.

#### 4. Fidelity Drop Computation
- At each sampled time step (every `sample_every_steps` steps), compute normalized conditional state.
- For the first sample, store it.
- For subsequent samples, compute fidelity `F = |<prev|curr>|^2`. Use Go's complex arithmetic; `dot = cdot(prev, curr)`; `F = real(dot * conj(dot))` (or `cmplx.Abs(dot)**2`).
- Drop = `1 - F`.
- Classify transition according to changes in `Q_A` and `Q_B` from previous sampled step.
- Accumulate drop values into separate running sums/counts for categories:
   - Environment jump (regardless of subsystem jump)
   - No environment jump
   - Any jump (global)
   - No jump (global)

#### 5. Empirical Equivariance
- For each sampled time step, collect configuration frequencies across trajectories.
- Compute L1 distance: `0.5 * sum_q |freq_q/N_traj - |psi_q|^2|` (or sum absolute difference, same factor).
- Track maximum over time and final value.

#### 6. Metrics Aggregation
- After all trajectories, compute means:
   - `MeanJumpCount` = average total jumps per trajectory.
   - `MeanEnvJumpCount`, `MeanSubsystemJumpCount`.
   - `ConditionalNormFailures` = total number of times normalization failed across all trajectories and steps.
   - For fidelity drops: compute means for each category (environment jumps, no environment jumps, any, none).
   - `ConditionalUpdateRatio` = mean drop with env jumps / mean drop without env jumps. If denominator < 1e-15, set as `NaN` or `0` with a flag.
- Determine `bridge_status`:
   - If any sampling failure (e.g., all trajectories have norm failures), set `bridge_toy_failed`.
   - Else if empirical equivariance max L1 is high (e.g., > 0.1) and trajectories are sufficiently many, may set `bridge_toy_inconclusive` due to noise.
   - Else if metrics are computed successfully and no implementation errors, set `bridge_toy_passed`.
- Determine `monitoring_like_signal`:
   - If `conditional_update_ratio` is significantly > 1 (e.g., > 1.1) and statistics are reliable, set `candidate_signal`; else `no_clear_signal`; if not enough env jumps, `not_assessed`.

---

### Config Parsing and Validation
- Extend `LoadConfig` to unmarshal optional `bridge` field.
- Validation rules:
   - If `bridge.Enabled` is true:
      - `Trajectories > 0`
      - `SampleEverySteps >= 1`
      - `SubsystemSites` and `EnvironmentSites` must partition all sites (0..L-1). No overlap, no missing.
      - The combination of sites must be consistent with the total number of sites L (from `LatticeSites`).
   - If `bridge.Enabled` is false or omitted, ignore other bridge fields (maybe still validate but not require).
- Also ensure `dt` and other simulation parameters are valid (existing validation).

---

### Testing Strategy

#### Unit Tests
- **Config**: test validation cases (disabled, enabled valid, overlap, missing, negative trajectories, zero sample_every).
- **Trajectory sampling**:
   - Mock rates/psi to test jump probability and destination selection.
   - Test initial configuration sampling matches distribution for small L.
   - Deterministic seed reproducibility.
- **Conditional**:
   - Split configurations correctly for different partitions.
   - Conditional state construction from known psi.
   - Normalization failure detection.
   - Fidelity computations with known vectors.
- **Bridge metrics**:
   - Accumulate metrics with dummy data, verify mean and ratio handling.
   - Status logic with thresholds (using constants).
- **Report**:
   - Old report unchanged when bridge disabled.
   - Bridge section included and correct when enabled.

#### Integration Tests
- Run the full program with a small L=4 (or 6) system and a known Hamiltonian, with bridge enabled, and verify:
   - Output JSON contains bridge fields.
   - `bridge_status` is set appropriately.
   - No panic.
   - For `bridge.enabled=false`, output matches previous version's format (maybe compare with golden file).

#### Validation Commands
```bash
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default   # old behavior
go run ./cmd/bellmipt --config testdata/bridge_enabled.json --out /tmp/bellmipt-bridge
```

---

### Report Schema Changes
- The existing `Report` struct will embed an optional `*BridgeReport`.
- JSON serialization: omit `bridge` field if nil.
- Markdown output: add a section "Bridge Audit" with metrics in a table, and the interpretation and limitations.

---

### Goal / Bridge Status Logic
- `goal_status` (old) remains unchanged; it only reflects `BELL-MIPT-0001` master-equation audit.
- `bridge_status` computed as described:
   - `bridge_disabled` if `bridge.enabled=false`.
   - `bridge_toy_passed` if:
      - Trajectories sampled without runtime errors.
      - Conditional state norm failures are below a threshold (e.g., < 5% of samples).
      - Empirical equivariance L1 max < 0.2 (loose threshold) – to avoid noise causing failure.
      - All metrics successfully computed.
   - `bridge_toy_failed` if:
      - Any panics or zero trajectories.
      - Most conditional states fail normalization.
      - or if the simulation crashes.
   - `bridge_toy_inconclusive` otherwise (e.g., too few environment jumps, noisy equivariance).

---

### EBP Debt Update
When bridge enabled and runs successfully, update `needMap`, `needInvariant`, `needToyCheck` as specified in the prompt:
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
If bridge disabled, keep previous debt status.

---

### Known Risks and Mitigations
- **Performance**: Sampling many trajectories (e.g., 200) with many time steps could be slow. Use efficient rate computation; keep L small (≤ 6) for toy. If performance is an issue, we can reduce trajectories or steps in tests.
- **Numerical stability**: Conditional states with very small norm may cause fidelity inaccuracies. Use a small epsilon (1e-12) for norm check.
- **RNG reproducibility**: Use a seeded `rand.New(rand.NewSource(seed))` to ensure deterministic results.
- **Rates computation**: Existing rates function may be O(N^2) – for small L it's fine. If L grows, we may need optimization, but not required for toy.
- **Discrete-time jump sampling**: Using exponential probability is exact for Poisson processes with constant rates over dt (rates change with psi, but in small dt it's a good approximation). The plan uses `p_jump = 1-exp(-lambda*dt)`, which is standard for time-dependent rates when dt is small.

---

### Non-Goals (explicitly restated)
- Do not implement MIPT phase diagram, area/volume law, entanglement entropy, mutual information, purity, monitored quantum circuits, projective measurements, Lindblad dynamics, holography, black holes, Lean theorem proving, AI agents, multi-flavor Majorana chains, or physics promotion.
- Do not claim Bell jumps are measurements or that Bell-MIPT bridge is established.
- Do not add subcommands.

---

### Recommendation
**Implement `BELL-MIPT-0002A` as a direct extension of the current command and internal package.**  
- Keep existing files and add new files (`trajectory.go`, `conditional.go`, `bridge_metrics.go`).
- Branch in `run.go` based on config.
- This approach minimizes code duplication and ensures old behavior is preserved.
- The new code will be self-contained and can be easily removed if needed.

---

### Expected Final Implementation Report (for the coding agent)
The agent should later provide:
1. List of files added/modified.
2. Confirmation that old behavior unchanged when bridge disabled.
3. Output of bridge-enabled run with key metrics.
4. Test commands run and results.
5. Known limitations.
6. EBP debt update status.
7. Explicit statement: No MIPT claim, no holography claim, no Bell-jumps-equal-measurements claim, no physics promotion.

---

### Sample JSON Config (for testing)
```json
{
  "lattice_sites": 6,
  "dt": 0.01,
  "total_time": 1.0,
  "initial_state": "product",
  "hamiltonian": "kitaev_chain",
  "mu": 0.0,
  "delta": 1.0,
  "bridge": {
    "enabled": true,
    "subsystem_sites": [0,1,2],
    "environment_sites": [3,4,5],
    "trajectories": 200,
    "seed": 777,
    "sample_every_steps": 1
  }
}
```

This plan provides a clear roadmap for implementing the ticket. The coding agent can now proceed with detailed code changes based on these specifications.
