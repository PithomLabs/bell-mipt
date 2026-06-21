Here is the detailed implementation plan for `BELL-MIPT-0002A`. It is structured to provide a coding agent with precise, unambiguous instructions while strictly adhering to your architectural constraints, mathematical definitions, and physics non-goals.

---

## 1. File Layout Changes

To ensure backward compatibility and prevent rewriting `BELL-MIPT-0001`, the new functionality will be implemented as an extension within the existing `internal/bellmipt` package.

* **`internal/bellmipt/config.go`**: Update to include the `BridgeConfig` struct.
* **`internal/bellmipt/trajectory.go`**: **[NEW]** Contains the trajectory sampling loop, RNG deterministic seeding, and discrete-time jump logic.
* **`internal/bellmipt/conditional.go`**: **[NEW]** Contains logic for splitting configurations into $Q_A$ and $Q_B$, extracting $\psi_A(a, t)$, and computing fidelities.
* **`internal/bellmipt/bridge_metrics.go`**: **[NEW]** Handles the aggregation of jump counts, fidelity drops, conditional norm failures, and L1 empirical equivariance.
* **`internal/bellmipt/report.go`**: Update `Report` structs to include the new `BridgeReport` sections and JSON tags.
* **`internal/bellmipt/run.go`**: Update the main execution flow to conditionally invoke trajectory simulation if `bridge.enabled` is true.

---

## 2. New Configuration Structs

Extend the existing configuration structs in `config.go` with an optional `Bridge` section.

```go
type BridgeConfig struct {
    Enabled            bool  `json:"enabled"`
    SubsystemSites     []int `json:"subsystem_sites"`
    EnvironmentSites   []int `json:"environment_sites"`
    Trajectories       int   `json:"trajectories"`
    Seed               int64 `json:"seed"`
    SampleEverySteps   int   `json:"sample_every_steps"`
}

// In main Config struct:
// Bridge *BridgeConfig `json:"bridge,omitempty"`

```

**Validation Rules to implement in `config.go`:**

* If `Bridge` is nil or `Enabled` is false, bypass all new checks.
* `SubsystemSites` + `EnvironmentSites` must perfectly partition `[0, N-1]` where `N` is the total number of sites. Check for no overlaps and no missing sites.
* `Trajectories` must be $> 0$.
* `SampleEverySteps` must be $\ge 1$.

---

## 3. New Data Structures

Add structures to manage the state of the bridge audit in `internal/bellmipt/trajectory.go` and `internal/bellmipt/conditional.go`.

```go
type TrajectoryState struct {
    CurrentConfig int // Integer representation of the basis state q
    SubsystemA    int // Extracted state a
    EnvironmentB  int // Extracted state b
}

type BridgeAudit struct {
    Config            *BridgeConfig
    Trajectories      []TrajectoryState
    // Metrics accumulators
    TrajectoryCount          int
    EnvJumps                 int
    SubsystemJumps           int
    FidelityDropEnvJump      float64
    FidelityDropNoEnvJump    float64
    EnvJumpEvents            int
    NoEnvJumpEvents          int
    ConditionalNormFailures  int
    
    // Equivariance tracking
    EmpiricalDist            map[int]int
}

```

---

## 4. Algorithms

### 4.1. Trajectory Sampling

Since the existing framework uses small, fixed time steps `dt`, use discrete-time thinning rather than a full Gillespie algorithm.

**Initialization ($t=0$):**
Seed a deterministic RNG (e.g., `rand.New(rand.NewSource(config.Bridge.Seed))`).
For each trajectory, sample the initial actual configuration $Q_0 = q$ based on the probability:


$$P(Q_0 = q) = |\psi_0(q)|^2$$

**Evolution (Step $t \to t+dt$):**

1. Compute the total outgoing rate for the current configuration $q$:

$$\lambda(q,t) = \sum_{n} \sigma(n \leftarrow q)$$


2. Calculate the jump probability:

$$p_{\text{jump}} = 1 - \exp(-\lambda(q,t) \cdot dt)$$


3. Draw a uniform random float $r \in [0,1)$.
4. If $r < p_{\text{jump}}$, a jump occurs. Determine the destination $n$ by sampling from the normalized rates:

$$P(\text{jump to } n) = \frac{\sigma(n \leftarrow q)}{\lambda(q,t)}$$


5. If $r \ge p_{\text{jump}}$, the configuration remains $q$.

