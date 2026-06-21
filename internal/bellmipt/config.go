// Package bellmipt implements Bell jump-rate computation for finite Kitaev chains
// and verification of equivariance under the Bell measurement protocol.
//
// Pairing sign convention: +Δ (positive pairing amplitude).
package bellmipt

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds the full simulation configuration, read from JSON.
type Config struct {
	SchemaVersion string        `json:"schema_version"`
	Model         string        `json:"model"`
	Sites         int           `json:"sites"`
	Boundary      string        `json:"boundary"`
	Parameters    Parameters    `json:"parameters"`
	InitialState  InitialState  `json:"initial_state"`
	Time          TimeConfig    `json:"time"`
	Audit         AuditConfig   `json:"audit"`
	Bridge        *BridgeConfig `json:"bridge,omitempty"`
}

// Parameters holds the Hamiltonian couplings for the Kitaev chain.
type Parameters struct {
	Mu    float64 `json:"mu"`    // chemical potential
	T     float64 `json:"t"`     // hopping amplitude
	Delta float64 `json:"delta"` // pairing amplitude (+Δ convention)
}

// InitialState specifies how the initial state vector is prepared.
type InitialState struct {
	Type string `json:"type"` // e.g. "random_normalized"
	Seed int64  `json:"seed"` // RNG seed for reproducibility
}

// TimeConfig specifies the time-stepping parameters for the RK4 integrator.
type TimeConfig struct {
	Dt    float64 `json:"dt"`    // time step
	Steps int     `json:"steps"` // number of integration steps
}

// AuditConfig specifies numerical tolerances for runtime sanity checks.
type AuditConfig struct {
	HermitianTolerance    float64 `json:"hermitian_tolerance"`
	NormTolerance         float64 `json:"norm_tolerance"`
	EquivarianceTolerance float64 `json:"equivariance_tolerance"`
}

// DefaultConfig returns the canonical 6-site periodic Kitaev chain configuration.
func DefaultConfig() Config {
	return Config{
		SchemaVersion: "bell_mipt_toy_v0",
		Model:         "finite_kitaev_chain",
		Sites:         6,
		Boundary:      "periodic",
		Parameters: Parameters{
			Mu:    1.0,
			T:     1.0,
			Delta: 0.5,
		},
		InitialState: InitialState{
			Type: "random_normalized",
			Seed: 12345,
		},
		Time: TimeConfig{
			Dt:    0.001,
			Steps: 1000,
		},
		Audit: AuditConfig{
			HermitianTolerance:    1e-10,
			NormTolerance:         1e-8,
			EquivarianceTolerance: 1e-5,
		},
	}
}

// LoadConfig reads a JSON configuration file from path and returns a Config.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("bellmipt: read config %q: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("bellmipt: parse config %q: %w", path, err)
	}
	return cfg, nil
}

// Validate checks that all Config fields have acceptable values.
// It returns an error describing the first invalid field found.
func (c Config) Validate() error {
	if c.SchemaVersion != "bell_mipt_toy_v0" {
		return fmt.Errorf("bellmipt: unsupported schema_version %q (want %q)",
			c.SchemaVersion, "bell_mipt_toy_v0")
	}
	if c.Model != "finite_kitaev_chain" {
		return fmt.Errorf("bellmipt: unsupported model %q (want %q)",
			c.Model, "finite_kitaev_chain")
	}
	if c.Boundary != "open" && c.Boundary != "periodic" {
		return fmt.Errorf("bellmipt: unsupported boundary %q (want \"open\" or \"periodic\")",
			c.Boundary)
	}
	if c.InitialState.Type != "random_normalized" {
		return fmt.Errorf("bellmipt: unsupported initial_state.type %q (want %q)",
			c.InitialState.Type, "random_normalized")
	}
	if c.Sites < 2 || c.Sites > 10 {
		return fmt.Errorf("bellmipt: sites=%d out of range [2, 10]", c.Sites)
	}
	if c.Time.Dt <= 0 {
		return fmt.Errorf("bellmipt: dt must be > 0, got %g", c.Time.Dt)
	}
	if c.Time.Steps <= 0 {
		return fmt.Errorf("bellmipt: steps must be > 0, got %d", c.Time.Steps)
	}
	if c.Audit.HermitianTolerance <= 0 {
		return fmt.Errorf("bellmipt: hermitian_tolerance must be > 0, got %g",
			c.Audit.HermitianTolerance)
	}
	if c.Audit.NormTolerance <= 0 {
		return fmt.Errorf("bellmipt: norm_tolerance must be > 0, got %g",
			c.Audit.NormTolerance)
	}
	if c.Audit.EquivarianceTolerance <= 0 {
		return fmt.Errorf("bellmipt: equivariance_tolerance must be > 0, got %g",
			c.Audit.EquivarianceTolerance)
	}

	if c.Bridge != nil {
		if err := c.Bridge.Validate(c.Sites); err != nil {
			return err
		}
	}

	return nil
}
