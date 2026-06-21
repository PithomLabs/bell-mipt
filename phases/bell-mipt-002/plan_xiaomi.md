# BELL-MIPT-0002A: Implementation Plan

## Bell Trajectory Sampler + Conditional Subsystem State Audit

---

## 0. Executive Summary

Extend the existing `bellmipt` single-command Go program with an optional `bridge` configuration block. When enabled, the program samples Bell configuration trajectories using discrete-time rejection, splits each configuration into subsystem/environment parts, constructs the conditional subsystem wave function from the environment branch, computes fidelity-based update metrics between consecutive conditional states, and performs an empirical trajectory equivariance check. When disabled, all existing `BELL-MIPT-0001` behavior is preserved byte-for-byte.

---

## 1. File Layout Changes

### Files to modify

```
internal/bellmipt/config.go          ← add BridgeConfig struct + validation
internal/bellmipt/run.go             ← add bridge orchestration path
internal/bellmipt/report.go          ← add bridge JSON section
internal/bellmipt/markdown.go        ← add bridge markdown section
internal/bellmipt/forbidden.go       ← extend forbidden-string list if needed
cmd/bellmipt/main.go                 ← no change expected (config already drives behavior)
```

### Files to add

```
internal/bellmipt/trajectory.go      ← Bell trajectory sampler
internal/bellmipt/conditional.go     ← conditional subsystem wave function
internal/bellmipt/bridgemetrics.go   ← bridge metric aggregation
internal/bellmipt/bridgemetrics_test.go
internal/bellmipt/trajectory_test.go
internal/bellmipt/conditional_test.go
internal/bellmipt/bridge_integration_test.go
```

### Files unchanged

```
internal/bellmipt/basis.go
internal/bellmipt/fermion.go
internal/bellmipt/hamiltonian.go
internal/bellmipt/current.go         ← called by trajectory sampler but not modified
internal/bellmipt/rates.go           ← called by trajectory sampler but not modified
internal/bellmipt/evolve.go          ← ψ evolution unchanged
internal/bellmipt/audit.go           ← master-equation equivariance unchanged
```

All new code lives under `internal/bellmipt`. No new top-level packages. No `internal/bridge` sub-package — the bridge logic is a consumer of existing basis/hamiltonian/current/rates primitives and does not warrant its own package.

---

## 2. Config Structs

### New type in `config.go`

```go
type BridgeConfig struct {
    Enabled           bool    `json:"enabled"`
    SubsystemSites    []int   `json:"subsystem_sites"`
    EnvironmentSites  []int   `json:"environment_sites"`
    Trajectories      int     `json:"trajectories"`
    Seed              int64   `json:"seed"`
    SampleEverySteps  int     `json:"sample_every_steps"`
}
```

### Embedded in existing config

Add field to existing `Config` struct:

```go
type Config struct {
    // ... existing fields ...
    Bridge *BridgeConfig `json:"bridge"`
}
```

Pointer type so `nil` means omitted. When `nil`, treat as disabled.

### Default config behavior

- If `bridge` key is absent or `bridge.enabled` is `false`, the entire bridge path is skipped. `run.go` calls the existing `BELL-MIPT-0001` code path unchanged. Report output is identical to `BELL-MIPT-0001`.
- Default built-in config (when `--config` omitted) has `bridge: nil`.

### Validation rules (in `config.go`, function `ValidateBridge`)

| Rule | Error |
|---|---|
| `Enabled == true` but `SubsystemSites` empty | `"subsystem_sites must not be empty when bridge is enabled"` |
| `Enabled == true` but `EnvironmentSites` empty | `"environment_sites must not be empty when bridge is enabled"` |
| Any site in `SubsystemSites` not in `[0, nSites)` | `"subsystem site out of range"` |
| Any site in `EnvironmentSites` not in `[0, nSites)` | `"environment site out of range"` |
| Overlap between subsystem and environment | `"subsystem and environment sites must not overlap"` |
| Union does not cover all sites | `"subsystem and environment sites must partition all sites"` |
| `Trajectories <= 0` when enabled | `"trajectories must be positive when bridge is enabled"` |
| `SampleEverySteps < 1` when enabled | `"sample_every_steps must be >= 1"` |

The existing `Config.Validate()` for the base fields is untouched. `ValidateBridge` is called only when `Bridge != nil && Bridge.Enabled`.

---

## 3. New Data Structures

### 3a. Configuration representation

A full basis configuration is already `uint64` (from `basis.go`). Split into subsystem and environment parts using bit masks:

```go
type SitePartition struct {
    SubsystemMask   uint64   // bits set at subsystem site indices
    EnvironmentMask uint64   // bits set at environment site indices
    SubsystemBits   []int    // sorted site indices
    EnvironmentBits []int    // sorted site indices
    SubsystemDim    int      // 2^len(SubsystemBits)
    EnvironmentDim  int      // 2^len(EnvironmentBits)
}
```

