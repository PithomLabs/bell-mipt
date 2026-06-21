# BELL-MIPT-0001 Toy Check Report

## Status
- **Goal Status**: toy_goal_passed
- **Toy ID**: BELL-MIPT-0001
- **Schema Version**: bell_mipt_report_v0_2a

## Bridge Audit (0002A)

**Bridge Status**: `bridge_audit_completed`

**Bridge Goal**: sample_bell_trajectories_and_audit_environment_projected_conditional_vectors

### Configuration

- **Trajectories**: 200
- **Sample Every Steps**: 1
- **Subsystem Sites (Canonical)**: [0 2 5]
- **Environment Sites (Canonical)**: [1 3 4]

### Metrics

- **Mean Jump Count**: 2.3900
- **Mean Strict Environment Jump Count**: 0.3750
- **Mean Strict Subsystem Jump Count**: 0.3600
- **Mean Boundary Crossing Jump Count**: 1.6550
- **Conditional Norm Failures**: 0
- **Conditional Update Ratio Status**: available
- **Conditional Update Ratio**: 217088.4907
- **Max Lambda DT**: 0.0167
- **Final Empirical Equivariance L1**: 3.4554e-01

### Interpretation

**Environment-Correlated Conditional Update**: `candidate_correlation`

> Descriptive finite-toy diagnostic only. This does not establish a Bell-MIPT bridge.

### Warnings

- **jordan_wigner_partition_convention**: Conditional vectors are projected in the canonical occupation basis. No additional Jordan-Wigner sign is applied during projection; all phases are inherited from the full wavefunction amplitudes.
- **boundary_crossing_jumps_observed**: boundary_crossing jumps occurred

## Scope
- **Toy Analysis Only**: true
- **Physics Claim**: none
- **Model**: finite_kitaev_chain
- **Sites**: 6
- **Hilbert Dimension**: 64
- **Boundary**: periodic
- **Goal**: compute_bell_rates_and_verify_equivariance

## What Was Checked
- Hamiltonian Hermitian: true
- State Norm Preserved: true
- Rho Sum Preserved: true
- Current Antisymmetric: true
- Rates Nonnegative: true
- Equivariance Error Within Tolerance: true
- No NaN Or Inf: true
- Forbidden Language Passed: true

## Metrics
| Metric | Value |
|---|---|
| Max Hermitian Error | 0.000000e+00 |
| Max Norm Error | 1.810774e-13 |
| Max Rho Sum Error | 7.771561e-16 |
| Max Rho Negative Violation | 0.000000e+00 |
| Max Current Antisymmetry Error | 0.000000e+00 |
| Max Rate Negative Violation | 0.000000e+00 |
| Max Equivariance L1 Error | 3.134990e-11 |
| Final Equivariance L1 Error | 1.783272e-11 |
| Mean Equivariance L1 Error | 1.542153e-11 |
| Mean Total Bell Activity | 2.412969e+00 |
| Max Total Bell Activity | 2.463512e+00 |
| Probability Floor Hits | 0 |

## EBP Debt Status
| Requirement | Status |
|---|---|
| needMap | partially_paid_environment_projected_conditional_vector_toy_only |
| needInvariant | partially_paid_equivariance_plus_descriptive_empirical_trajectory_check |
| needToyCheck | partially_paid_rate_algebra_and_conditional_vector_toy |
| needNullModel | unpaid |
| needObstruction | bell_jumps_are_not_measurements |
| needFaithfulnessReview | source_code_review_required_for_0002A |

## Limitations
- This checks Bell-rate algebra in a finite toy model only.
- This does not implement MIPT.
- This does not show Bell jumps are measurements.
- This does not construct a conditional-wave-function bridge.
- This does not support any holography or black-hole claim.
- This is not a physics promotion.
