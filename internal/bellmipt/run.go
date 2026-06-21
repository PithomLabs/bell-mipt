package bellmipt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RunResult holds the output report and any initialization/IO error.
// Goal failures or inconclusive results do not set the Error field.
type RunResult struct {
	Report Report
	Error  error
}

// ExitCodeForGoalStatus maps the final goal status string to an exit code.
// 0 for toy_goal_passed, 1 for toy_goal_failed, 2 for toy_goal_inconclusive.
func ExitCodeForGoalStatus(status string) int {
	switch status {
	case "toy_goal_passed":
		return 0
	case "toy_goal_failed":
		return 1
	case "toy_goal_inconclusive":
		return 2
	default:
		return 1
	}
}

// Run performs the complete simulation and audit check.
func Run(cfg Config, outDir string) RunResult {
	// 1. Validate configuration
	if err := cfg.Validate(); err != nil {
		return RunResult{Error: fmt.Errorf("configuration validation failed: %w", err)}
	}

	// 2. Create output directory
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return RunResult{Error: fmt.Errorf("failed to create output directory: %w", err)}
	}

	// 3. Write input.json
	inputData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return RunResult{Error: fmt.Errorf("failed to marshal config: %w", err)}
	}
	if err := os.WriteFile(filepath.Join(outDir, "input.json"), inputData, 0644); err != nil {
		return RunResult{Error: fmt.Errorf("failed to write input.json: %w", err)}
	}

	// 4. Build basis
	basis, err := NewBasis(cfg.Sites)
	if err != nil {
		return RunResult{Error: fmt.Errorf("failed to build basis: %w", err)}
	}

	// 5. Build Hamiltonian matrix
	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		return RunResult{Error: fmt.Errorf("failed to build Hamiltonian: %w", err)}
	}

	// 6. Audit Hermiticity
	hermErr := HermitianError(H)
	audit := &AuditAccumulator{}
	audit.RecordHermitianError(hermErr)

	// 7. Generate deterministic random normalized initial state psi0
	// Snapshot storage for 0002A bridge audit (Pass 2)
	psiSnapshots := make([][]complex128, 0, cfg.Time.Steps+1)

	psi := RandomNormalizedState(basis.Dim, cfg.InitialState.Seed)
	rho := Probabilities(psi)
	psiSnapshots = append(psiSnapshots, psi)

	// 8. Evolution loop
	for step := 0; step < cfg.Time.Steps; step++ {
		nextPsi, nextRho, stepStats := RK4Step(H, psi, rho, cfg.Time.Dt)

		psi = nextPsi
		rho = nextRho
		psiSnapshots = append(psiSnapshots, psi)

		// Check and record step audit
		stepAudit := ComputeStepAudit(H, nextPsi, nextRho, stepStats)
		audit.RecordStep(stepAudit)
	}

	// 9. Draft report with placeholder for forbidden language check
	draftForbidden := ForbiddenLanguageAudit{Passed: true}
	report := BuildReport(cfg, audit, draftForbidden)

	// Pass 2: Optional bridge audit
	if cfg.Bridge != nil && cfg.Bridge.Enabled {
		report.SchemaVersion = "bell_mipt_report_v0_2a"
		bridgeReport := RunBridgeAudit(cfg.Bridge, basis, H, psiSnapshots, cfg.Time.Dt)
		report.Bridge = bridgeReport

		// Update Debt Status ONLY if bridge audit completed successfully
		if bridgeReport.BridgeStatus == "bridge_audit_completed" {
			report.DebtStatus["needMap"] = "partially_paid_environment_projected_conditional_vector_toy_only"
			report.DebtStatus["needInvariant"] = "partially_paid_equivariance_plus_descriptive_empirical_trajectory_check"
			report.DebtStatus["needToyCheck"] = "partially_paid_rate_algebra_and_conditional_vector_toy"
			report.DebtStatus["needFaithfulnessReview"] = "source_code_review_required_for_0002A"
		}
	}

	// 10. Render draft report markdown
	mdText := ReportMarkdown(report)

	// 11. Scan rendered markdown for forbidden language
	finalForbidden := AuditForbiddenLanguage(mdText)

	// 12. Build final report with actual forbidden language results
	report.ForbiddenLanguageAudit = finalForbidden
	if !finalForbidden.Passed {
		report.GoalStatus = "toy_goal_failed"
		report.Checks.ForbiddenLanguagePassed = false
	}

	// 13. Render final markdown report
	finalMdText := ReportMarkdown(report)

	// 14. Write report.json
	reportJSONBytes, err := ReportJSON(report)
	if err != nil {
		return RunResult{Error: fmt.Errorf("failed to marshal report: %w", err)}
	}
	if err := os.WriteFile(filepath.Join(outDir, "report.json"), reportJSONBytes, 0644); err != nil {
		return RunResult{Error: fmt.Errorf("failed to write report.json: %w", err)}
	}

	// 15. Write report.md
	if err := os.WriteFile(filepath.Join(outDir, "report.md"), []byte(finalMdText), 0644); err != nil {
		return RunResult{Error: fmt.Errorf("failed to write report.md: %w", err)}
	}

	// 16. Print console summary
	fmt.Printf("Simulation run completed.\n")
	fmt.Printf("Output directory: %s\n", outDir)
	fmt.Printf("Goal Status:      %s\n", report.GoalStatus)
	fmt.Printf("Max Equivariance L1 Error: %e\n", report.Metrics.MaxEquivarianceL1Error)

	return RunResult{
		Report: report,
	}
}