Constructed once from `BridgeConfig` at run start.

Helper functions:

```go
func SplitConfig(q uint64, p SitePartition) (subsysPart, envPart uint64)
func SubsystemIndex(q uint64, p SitePartition) int    // compact index into 2^|A| space
func EnvironmentIndex(q uint64, p SitePartition) int  // compact index into 2^|B| space
```

`SubsystemIndex` extracts the subsystem bits from `q` and re-indexes them into a contiguous `[0, 2^|A|)` integer. Same for environment. This is done by iterating the sorted bit positions and shifting.

### 3b. Trajectory record

```go
type TrajectoryStep struct {
    TimeIndex      int
    Config         uint64     // Q(t)
    SubsystemPart  uint64     // Q_A(t)
    EnvironmentPart uint64    // Q_B(t)
    Jumped         bool       // any jump occurred
    EnvJumped      bool       // environment part changed
    SubsysJumped   bool       // subsystem part changed
}

type Trajectory struct {
    Steps []TrajectoryStep
    JumpCount           int
    EnvironmentJumpCount int
    SubsystemJumpCount   int
}
```

### 3c. Conditional state snapshot

```go
type ConditionalSnapshot struct {
    TimeIndex      int
    EnvConfig      uint64         // actual B(t) value
    State          []complex128   // length 2^|A|, normalized
    Norm           float64        // norm before normalization
    NormFailure    bool           // true if norm < threshold
}
```

### 3d. Bridge metrics accumulator

```go
type BridgeMetrics struct {
    TrajectoryCount                 int
    TotalJumps                      int
    TotalEnvironmentJumps           int
    TotalSubsystemJumps             int
    ConditionalNormFailures         int
    
    // Fidelity drops, binned by transition type
    FidelityDropSumEnvJump         float64
    FidelityDropCountEnvJump       int
    FidelityDropSumNoEnvJump       float64
    FidelityDropCountNoEnvJump     int
    FidelityDropSumAnyJump         float64
    FidelityDropCountAnyJump       int
    FidelityDropSumNoJump          float64
    FidelityDropCountNoJump        int
    
    // Empirical equivariance
    EmpiricalDist                   []float64   // accumulated frequency per basis state
    EmpiricalSampleCount            int         // total samples accumulated
    MaxEmpiricalEquivarianceL1      float64
    FinalEmpiricalEquivarianceL1    float64
}
```

---

## 4. Trajectory Sampling Algorithm

### Location: `internal/bellmipt/trajectory.go`

#### 4a. Initial configuration sampling

```
func SampleInitialConfig(rng *rand.Rand, psi []complex128) uint64
```

Algorithm:

1. Compute cumulative distribution: `cdf[i] = sum_{j<=i} |psi[j]|^2`.
2. Draw `u ~ Uniform(0,1)` from the seeded RNG.
3. Binary-search `cdf` to find the smallest `i` with `cdf[i] >= u`.
4. Return `basis[i]`.

This is standard inverse-CDF sampling. Validate that `|psi|^2` sums to 1 within `norm_tolerance` before sampling; if not, return an error.

#### 4b. One trajectory step

```
func TrajectoryStep(
    currentConfig uint64,
    psi []complex128,
    H [][]complex128,
    basis []uint64,
    dt float64,
    rng *rand.Rand,
    partition SitePartition,
) (nextConfig uint64, jumped, envJumped, subsysJumped bool)
```

Algorithm:

1. Compute Bell current `J[n][m]` for all `n` given `m = currentConfig` index.
   - Use existing `current.go` function. Only the column corresponding to `currentConfig` is needed, but the existing implementation likely computes the full matrix. Call it and extract the relevant column. If performance matters later, optimize to single-column — but correctness first.
2. Compute rates `sigma(n <- q)` for all destination states `n`:
   - `sigma = max(J[n][q], 0) / |psi[q]|^2`
   - Use existing `rates.go` function.
3. Compute total outgoing rate: `lambda = sum_n sigma(n <- q)`.
4. Compute jump probability: `p_jump = 1 - exp(-lambda * dt)`.
   - For small `lambda * dt`, this is approximately `lambda * dt`. Use the exact formula for correctness.
5. Draw `u ~ Uniform(0,1)`.
6. If `u < p_jump`:
   - Draw `v ~ Uniform(0,1)`.
   - Choose destination `n` by scanning `sigma(n <- q) / lambda` cumulatively.
   - `nextConfig = basis[n]`.
   - Set `jumped = true`.
   - Compare subsystem/environment parts of old and new config to set `envJumped`, `subsysJumped`.
7. Else:
   - `nextConfig = currentConfig`.
   - All jump flags `= false`.

#### 4c. Full trajectory

