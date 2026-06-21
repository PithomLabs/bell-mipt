# BELL-MIPT-0001 Toy Check Report

## Status
- **Goal Status**: toy_goal_passed
- **Toy ID**: BELL-MIPT-0001
- **Schema Version**: bell_mipt_report_v0

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
| needMap | unpaid |
| needInvariant | partially_paid_equivariance_only |
| needToyCheck | partially_paid_rate_algebra_only |
| needNullModel | unpaid |
| needObstruction | bell_jumps_are_not_measurements |
| needFaithfulnessReview | unpaid |

## Limitations
- This checks Bell-rate algebra in a finite toy model only.
- This does not implement MIPT.
- This does not show Bell jumps are measurements.
- This does not construct a conditional-wave-function bridge.
- This does not support any holography or black-hole claim.
- This is not a physics promotion.
