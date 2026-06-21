package bellmipt

import "encoding/json"

// Report is the main output structure for the BELL-MIPT-0001 toy check.
type Report struct {
	SchemaVersion          string                 `json:"schema_version"`
	ToyID                  string                 `json:"toy_id"`
	ToyAnalysisOnly        bool                   `json:"toy_analysis_only"`
	PhysicsClaim           string                 `json:"physics_claim"`
	Model                  string                 `json:"model"`
	Sites                  int                    `json:"sites"`
	HilbertDim             int                    `json:"hilbert_dim"`
	Boundary               string                 `json:"boundary"`
	Goal                   string                 `json:"goal"`
	GoalStatus             string                 `json:"goal_status"`
	Checks                 Checks                 `json:"checks"`
	Metrics                Metrics                `json:"metrics"`
	DebtStatus             map[string]string      `json:"debt_status"`
	Limitations            []string               `json:"limitations"`
	ForbiddenLanguageAudit ForbiddenLanguageAudit `json:"forbidden_language_audit"`
	Bridge                 *BridgeReport          `json:"bridge,omitempty"`
}

// Checks holds the pass/fail status for each audit check.
type Checks struct {
	HamiltonianHermitian             bool `json:"hamiltonian_hermitian"`
	StateNormPreserved               bool `json:"state_norm_preserved"`
	RhoSumPreserved                  bool `json:"rho_sum_preserved"`
	CurrentAntisymmetric             bool `json:"current_antisymmetric"`
	RatesNonnegative                 bool `json:"rates_nonnegative"`
	EquivarianceErrorWithinTolerance bool `json:"equivariance_error_within_tolerance"`
	NoNaNOrInf                       bool `json:"no_nan_or_inf"`
	ForbiddenLanguagePassed          bool `json:"forbidden_language_passed"`
}

// Metrics holds the quantitative results from the simulation.
type Metrics struct {
	MaxHermitianError           float64 `json:"max_hermitian_error"`
	MaxNormError                float64 `json:"max_norm_error"`
	MaxRhoSumError              float64 `json:"max_rho_sum_error"`
	MaxRhoNegativeViolation     float64 `json:"max_rho_negative_violation"`
	MaxCurrentAntisymmetryError float64 `json:"max_current_antisymmetry_error"`
	MaxRateNegativeViolation    float64 `json:"max_rate_negative_violation"`
	MaxEquivarianceL1Error      float64 `json:"max_equivariance_l1_error"`
	FinalEquivarianceL1Error    float64 `json:"final_equivariance_l1_error"`
	MeanEquivarianceL1Error     float64 `json:"mean_equivariance_l1_error"`
	MeanTotalBellActivity       float64 `json:"mean_total_bell_activity"`
	MaxTotalBellActivity        float64 `json:"max_total_bell_activity"`
	ProbabilityFloorHits        int     `json:"probability_floor_hits"`
}

// RequiredDebtStatus returns the fixed debt status for BELL-MIPT-0001.
func RequiredDebtStatus() map[string]string {
	return map[string]string{
		"needMap":                "unpaid",
		"needInvariant":          "partially_paid_equivariance_only",
		"needToyCheck":           "partially_paid_rate_algebra_only",
		"needNullModel":          "unpaid",
		"needObstruction":        "bell_jumps_are_not_measurements",
		"needFaithfulnessReview": "unpaid",
	}
}

// RequiredLimitations returns the fixed limitation statements for BELL-MIPT-0001.
func RequiredLimitations() []string {
	return []string{
		"This checks Bell-rate algebra in a finite toy model only.",
		"This does not implement MIPT.",
		"This does not show Bell jumps are measurements.",
		"This does not construct a conditional-wave-function bridge.",
		"This does not support any holography or black-hole claim.",
		"This is not a physics promotion.",
	}
}

// BuildReport constructs the Report from config and accumulated audit data.
func BuildReport(cfg Config, audit *AuditAccumulator, forbiddenAudit ForbiddenLanguageAudit) Report {
	currentTol := CurrentAntisymmetryTolerance(cfg.Audit.HermitianTolerance)

	checks := Checks{
		HamiltonianHermitian:             audit.MaxHermitianError <= cfg.Audit.HermitianTolerance,
		StateNormPreserved:               audit.MaxNormError <= cfg.Audit.NormTolerance,
		RhoSumPreserved:                  audit.MaxRhoSumError <= cfg.Audit.NormTolerance,
		CurrentAntisymmetric:             audit.MaxCurrentAntisymmetryError <= currentTol,
		RatesNonnegative:                 audit.MaxRateNegativeViolation <= 0,
		EquivarianceErrorWithinTolerance: audit.MaxEquivarianceL1Error <= cfg.Audit.EquivarianceTolerance,
		NoNaNOrInf:                       !audit.NaNOrInfDetected,
		ForbiddenLanguagePassed:          forbiddenAudit.Passed,
	}

	metrics := Metrics{
		MaxHermitianError:           audit.MaxHermitianError,
		MaxNormError:                audit.MaxNormError,
		MaxRhoSumError:              audit.MaxRhoSumError,
		MaxRhoNegativeViolation:     audit.MaxRhoNegativeViolation,
		MaxCurrentAntisymmetryError: audit.MaxCurrentAntisymmetryError,
		MaxRateNegativeViolation:    audit.MaxRateNegativeViolation,
		MaxEquivarianceL1Error:      audit.MaxEquivarianceL1Error,
		FinalEquivarianceL1Error:    audit.FinalEquivarianceL1Error,
		MeanEquivarianceL1Error:     audit.MeanEquivarianceL1Error(),
		MeanTotalBellActivity:       audit.MeanBellActivity(),
		MaxTotalBellActivity:        audit.MaxTotalBellActivity,
		ProbabilityFloorHits:        audit.TotalProbabilityFloorHits,
	}

	goalStatus := audit.DetermineGoalStatus(cfg.Audit, forbiddenAudit.Passed)

	return Report{
		SchemaVersion:          "bell_mipt_report_v0",
		ToyID:                  "BELL-MIPT-0001",
		ToyAnalysisOnly:        true,
		PhysicsClaim:           "none",
		Model:                  cfg.Model,
		Sites:                  cfg.Sites,
		HilbertDim:             1 << cfg.Sites,
		Boundary:               cfg.Boundary,
		Goal:                   "compute_bell_rates_and_verify_equivariance",
		GoalStatus:             goalStatus,
		Checks:                 checks,
		Metrics:                metrics,
		DebtStatus:             RequiredDebtStatus(),
		Limitations:            RequiredLimitations(),
		ForbiddenLanguageAudit: forbiddenAudit,
	}
}

// ReportJSON returns the JSON-encoded report with indentation.
func ReportJSON(r Report) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