```
func RunTrajectory(
    rng *rand.Rand,
    psi0 []complex128,
    basis []uint64,
    H [][]complex128,
    timeSteps int,
    dt float64,
    sampleEvery int,
    partition SitePartition,
) (Trajectory, []ConditionalSnapshot, error)
```

Algorithm:

1. Sample `Q_0` from `|psi_0|^2`.
2. Initialize `rho` from `psi0` (copy master-equation initial state, or recompute from evolved `psi`).
3. For each time step `k = 0 .. timeSteps-1`:
   a. Evolve `psi` by one RK4 step (reusing existing `evolve.go`). The trajectory shares the same universal `psi(t)` — it does not branch.
   b. If `k % sampleEvery == 0`:
      - Record `TrajectoryStep` (config, parts, jump flags).
      - Compute conditional snapshot (see Section 5).
      - Compute fidelity drop against previous snapshot (see Section 6).
   c. Sample next config from current config using the algorithm in 4b, using the current `psi(t)` for rates.
4. Return the trajectory record and conditional snapshots.

**Critical design point:** The universal wave function `psi(t)` is the same for all trajectories. It is evolved once at the run level and shared. Each trajectory only samples the stochastic configuration path `Q(t)`. This is correct because Bell dynamics has the universal wave function evolving by Schrödinger's equation independently of the actual configuration.

#### 4d. Multi-trajectory orchestrator

```
func RunBridge(
    cfg BridgeConfig,
    basis []uint64,
    H [][]complex128,
    psi0 []complex128,
    timeSteps int,
    dt float64,
    partition SitePartition,
) (BridgeMetrics, []Trajectory, error)
```

Algorithm:

1. Create master RNG seeded from `cfg.Seed`.
2. Evolve `psi` for all time steps and store snapshots of `psi` at each step (needed for rate computation during trajectory stepping). Alternatively, recompute from `psi0` — but storing is simpler given small systems.
3. For each trajectory `i = 0 .. cfg.Trajectories-1`:
   a. Create a child RNG seeded from the master (e.g., `master.Int63()` as seed).
   b. Run `RunTrajectory` with this child RNG and the pre-computed `psi` history.
   c. Accumulate trajectory-level metrics into `BridgeMetrics`.
4. After all trajectories, compute derived metrics (ratios, means).
5. Return.

**Note on psi history storage:** For 6 sites, Hilbert dimension is 64 and time steps are 1000. Storing `1000 × 64 × 16 bytes = ~1 MB`. Completely fine. Store as `[][]complex128` — a slice of psi snapshots indexed by time step.

---

## 5. Conditional Wave Function Construction

### Location: `internal/bellmipt/conditional.go`

```
func ComputeConditionalState(
    psi []complex128,
    basis []uint64,
    envConfig uint64,
    partition SitePartition,
    normThreshold float64,
) ConditionalSnapshot
```

Algorithm:

1. Initialize `condState` as `[]complex128` of length `SubsystemDim = 2^|A|`, all zeros.
2. For each full basis state `q = basis[i]`:
   a. Extract environment part: `b = SplitConfig(q, partition).envPart`.
   b. If `b != envConfig`, skip.
   c. Extract subsystem index: `a = SubsystemIndex(q, partition)`.
   d. `condState[a] += psi[i]`.
3. Compute `norm = sqrt(sum |condState[a]|^2)`.
4. If `norm < normThreshold` (use `1e-12`):
   - Set `NormFailure = true`, return with unnormalized (zero) state.
5. Else normalize: `condState[a] /= norm` for all `a`.
6. Return `ConditionalSnapshot` with the normalized state, norm, and env config.

**Index mapping detail:** Given sorted subsystem site indices `[s0, s1, ..., s_{k-1}]`, a full configuration `q` has subsystem bits at positions `s0, s1, ...`. The subsystem compact index is:

```
a = bit(q, s0) * 2^0 + bit(q, s1) * 2^1 + ... + bit(q, s_{k-1}) * 2^{k-1}
```

where `bit(q, s)` is 1 if bit `s` is set in `q`, else 0. This maps all `2^|A|` subsystem configurations to contiguous indices `[0, 2^|A|)`.

Same logic for environment compact index.

### Fidelity computation

```
func ConditionalFidelity(a, b []complex128) float64
```

```
inner = sum_i conj(a[i]) * b[i]
F = |inner|^2
```

Returns value in `[0, 1]`. If either vector has wrong length, return error.

---

## 6. Bridge Metrics Aggregation

### Location: `internal/bellmipt/bridgemetrics.go`

```
func (m *BridgeMetrics) RecordStep(
    oldSnap, newSnap ConditionalSnapshot,
    jumped, envJumped, subsysJumped bool,
)
```

1. Increment jump counters as appropriate.
2. Compute fidelity: `F = ConditionalFidelity(oldSnap.State, newSnap.State)`.
3. Compute drop: `drop = 1.0 - F`.
4. Classify transition and add drop to the appropriate sum/count bucket.