### 4.2. Conditional Wave-Function Construction

At each sample step (dictated by `sample_every_steps`), evaluate the conditional state for every trajectory.

1. **Split Configuration:** Given full configuration $q$ (binary representation), extract bits corresponding to `SubsystemSites` to form $a$, and `EnvironmentSites` to form $b_0$.
2. **Extract Unnormalized Vector:** 
$$\psi_A(a,t) = \Psi(a,b_0,t)$$



Iterate over all possible $2^{|A|}$ configurations of the subsystem, combining them with $b_0$ to map to the full basis state index, and read the complex amplitude from the global $\Psi(t)$.
3. **Normalize:**

$$\psi_{A,\text{normalized}}(a,t) = \frac{\psi_A(a,t)}{\sqrt{\sum_{a'} |\psi_A(a',t)|^2}}$$



*Safety:* If $\sum | \psi_A |^2 < 10^{-12}$, increment `ConditionalNormFailures` and skip fidelity updates for this step.

---

## 5. Bridge Metrics & Fidelity

For each trajectory, at each sampled time step, compare the newly computed $\psi_A(t+dt)$ with the previous $\psi_A(t)$.

1. **Fidelity:**

$$F(t, t+dt) = |\langle\psi_A(t) | \psi_A(t+dt)\rangle|^2$$


2. **Fidelity Drop:**

$$\text{drop} = 1 - F$$


3. **Classification:**
* Compare $Q_B(t)$ with $Q_B(t+dt)$. If changed $\rightarrow$ `environment_jump`.
* Compare $Q_A(t)$ with $Q_A(t+dt)$. If changed $\rightarrow$ `subsystem_jump`.


4. **Aggregation:** Accumulate `drop` into the appropriate bucket (`mean_fidelity_drop_at_environment_jumps` or `mean_fidelity_drop_without_environment_jumps`) based on the classification.
5. **Ratio:**

$$\text{conditional\_update\_ratio} = \frac{\text{mean\_fidelity\_drop\_at\_environment\_jumps}}{\text{mean\_fidelity\_drop\_without\_environment\_jumps}}$$



*(If the denominator is 0 or $< 10^{-15}$, set to 0 and flag in evaluation).*
6. **Empirical Equivariance:**

$$\text{empirical\_equivariance\_l1}(t) = \sum_{q} \left| \frac{\text{freq}(q,t)}{\text{trajectories}} - |\psi_q(t)|^2 \right|$$



---

## 6. Report Schema Changes

Extend the output JSON to include the `BridgeReport`.

```go
type BridgeInterpretation struct {
    MonitoringLikeSignal string `json:"monitoring_like_signal"`
    Reason               string `json:"reason"`
}

type BridgeMetrics struct {
    TrajectoryCount                     int     `json:"trajectory_count"`
    MeanJumpCount                       float64 `json:"mean_jump_count"`
    MeanEnvironmentJumpCount            float64 `json:"mean_environment_jump_count"`
    MeanSubsystemJumpCount              float64 `json:"mean_subsystem_jump_count"`
    ConditionalNormFailures             int     `json:"conditional_norm_failures"`
    MeanFidelityDropAtEnvJumps          float64 `json:"mean_fidelity_drop_at_environment_jumps"`
    MeanFidelityDropWithoutEnvJumps     float64 `json:"mean_fidelity_drop_without_environment_jumps"`
    ConditionalUpdateRatio              float64 `json:"conditional_update_ratio"`
    MaxEmpiricalEquivarianceL1          float64 `json:"max_empirical_equivariance_l1"`
    FinalEmpiricalEquivarianceL1        float64 `json:"final_empirical_equivariance_l1"`
}

type BridgeReport struct {
    Enabled          bool                  `json:"enabled"`
    BridgeGoal       string                `json:"bridge_goal"`
    BridgeStatus     string                `json:"bridge_status"`
    Trajectories     int                   `json:"trajectories"`
    SubsystemSites   []int                 `json:"subsystem_sites"`
    EnvironmentSites []int                 `json:"environment_sites"`
    SampleEverySteps int                   `json:"sample_every_steps"`
    Metrics          BridgeMetrics         `json:"metrics"`
    Interpretation   BridgeInterpretation  `json:"interpretation"`
}

```

---

## 7. Goal / Bridge Status Logic

