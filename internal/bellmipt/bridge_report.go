package bellmipt

// BridgeWarning represents a warning emitted during the bridge audit.
type BridgeWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BridgeInterpretation explains the descriptive finding regarding conditional update correlation.
type BridgeInterpretation struct {
	EnvironmentCorrelatedConditionalUpdate string `json:"environment_correlated_conditional_update"`
	Reason                                 string `json:"reason"`
}

// BridgeReport encapsulates all findings of the optional bridge audit.
type BridgeReport struct {
	Enabled      bool   `json:"enabled"`
	BridgeGoal   string `json:"bridge_goal,omitempty"`
	BridgeStatus string `json:"bridge_status,omitempty"`

	SubsystemSitesRequested   []int `json:"subsystem_sites_requested,omitempty"`
	EnvironmentSitesRequested []int `json:"environment_sites_requested,omitempty"`
	SubsystemSitesCanonical   []int `json:"subsystem_sites_canonical,omitempty"`
	EnvironmentSitesCanonical []int `json:"environment_sites_canonical,omitempty"`

	Trajectories     int `json:"trajectories,omitempty"`
	SampleEverySteps int `json:"sample_every_steps,omitempty"`

	Metrics        *BridgeMetrics        `json:"metrics,omitempty"`
	Interpretation *BridgeInterpretation `json:"interpretation,omitempty"`
	Warnings       []BridgeWarning       `json:"warnings,omitempty"`
}

// InterpretConditionalUpdate evaluates the primary conditional update ratio and assigns a descriptive category.
func InterpretConditionalUpdate(ratio *float64, status string) BridgeInterpretation {
	reason := "Descriptive finite-toy diagnostic only. This does not establish a Bell-MIPT bridge."

	if status != "bridge_audit_completed" || ratio == nil {
		return BridgeInterpretation{
			EnvironmentCorrelatedConditionalUpdate: "not_assessed",
			Reason:                                 reason,
		}
	}

	val := *ratio
	var category string
	if val < 1.5 {
		category = "no_clear_correlation"
	} else if val < 3.0 {
		category = "weak_correlation"
	} else {
		category = "candidate_correlation"
	}

	return BridgeInterpretation{
		EnvironmentCorrelatedConditionalUpdate: category,
		Reason:                                 reason,
	}
}