```
func (m *BridgeMetrics) RecordEmpiricalSample(configIndex int, prob []float64)
```

1. Increment `EmpiricalDist[configIndex]` by 1.
2. Increment `EmpiricalSampleCount`.
3. If this is a checkpoint step, compute `empirical_equivariance_l1 = sum_i |EmpiricalDist[i]/EmpiricalSampleCount - prob[i]|` and update `MaxEmpiricalEquivarianceL1` and `FinalEmpiricalEquivarianceL1`.

```
func (m *BridgeMetrics) Finalize() BridgeReportMetrics
```

Compute derived quantities:

```
MeanJumpCount           = TotalJumps / TrajectoryCount
MeanEnvJumpCount        = TotalEnvironmentJumps / TrajectoryCount
MeanSubsysJumpCount     = TotalSubsystemJumps / TrajectoryCount
MeanFidelityDropAtEnvJump   = FidelityDropSumEnvJump / FidelityDropCountEnvJump   (or null)
MeanFidelityDropWithoutEnvJump = FidelityDropSumNoEnvJump / FidelityDropCountNoEnvJump (or null)
MeanFidelityDropAtAnyJump  = FidelityDropSumAnyJump / FidelityDropCountAnyJump    (or null)
MeanFidelityDropNoJump     = FidelityDropSumNoJump / FidelityDropCountNoJump      (or null)
ConditionalUpdateRatio = MeanFidelityDropAtEnvJump / MeanFidelityDropWithoutEnvJump (or null)
```

Handle division by zero: if count is 0 or denominator is < `1e-15`, emit `null` in JSON (use `*float64` pointer in Go struct).

---

## 7. Report Schema Changes

### Location: `internal/bellmipt/report.go`

Add to existing report struct:

```go
type Report struct {
    // ... all existing BELL-MIPT-0001 fields unchanged ...
    Bridge *BridgeReport `json:"bridge,omitempty"`
}

type BridgeReport struct {
    Enabled       bool                   `json:"enabled"`
    BridgeGoal    string                 `json:"bridge_goal"`
    BridgeStatus  string                 `json:"bridge_status"`
    Trajectories  int                    `json:"trajectories"`
    SubsystemSites   []int               `json:"subsystem_sites"`
    EnvironmentSites []int               `json:"environment_sites"`
    SampleEverySteps int                 `json:"sample_every_steps"`
    Metrics       BridgeReportMetrics    `json:"metrics"`
    Interpretation BridgeInterpretation  `json:"interpretation"`
    Limitations   []string              `json:"limitations"`
}

type BridgeReportMetrics struct {
    TrajectoryCount                     int      `json:"trajectory_count"`
    MeanJumpCount                       float64  `json:"mean_jump_count"`
    MeanEnvironmentJumpCount            float64  `json:"mean_environment_jump_count"`
    MeanSubsystemJumpCount              float64  `json:"mean_subsystem_jump_count"`
    ConditionalNormFailures             int      `json:"conditional_norm_failures"`
    MeanFidelityDropAtEnvJumps          *float64 `json:"mean_fidelity_drop_at_environment_jumps"`
    MeanFidelityDropWithoutEnvJumps     *float64 `json:"mean_fidelity_drop_without_environment_jumps"`
    MeanFidelityDropAtAnyJumps          *float64 `json:"mean_fidelity_drop_at_any_jumps"`
    MeanFidelityDropNoJump              *float64 `json:"mean_fidelity_drop_no_jump"`
    ConditionalUpdateRatio              *float64 `json:"conditional_update_ratio"`
    MaxEmpiricalEquivarianceL1          float64  `json:"max_empirical_equivariance_l1"`
    FinalEmpiricalEquivarianceL1        float64  `json:"final_empirical_equivariance_l1"`
}

type BridgeInterpretation struct {
    MonitoringLikeSignal string `json:"monitoring_like_signal"`
    Reason               string `json:"reason"`
}
```

### Monitoring-like signal classification

In `bridgemetrics.go`, after `Finalize()`:

```
func ClassifySignal(metrics BridgeReportMetrics) BridgeInterpretation
```

| Condition | Classification |
|---|---|
| `ConditionalUpdateRatio == nil` | `"not_assessed"` — too few events to compare |
| `ConditionalUpdateRatio < 1.5` | `"no_clear_signal"` — environment jumps do not produce meaningfully larger fidelity drops than smooth evolution |
| `1.5 <= ConditionalUpdateRatio < 3.0` | `"weak_signal"` — some elevated response to environment jumps |
| `ConditionalUpdateRatio >= 3.0` | `"candidate_signal"` — environment jumps produce notably larger conditional-state changes |

These thresholds are descriptive heuristics for a finite toy model. They are not physics claims. The `reason` field should state the ratio value and the classification rationale factually.

### Debt status in report

When bridge is enabled and runs successfully:

```json
{
  "debt_status": {
    "needMap": "partially_paid_conditional_state_toy_only",
    "needInvariant": "partially_paid_equivariance_plus_empirical_trajectory_check",
    "needToyCheck": "partially_paid_rate_algebra_and_conditional_state_toy",
    "needNullModel": "unpaid",
    "needObstruction": "bell_jumps_are_not_measurements",
    "needFaithfulnessReview": "source_code_reviewed_for_0001_only"
  }
}
```

When bridge is disabled, retain original `BELL-MIPT-0001` debt status unchanged.

### Markdown report additions

In `internal/bellmipt/markdown.go`, when bridge is enabled, append a section:

```markdown
## Bridge: Bell Trajectory + Conditional Subsystem State

**Bridge Status:** bridge_toy_passed

### Configuration
- Subsystem sites: [0, 1, 2]
- Environment sites: [3, 4, 5]
- Trajectories: 200
- Sample every: 1 step(s)

### Trajectory Metrics
- Mean total jumps per trajectory: X.XX
- Mean environment jumps: X.XX
- Mean subsystem jumps: X.XX
- Conditional norm failures: 0

### Fidelity Analysis
- Mean fidelity drop at environment jumps: X.XXXX
- Mean fidelity drop without environment jumps: X.XXXX
- Conditional update ratio: X.XX

### Empirical Equivariance
- Max L1 error: X.XXe-XX
- Final L1 error: X.XXe-XX

### Interpretation
- Monitoring-like signal: no_clear_signal
- Reason: Conditional update ratio of X.XX indicates environment jumps do not produce meaningfully larger conditional-state changes than smooth evolution in this finite toy.

### Limitations
- This samples Bell configuration histories in a finite toy model only.
- This conditional-state audit is not a monitored quantum trajectory simulation.
- This does not implement MIPT.
- This does not show Bell jumps are measurements.
- This does not establish a Bell-MIPT bridge.
- This does not support any holography or black-hole claim.
- This is not a physics promotion.
```

### Forbidden language audit

Extend `forbidden.go` scan to also cover the bridge section of the report. The forbidden strings remain:

- "MIPT" (except in toy_id / schema_version which are identifiers)
- "holography"
- "black hole"
- "area law"
- "volume law"
- "entanglement entropy"
- "monitored" (except in the literal limitation sentence "not a monitored quantum trajectory simulation")
- "measurement" (except in the literal "bell_jumps_are_not_measurements" debt string and the limitation sentence)

The `forbidden.go` function should have an allowlist for these specific literal exceptions within the bridge limitations.

---

## 8. Goal / Bridge Status Logic

### Existing `goal_status` (unchanged logic)

Reports master-equation equivariance audit from `BELL-MIPT-0001`. Computed by `audit.go` as before. Independent of bridge.

### New `bridge_status` (in `run.go`)

```
if bridge == nil || !bridge.Enabled:
    bridge_status = "bridge_disabled"
else:
    run bridge
    if any fatal error (config validation, RNG failure, allocation):
        bridge_status = "bridge_toy_failed"
    elif ConditionalNormFailures > 0.5 * TrajectoryCount * SampleCount:
        bridge_status = "bridge_toy_inconclusive"
        reason = "too many conditional norm failures"
    elif TotalEnvironmentJumps < 10 across all trajectories:
        bridge_status = "bridge_toy_inconclusive"
        reason = "too few environment jumps to assess"
    elif any NaN/Inf in metrics:
        bridge_status = "bridge_toy_inconclusive"
        reason = "numerical instability"
    else:
        bridge_status = "bridge_toy_passed"
```

The thresholds (50% norm failure rate, 10 minimum environment jumps) are conservative for a 6-site system with 200 trajectories. If the system is too small or parameters produce negligible jump rates, the inconclusive path fires honestly rather than fabricating a signal.

---

## 9. Run Orchestration Changes

### Location: `internal/bellmipt/run.go`

Current `Run(cfg)` function does:

```
1. Build basis
2. Build Hamiltonian
3. Audit Hermiticity
4. Evolve ψ and ρ simultaneously
5. Audit equivariance
6. Write reports
```

New flow:

```
1. Build basis
2. Build Hamiltonian
3. Audit Hermiticity
4. Evolve ψ and ρ simultaneously
   (store psi history: snapshot of ψ at each time step)
5. Audit equivariance
6. If bridge != nil && bridge.Enabled:
   a. Validate bridge config
   b. Build SitePartition
   c. Run bridge (multi-trajectory)
   d. Classify signal
   e. Determine bridge_status
7. Write reports (including bridge section if applicable)
```

The `psi` history (step 4) is stored as `[][]complex128` of length `timeSteps+1`, each of length `hilbertDim`. For 6 sites / 1000 steps this is ~1 MB. Acceptable.

The ρ master-equation evolution (existing audit path) continues independently. The bridge does not interfere with it.