Maintain `goal_status` strictly for the master-equation logic (BELL-MIPT-0001). Implement the following state machine for `bridge_status`:

* **`bridge_disabled`**: If `config.Bridge == nil` or `config.Bridge.Enabled == false`.
* **`bridge_toy_failed`**: If `panic` occurs, math/matrix bounds fail, partition validation fails, or metrics cannot be computed due to structural code errors.
* **`bridge_toy_inconclusive`**: If `ConditionalNormFailures` > 5% of total evaluations, OR if `EnvJumpEvents` == 0 (preventing ratio calculation), OR if `MaxEmpiricalEquivarianceL1` > 0.15 (indicating trajectory sampling has heavily drifted from master equation, accounting for finite sampling noise).
* **`bridge_toy_passed`**: Trajectories sampled successfully, state computes reliably, norm failures are minimal, and equivariance is within finite-sample tolerance.

*Signal Logic (`monitoring_like_signal`):*

* If Ratio > 1.5 $\rightarrow$ `candidate_signal`.
* If Ratio > 1.1 $\rightarrow$ `weak_signal`.
* Otherwise $\rightarrow$ `no_clear_signal`.

---

## 8. Required Tests

In `internal/bellmipt/bridge_test.go` and `config_test.go`:

* **Config Tests**:
* `TestBridgeDisabledDefaults`: Ensure nil bridge = disabled.
* `TestBridgePartitionValidation`: Test overlapping arrays, missing sites, accepted splits.


* **Trajectory Tests**:
* `TestInitialSampling`: Generate 10k samples of a known $|\psi_0|^2$; assert L1 error is within statistical bounds.
* `TestDeterministicTrajectories`: Assert identical seed outputs identical jump sequences.
* `TestOutgoingRateZero`: Ensure `p_jump` == 0 and configuration remains static when $\lambda = 0$.


* **Conditional Wave Function Tests**:
* `TestStateSplitting`: Hardcode a 4-site state, extract $A=[0,1]$, $B=[2,3]$. Verify binary operations.
* `TestConditionalNorm`: Test vector size ($2^{|A|}$) and proper normalization.
* `TestFidelityCalculations`: Validate $F=1$ for identical vectors; correct drop logic.


* **Report & Output Tests**:
* `TestBridgeDisabledPreservesOldReport`: Ensure `bridge` key is handled correctly and `BELL-MIPT-0001` metrics match exactly.
* `TestForbiddenLanguageAudit`: Ensure tests fail if strings like "MIPT established" or "holography" appear in the codebase or test assertions.



---

## 9. Validation Commands

The coding agent must run and pass the following pipeline:

```bash
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/bellmipt --out /tmp/bellmipt-default
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge

```

---

## 10. EBP Debt Updates

Upon successful execution of a bridge-enabled run, write the following to the debt tracking logic/report:

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

---

## 11. Known Risks & Non-Goals

### Risks

* **Numerical Instability:** Wavefunction amplitudes for certain environmental configurations $b_0$ may be extremely close to zero, leading to division-by-zero during conditional normalization. The `ConditionalNormFailures` threshold explicitly guards against this.
* **Finite Sampling Noise:** 200 trajectories (as suggested in the toy config) will produce a noisy empirical L1 distance. Do not set master-equation-level L1 thresholds ($10^{-11}$) for the empirical check.

### Strict Non-Goals (Required limitations in report text)

The final `report.go` strings must explicitly include:

* "This samples Bell configuration histories in a finite toy model only."
* "This conditional-state audit is not a monitored quantum trajectory simulation."
* "This does not implement MIPT."
* "This does not show Bell jumps are measurements."
* "This does not establish a Bell-MIPT bridge."
* "This does not support any holography or black-hole claim."
* "This is not a physics promotion."

---

## 12. Architectural Recommendation

**Recommendation: Direct Extension (`internal/bellmipt`).** Do *not* create a separate binary or an entirely new package outside of `internal/bellmipt`. The trajectory sampler requires deep, high-frequency access to `basis.go`, `rates.go`, and the wavefunction arrays in `evolve.go`.

Creating a separate package would force these internals to be exported, polluting the API boundary. By keeping it within `internal/bellmipt`, you ensure that `BELL-MIPT-0002A` can access the necessary Hamiltonian matrices and basis mappings cleanly, while feature-flagging the logic tightly behind `config.Bridge.Enabled`. The command interface remains untouched, fulfilling the requirement perfectly.
