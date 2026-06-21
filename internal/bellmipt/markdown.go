package bellmipt

import (
	"fmt"
	"strings"
)

// ReportMarkdown renders the Report struct as a Markdown string.
func ReportMarkdown(r Report) string {
	var sb strings.Builder

	sb.WriteString("# BELL-MIPT-0001 Toy Check Report\n\n")

	sb.WriteString("## Status\n")
	sb.WriteString(fmt.Sprintf("- **Goal Status**: %s\n", r.GoalStatus))
	sb.WriteString(fmt.Sprintf("- **Toy ID**: %s\n", r.ToyID))
	sb.WriteString(fmt.Sprintf("- **Schema Version**: %s\n\n", r.SchemaVersion))

	if r.Bridge != nil && r.Bridge.Enabled {
		renderBridgeSection(&sb, r.Bridge)
	}

	sb.WriteString("## Scope\n")
	sb.WriteString(fmt.Sprintf("- **Toy Analysis Only**: %t\n", r.ToyAnalysisOnly))
	sb.WriteString(fmt.Sprintf("- **Physics Claim**: %s\n", r.PhysicsClaim))
	sb.WriteString(fmt.Sprintf("- **Model**: %s\n", r.Model))
	sb.WriteString(fmt.Sprintf("- **Sites**: %d\n", r.Sites))
	sb.WriteString(fmt.Sprintf("- **Hilbert Dimension**: %d\n", r.HilbertDim))
	sb.WriteString(fmt.Sprintf("- **Boundary**: %s\n", r.Boundary))
	sb.WriteString(fmt.Sprintf("- **Goal**: %s\n\n", r.Goal))

	sb.WriteString("## What Was Checked\n")
	sb.WriteString(fmt.Sprintf("- Hamiltonian Hermitian: %t\n", r.Checks.HamiltonianHermitian))
	sb.WriteString(fmt.Sprintf("- State Norm Preserved: %t\n", r.Checks.StateNormPreserved))
	sb.WriteString(fmt.Sprintf("- Rho Sum Preserved: %t\n", r.Checks.RhoSumPreserved))
	sb.WriteString(fmt.Sprintf("- Current Antisymmetric: %t\n", r.Checks.CurrentAntisymmetric))
	sb.WriteString(fmt.Sprintf("- Rates Nonnegative: %t\n", r.Checks.RatesNonnegative))
	sb.WriteString(fmt.Sprintf("- Equivariance Error Within Tolerance: %t\n", r.Checks.EquivarianceErrorWithinTolerance))
	sb.WriteString(fmt.Sprintf("- No NaN Or Inf: %t\n", r.Checks.NoNaNOrInf))
	sb.WriteString(fmt.Sprintf("- Forbidden Language Passed: %t\n\n", r.Checks.ForbiddenLanguagePassed))

	sb.WriteString("## Metrics\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|---|---|\n")
	sb.WriteString(fmt.Sprintf("| Max Hermitian Error | %e |\n", r.Metrics.MaxHermitianError))
	sb.WriteString(fmt.Sprintf("| Max Norm Error | %e |\n", r.Metrics.MaxNormError))
	sb.WriteString(fmt.Sprintf("| Max Rho Sum Error | %e |\n", r.Metrics.MaxRhoSumError))
	sb.WriteString(fmt.Sprintf("| Max Rho Negative Violation | %e |\n", r.Metrics.MaxRhoNegativeViolation))
	sb.WriteString(fmt.Sprintf("| Max Current Antisymmetry Error | %e |\n", r.Metrics.MaxCurrentAntisymmetryError))
	sb.WriteString(fmt.Sprintf("| Max Rate Negative Violation | %e |\n", r.Metrics.MaxRateNegativeViolation))
	sb.WriteString(fmt.Sprintf("| Max Equivariance L1 Error | %e |\n", r.Metrics.MaxEquivarianceL1Error))
	sb.WriteString(fmt.Sprintf("| Final Equivariance L1 Error | %e |\n", r.Metrics.FinalEquivarianceL1Error))
	sb.WriteString(fmt.Sprintf("| Mean Equivariance L1 Error | %e |\n", r.Metrics.MeanEquivarianceL1Error))
	sb.WriteString(fmt.Sprintf("| Mean Total Bell Activity | %e |\n", r.Metrics.MeanTotalBellActivity))
	sb.WriteString(fmt.Sprintf("| Max Total Bell Activity | %e |\n", r.Metrics.MaxTotalBellActivity))
	sb.WriteString(fmt.Sprintf("| Probability Floor Hits | %d |\n\n", r.Metrics.ProbabilityFloorHits))

	sb.WriteString("## EBP Debt Status\n")
	sb.WriteString("| Requirement | Status |\n")
	sb.WriteString("|---|---|\n")
	sb.WriteString(fmt.Sprintf("| needMap | %s |\n", r.DebtStatus["needMap"]))
	sb.WriteString(fmt.Sprintf("| needInvariant | %s |\n", r.DebtStatus["needInvariant"]))
	sb.WriteString(fmt.Sprintf("| needToyCheck | %s |\n", r.DebtStatus["needToyCheck"]))
	sb.WriteString(fmt.Sprintf("| needNullModel | %s |\n", r.DebtStatus["needNullModel"]))
	sb.WriteString(fmt.Sprintf("| needObstruction | %s |\n", r.DebtStatus["needObstruction"]))
	sb.WriteString(fmt.Sprintf("| needFaithfulnessReview | %s |\n\n", r.DebtStatus["needFaithfulnessReview"]))

	sb.WriteString("## Limitations\n")
	for _, lim := range r.Limitations {
		sb.WriteString(fmt.Sprintf("- %s\n", lim))
	}

	return sb.String()
}