---

## 10. Test Plan

### 10a. Config tests (`config_test.go` extension)

| Test | Input | Expected |
|---|---|---|
| Bridge omitted | JSON without `bridge` key | `Bridge == nil`, validation passes |
| Bridge disabled | `"bridge": {"enabled": false}` | `Bridge.Enabled == false`, validation passes |
| Valid partition | `"subsystem_sites":[0,1,2], "environment_sites":[3,4,5]` on 6 sites | Validation passes |
| Overlapping sites | `"subsystem_sites":[0,1,2], "environment_sites":[2,3,4]` | Error: overlap |
| Missing site | `"subsystem_sites":[0,1], "environment_sites":[3,4,5]` on 6 sites | Error: site 2 missing |
| Out-of-range site | `"subsystem_sites":[0,1,9]` on 6 sites | Error: out of range |
| Zero trajectories | `"trajectories": 0` | Error: must be positive |
| Zero sample_every | `"sample_every_steps": 0` | Error: must be >= 1 |

### 10b. Trajectory sampling tests (`trajectory_test.go`)

| Test | Description |
|---|---|
| Initial config distribution | Sample 10000 initial configs from a known `psi0`. Verify empirical distribution matches `|psi0|^2` within statistical tolerance (chi-squared or L1 < 0.05). |
| Jump probability correctness | For a known state with known rates, verify that jump frequency over 10000 trials matches `1 - exp(-lambda * dt)` within tolerance. |
| Destination proportionality | Conditional on a jump occurring, verify destination frequencies match `sigma(n <- q) / lambda` within tolerance. |
| Zero-rate no-jump | Set up a state where `lambda = 0` (e.g., eigenstate of H). Verify no jumps occur in 1000 steps. |
| Deterministic output | Run same config+seed twice, verify identical trajectory records (bitwise). |

### 10c. Conditional wave function tests (`conditional_test.go`)

| Test | Description |
|---|---|
| Correct split | For a 4-site system split [0,1]/[2,3], verify that `SplitConfig(0b1101, partition)` gives subsys=`0b01`, env=`0b11`. |
| Subsystem index | Verify `SubsystemIndex` maps all `2^|A|` subsystem patterns to `[0, 2^|A|)` uniquely. |
| Conditional state dimension | Verify conditional state vector has length `2^|A|`. |
| Normalization | Verify `sum |condState[a]|^2 = 1` after normalization within tolerance. |
| Norm failure | Create a state `psi` where no basis state has the given env config. Verify `NormFailure == true`. |
| Fidelity of identical states | `ConditionalFidelity(v, v) == 1.0` within tolerance. |
| Fidelity of orthogonal states | `ConditionalFidelity([1,0], [0,1]) == 0.0` within tolerance. |
| Fidelity drop range | For random normalized vectors, verify `drop = 1 - F` is in `[0, 1]`. |

### 10d. Bridge metrics tests (`bridgemetrics_test.go`)

| Test | Description |
|---|---|
| Jump counting | Manually feed steps with known jump flags, verify counters match. |
| Fidelity drop binning | Feed steps with known transition types, verify drops land in correct buckets. |
| Zero denominator ratio | With zero environment jumps, verify `ConditionalUpdateRatio` is `nil`. |
| Empirical equivariance | For a known distribution, verify L1 computation against a hand-calculated value. |
| Status: passed | Run a healthy bridge, verify `bridge_toy_passed`. |
| Status: inconclusive (norm failures) | Force > 50% norm failures, verify `bridge_toy_inconclusive`. |
| Status: inconclusive (few jumps) | Use eigenstate (zero rates), verify `bridge_toy_inconclusive`. |

### 10e. Integration tests (`bridge_integration_test.go`)

| Test | Description |
|---|---|
| Bridge disabled backward compat | Run with default config (bridge disabled). Verify report JSON is byte-identical to `BELL-MIPT-0001` output (modulo timestamps if any). |
| Bridge enabled full run | Run with 4-site system, bridge enabled, 50 trajectories. Verify report contains bridge section, status is non-empty, metrics are finite numbers. |
| Report backward compatibility | Verify all `BELL-MIPT-0001` fields present when bridge is enabled. |
| Forbidden language | Scan full report JSON+MD for forbidden strings. Verify audit passes. |
| Limitations present | Verify limitations array in bridge report contains all 6 required limitation sentences. |
| Deterministic output | Same config+seed twice → byte-identical `report.json`. |

---

## 11. Validation Commands

The implementation plan requires the following commands to pass before acceptance:

