# BELL-MIPT-0002A Implementation Plan

This plan details the implementation of **BELL-MIPT-0002A**: adding a Bell trajectory sampler and environment-projected conditional-vector audit to the existing `BELL-MIPT-0001` codebase, without disturbing the baseline.

## User Review Required

> [!IMPORTANT]
> Please review this plan to confirm that it faithfully captures the requirements of `prompt_plan3.md`, particularly the two-pass architecture, test-first requirements, and the strict isolation of boundary-crossing jumps.

## Proposed Changes

The implementation uses a two-pass architecture:
1. **Pass 1**: Run the existing `BELL-MIPT-0001` deterministic master-equation/equivariance audit, storing `ψ[k]` snapshots at full time-step boundaries.
2. **Pass 2**: If the bridge config is enabled, sample Bell trajectories from the stored `ψ[k]`, construct environment-projected conditional vectors, compute fidelity-drop metrics, and emit bridge statuses/warnings.

---

### Bridge Configuration & Validation

#### [NEW] [bridge_config.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/bridge_config.go)
- Define `BridgeConfig` struct (`Enabled`, `SubsystemSites`, `EnvironmentSites`, `Trajectories`, `Seed`, `SampleEverySteps`).
- Implement validation logic (non-empty, no duplicates, no overlap, etc.).
- Produce Canonical (sorted) vs Requested site lists.

#### [NEW] [bridge_config_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/bridge_config_test.go)
- Unit tests for configuration validation.

#### [MODIFY] [config.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/config.go)
- Extend `Config` struct with `Bridge *BridgeConfig `json:"bridge,omitempty"``.

#### [NEW] [bellmipt_bridge.json](file:///home/chaschel/Documents/ibm/go/bft-mipt/bellmipt_bridge.json)
- Example configuration file with `bridge` enabled.

---

### Site Split & Conditional Vectors

#### [NEW] [site_split.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/site_split.go)
- Define `SiteSplit` struct.
- Implement `SplitConfig(q uint64, split SiteSplit) (subQ, envQ uint64)` and `CombineConfig(...)`. Ensure non-contiguous partition support.

#### [NEW] [site_split_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/site_split_test.go)
- Tests for `TestSplitCombineRoundTripAllConfigs` and `TestNonContiguousPartition`.

#### [NEW] [conditional.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/conditional.go)
- Implement `ConditionalProjector` and `BuildConditionalProjector()`.
- Implement `EnvironmentProjectedConditionalVector()`.
- Implement `ConditionalFidelity()` to compute $1 - |\sum \bar{a}_i b_i|^2$ (with 1e-12 clamping).

#### [NEW] [conditional_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/conditional_test.go)
- Tests: `TestConditionalVectorDimension`, `TestConditionalProjectionPreservesPhase`, `TestConditionalProjectorGroupsByEnvironment`, `TestConditionalNormFailureThresholds`.

---

### Trajectory Sampling

#### [NEW] [trajectory.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/trajectory.go)
- Distinguish `QIndex` and `QConfig`.
- Implement deterministic trajectory sampler logic.
- Time-step alignment logic using $\psi[k]$.
- Implement discrete-time thinning: $p\_jump = 1 - \exp(-\lambda dt)$, and track `max_lambda_dt`.
- Implement `RateProbabilityFloor` (1e-14) guard.

#### [NEW] [trajectory_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/trajectory_test.go)
- Tests: `TestTimeStepAlignmentUsesPsiKForIntervalK`, `TestRateOrientationDestinationSource`, `TestTotalOutgoingRateUsesSourceColumn`, `TestFakeRateTrajectoryKnownDestination`, `TestJumpFrequencyMatchesExpectedProbability`, `TestBridgeRatesMatch0001Rates`, `TestZeroRateNoJumpInconclusive`, `TestNearZeroCurrentConfigurationProbability`, `TestMaxLambdaDTWarningAndInconclusive`.

---

### Bridge Metrics & Equivariance

#### [NEW] [bridgemetrics.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/bridgemetrics.go)
- Jump classification: `no_jump`, `strict_environment_jump`, `strict_subsystem_jump`, `boundary_crossing_jump`.
- Compute conditional update ratio.

#### [NEW] [bridgemetrics_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/bridgemetrics_test.go)
- Tests: `TestBoundaryCrossingJumpBothSidesChanged`, `TestBoundaryCrossingJumpExcludedFromPrimaryRatio`, `TestBoundaryCrossingJumpReportedSeparately`, `TestConditionalUpdateRatioNullWhenLowEvents`.

#### [NEW] [equivariance_empirical.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/equivariance_empirical.go)
- Empirical trajectory equivariance diagnostics at sampled steps ($L_1$ distance tracking).

---

### Reporting & Orchestration

#### [NEW] [bridge_report.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/bridge_report.go)
- Define bridge report schema structures (`BridgeReport`, `BridgeInterpretation`, `BridgeWarning`).
- Interpret the environment-correlated conditional update (not_assessed, no_clear_correlation, weak_correlation, candidate_correlation).

#### [NEW] [bridge_report_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/bridge_report_test.go)
- Reporting logic test coverage.

#### [NEW] [bridge_integration_test.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/bridge_integration_test.go)
- Tests: `TestBridgeDisabledPreserves0001SemanticMetrics`, `TestEmpiricalEquivarianceSampledStepsOnly`, `TestEmpiricalEquivarianceAgainstPsiSquared`.

#### [MODIFY] [run.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/run.go)
- Orchestrate the two-pass architecture.
- Pass 1: standard audit, saving $\psi$ snapshots.
- Pass 2: bridge audit using snapshots.

#### [MODIFY] [report.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/report.go)
- Extend report structure to conditionally emit schema `bell_mipt_report_v0_2a` and the nested `bridge` object.

#### [MODIFY] [markdown.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/markdown.go)
- Add rendering logic for the new bridge fields, warnings, explicit non-claims, and interpretation.

#### [MODIFY] [forbidden.go](file:///home/chaschel/Documents/ibm/go/bft-mipt/internal/bellmipt/forbidden.go)
- Add new forbidden language phrases (e.g. `Bell jumps are measurements`, `Bohmian mechanics validated`, etc.) and allowed negated limitations.
- Add `TestForbiddenAuditScansFullJSONAndMarkdown` test to existing `report_test.go` or `forbidden_test.go`.

## Verification Plan

### Automated Tests
```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
```

### Integration Verification
```bash
go run ./cmd/bellmipt --out /tmp/bellmipt-default
go run ./cmd/bellmipt --config bellmipt_bridge.json --out /tmp/bellmipt-bridge
```
I will then verify that `/tmp/bellmipt-default/report.json` remains identical in semantic metrics and schema to `BELL-MIPT-0001`, and that `/tmp/bellmipt-bridge/report.json` correctly outputs the `bell_mipt_report_v0_2a` schema with the expected diagnostic properties and limitations.