func renderBridgeSection(sb *strings.Builder, b *BridgeReport) {
	sb.WriteString("## Bridge Audit (0002A)\n\n")
	sb.WriteString(fmt.Sprintf("**Bridge Status**: `%s`\n\n", b.BridgeStatus))

	if b.BridgeGoal != "" {
		sb.WriteString(fmt.Sprintf("**Bridge Goal**: %s\n\n", b.BridgeGoal))
	}

	sb.WriteString("### Configuration\n\n")
	sb.WriteString(fmt.Sprintf("- **Trajectories**: %d\n", b.Trajectories))
	sb.WriteString(fmt.Sprintf("- **Sample Every Steps**: %d\n", b.SampleEverySteps))
	sb.WriteString(fmt.Sprintf("- **Subsystem Sites (Canonical)**: %v\n", b.SubsystemSitesCanonical))
	sb.WriteString(fmt.Sprintf("- **Environment Sites (Canonical)**: %v\n\n", b.EnvironmentSitesCanonical))

	if b.Metrics != nil {
		sb.WriteString("### Metrics\n\n")
		sb.WriteString(fmt.Sprintf("- **Mean Jump Count**: %.4f\n", b.Metrics.MeanJumpCount))
		sb.WriteString(fmt.Sprintf("- **Mean Strict Environment Jump Count**: %.4f\n", b.Metrics.MeanStrictEnvironmentJumpCount))
		sb.WriteString(fmt.Sprintf("- **Mean Strict Subsystem Jump Count**: %.4f\n", b.Metrics.MeanStrictSubsystemJumpCount))
		sb.WriteString(fmt.Sprintf("- **Mean Boundary Crossing Jump Count**: %.4f\n", b.Metrics.MeanBoundaryCrossingJumpCount))
		sb.WriteString(fmt.Sprintf("- **Conditional Norm Failures**: %d\n", b.Metrics.ConditionalNormFailures))

		sb.WriteString(fmt.Sprintf("- **Conditional Update Ratio Status**: %s\n", b.Metrics.ConditionalUpdateRatioStatus))
		if b.Metrics.ConditionalUpdateRatio != nil {
			sb.WriteString(fmt.Sprintf("- **Conditional Update Ratio**: %.4f\n", *b.Metrics.ConditionalUpdateRatio))
		}
		sb.WriteString(fmt.Sprintf("- **Max Lambda DT**: %.4f\n", b.Metrics.MaxLambdaDT))
		sb.WriteString(fmt.Sprintf("- **Final Empirical Equivariance L1**: %.4e\n\n", b.Metrics.FinalEmpiricalEquivarianceL1))
	}

	if b.Interpretation != nil {
		sb.WriteString("### Interpretation\n\n")
		sb.WriteString(fmt.Sprintf("**Environment-Correlated Conditional Update**: `%s`\n\n", b.Interpretation.EnvironmentCorrelatedConditionalUpdate))
		sb.WriteString(fmt.Sprintf("> %s\n\n", b.Interpretation.Reason))
	}

	if len(b.Warnings) > 0 {
		sb.WriteString("### Warnings\n\n")
		for _, w := range b.Warnings {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", w.Code, w.Message))
		}
		sb.WriteString("\n")
	}
}