```bash
# Unit tests
go test ./...

# Race detector
go test -race ./...

# Static analysis
go vet ./...

# Default run (bridge disabled — backward compatible)
go run ./cmd/bellmipt --out /tmp/bellmipt-default

# Bridge-enabled run
# (requires a bellmipt_bridge.json with bridge.enabled=true)
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge

# Verify bridge-disabled report matches BELL-MIPT-0001 output
diff <(jq 'del(.bridge)' /tmp/bellmipt-default/report.json) <(expected_0001_report.json)

# Verify bridge report contains required fields
jq '.bridge.bridge_status' /tmp/bellmipt-bridge/report.json
jq '.bridge.metrics.trajectory_count' /tmp/bellmipt-bridge/report.json
jq '.bridge.interpretation.monitoring_like_signal' /tmp/bellmipt-bridge/report.json
```

---

## 12. EBP Debt Updates

### If bridge is disabled

Debt status unchanged from `BELL-MIPT-0001`:

```json
{
  "needMap": "unpaid",
  "needInvariant": "partially_paid_equivariance_only",
  "needToyCheck": "partially_paid_rate_algebra_only",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "source_code_reviewed"
}
```

### If bridge is enabled and runs successfully

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

**Rationale for changes:**

- `needMap`: Moves from `"unpaid"` to `"partially_paid_conditional_state_toy_only"` because the conditional subsystem state construction is a minimal version of the "map from full state to subsystem description" that would eventually be needed for any Bell-MIPT analysis. The "partially" and "toy_only" qualifiers are honest: this is a finite-system direct-index extraction, not a general mapping.
- `needInvariant`: Moves from `"partially_paid_equivariance_only"` to `"partially_paid_equivariance_plus_empirical_trajectory_check"` because empirical trajectory equivariance adds a second, independent check on Bell-rate correctness (stochastic sampling vs. deterministic master equation).
- `needToyCheck`: Moves to `"partially_paid_rate_algebra_and_conditional_state_toy"` because we are now also testing the conditional-state construction, not just the rate algebra.

**What remains unpaid and why:**

- `needNullModel`: Still unpaid. No null/hypothesis model has been implemented to compare against (e.g., random jumps with uniform rates, or Schrödinger-only evolution without Bell dynamics).
- `needObstruction`: `"bell_jumps_are_not_measurements"` — unchanged. This ticket does not resolve the obstruction; it merely measures conditional-state changes, which is not the same as showing Bell jumps function as measurements.
- `needFaithfulnessReview`: Only `0001` source code has been reviewed so far. `0002A` code needs its own review.

---

## 13. Known Risks

### 13a. Small system, small signals

With 6 sites (Hilbert dimension 64), the subsystem-environment split is likely 3+3 (dimension 8 each). Environment jumps may be rare depending on parameters. The conditional update ratio may be noisy or indeterminate. **Mitigation:** The `bridge_toy_inconclusive` status handles this honestly. The plan does not tune parameters to force a signal.

### 13b. Discrete-time approximation

The discrete-time jump sampling (`p_jump = 1 - exp(-lambda * dt)`) is an approximation to continuous-time Bell dynamics. For the small `dt` values used in `BELL-MIPT-0001` (e.g., 0.001), this is accurate as long as `lambda * dt << 1`. If `lambda` is large, multi-jump events within one `dt` are possible and missed. **Mitigation:** Add a runtime check: if `lambda * dt > 0.1` at any step, log a warning and consider the trajectory potentially inaccurate. Report `max_lambda_dt` in metrics.

### 13c. Psi history memory

Storing `psi` at every time step: `(steps+1) * dim * 16 bytes`. For 6 sites / 1000 steps: ~1 MB. For 10 sites / 100000 steps: ~1.6 GB. **Mitigation:** For `BELL-MIPT-0002A`, cap at 6 sites as in `0001`. Document the scaling limitation. A future ticket could store psi at `sample_every_steps` intervals only, or recompute on the fly.

### 13d. RNG determinism across Go versions

`math/rand` is deterministic for a given seed within a Go major version but the implementation has changed between Go versions. **Mitigation:** Pin Go version in `go.mod`. Document that deterministic output is version-dependent.

### 13e. Conditional norm near zero

When `|psi_m|^2` is small for the actual environment configuration, the conditional state has near-zero norm and normalization amplifies numerical noise. **Mitigation:** The `normThreshold` (1e-12) catches true zeros. For small-but-nonzero norms, the conditional state may be noisy but is still valid. Track `ConditionalNormFailures` and feed into bridge status logic.

### 13f. Fidelity metric sensitivity

The fidelity `F = |<a|b>|^2` is sensitive to phase changes but not to amplitude redistribution within the same phase pattern. Two conditional states with the same amplitude profile but different global phases have `F = 1`. **Mitigation:** This is a known property of fidelity. For the purpose of detecting "jumps cause conditional-state changes," fidelity is appropriate because Bell jumps change the amplitudes (not just phases) of the conditional state. Document this as a known metric limitation.

---

## 14. Non-Goals (Explicit)

The following are explicitly out of scope and must not be implemented, referenced, or implied:

- MIPT phase diagram or transition detection
- Area-law / volume-law scaling analysis
- Entanglement entropy computation
- Mutual information or purity calculations
- Monitored quantum circuits
- Projective measurements (Bell jumps are not claimed to be measurements)
- Lindblad dynamics
- Holography or AdS/CFT connections
- Black hole physics
- Lean theorem proving
- AI agents
- Multi-flavor Majorana chains
- Any physics promotion language in reports
- Continuous-time Gillespie sampler (discrete-time is sufficient for this toy)
- Random trajectory sampling for the master-equation audit (existing audit is deterministic and stays)

---

## 15. Implementation Sequence

### Phase 1: Config + Partition (no behavior change)

1. Add `BridgeConfig` struct to `config.go`.
2. Add `SitePartition` and helpers to a new file or `config.go`.
3. Add validation logic.
4. Write config tests.
5. Verify `go test ./...` passes and default run is unchanged.

### Phase 2: Conditional Wave Function

6. Implement `ComputeConditionalState` in `conditional.go`.
7. Implement `ConditionalFidelity`.
8. Write conditional tests.
9. Verify `go test ./...` passes.

### Phase 3: Trajectory Sampler

10. Implement `SampleInitialConfig` in `trajectory.go`.
11. Implement single-step trajectory logic.
12. Implement `RunTrajectory`.
13. Write trajectory tests (initial distribution, jump probability, determinism).
14. Verify `go test ./...` passes.

### Phase 4: Bridge Orchestrator + Metrics

15. Implement `BridgeMetrics` accumulator in `bridgemetrics.go`.
16. Implement `RunBridge` in `trajectory.go` or `run.go`.
17. Implement `ClassifySignal`.
18. Write bridge metrics tests.
19. Verify `go test ./...` passes.

### Phase 5: Run Integration + Report

20. Modify `run.go` to store psi history and call bridge when enabled.
21. Modify `report.go` to include bridge section.
22. Modify `markdown.go` to include bridge markdown.
23. Update `forbidden.go` with bridge-specific allowlist exceptions.
24. Write integration tests.
25. Verify all validation commands pass.

### Phase 6: Validation

26. `go test ./...`
27. `go test -race ./...`
28. `go vet ./...`
29. Default run (bridge disabled) — compare with `BELL-MIPT-0001` baseline.
30. Bridge-enabled run — verify report structure and metrics.

---

## 16. Recommendation

**Implement `BELL-MIPT-0002A` as a direct extension of the existing `internal/bellmipt` package.** Do not create a separate `internal/bridge` package.

**Rationale:**

1. The bridge code depends heavily on existing types and functions (basis states, Hamiltonian matrix, Bell current, Bell rates, ψ evolution). A separate package would either duplicate these dependencies or require a shared interface layer that adds complexity for no benefit.

2. The bridge adds ~4 files (`trajectory.go`, `conditional.go`, `bridgemetrics.go`, plus tests) to a package that currently has ~12 files. This is a modest increase. The package is not yet large enough to justify splitting.

3. The single-command shape is preserved. The `run.go` orchestrator remains the single entry point. The bridge is a conditional code path within the existing flow, not a new subsystem.

4. If the codebase grows significantly in future tickets (e.g., `0002B`, `0003`), refactoring into sub-packages at that point is straightforward because the bridge code is cleanly separated into its own files with clear interfaces (takes config, basis, H, psi history; returns metrics).

**When to split:** If a future ticket adds more than ~500 lines of new bridge-related code, or if the `internal/bellmipt` package exceeds ~20 files, extract bridge logic into `internal/bellmipt/bridge/` with explicit dependency injection of the base types.

---

## 17. File Inventory Summary

| File | Action | Lines (est.) |
|---|---|---|
| `internal/bellmipt/config.go` | Modify | +60 |
| `internal/bellmipt/trajectory.go` | New | ~200 |
| `internal/bellmipt/conditional.go` | New | ~80 |
| `internal/bellmipt/bridgemetrics.go` | New | ~180 |
| `internal/bellmipt/run.go` | Modify | +80 |
| `internal/bellmipt/report.go` | Modify | +60 |
| `internal/bellmipt/markdown.go` | Modify | +50 |
| `internal/bellmipt/forbidden.go` | Modify | +20 |
| `internal/bellmipt/trajectory_test.go` | New | ~150 |
| `internal/bellmipt/conditional_test.go` | New | ~100 |
| `internal/bellmipt/bridgemetrics_test.go` | New | ~120 |
| `internal/bellmipt/bridge_integration_test.go` | New | ~100 |
| `cmd/bellmipt/main.go` | Unchanged | 0 |

**Total estimated new/modified lines: ~1200** (including tests).

All existing `BELL-MIPT-0001` source files (`basis.go`, `fermion.go`, `hamiltonian.go`, `current.go`, `rates.go`, `evolve.go`, `audit.go`) are **not modified**. They are consumed as-is by the new bridge code.
