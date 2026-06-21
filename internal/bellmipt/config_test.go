package bellmipt

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultConfigValid verifies that DefaultConfig() passes validation.
func TestDefaultConfigValid(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DefaultConfig().Validate() returned error: %v", err)
	}

	// Spot-check a few fields.
	if cfg.Sites != 6 {
		t.Errorf("Sites = %d, want 6", cfg.Sites)
	}
	if cfg.Boundary != "periodic" {
		t.Errorf("Boundary = %q, want %q", cfg.Boundary, "periodic")
	}
	if cfg.Parameters.Delta != 0.5 {
		t.Errorf("Delta = %g, want 0.5", cfg.Parameters.Delta)
	}
}

// TestLoadConfig writes a temporary JSON config, loads it, and verifies fields.
func TestLoadConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Sites = 4
	cfg.Parameters.Mu = 2.0

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "test_config.json")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded, err := LoadConfig(tmp)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Sites != 4 {
		t.Errorf("Sites = %d, want 4", loaded.Sites)
	}
	if loaded.Parameters.Mu != 2.0 {
		t.Errorf("Mu = %g, want 2.0", loaded.Parameters.Mu)
	}
	if err := loaded.Validate(); err != nil {
		t.Errorf("loaded config failed validation: %v", err)
	}
}

// TestLoadConfigTestdata loads the small_default.json testdata fixture.
func TestLoadConfigTestdata(t *testing.T) {
	cfg, err := LoadConfig("testdata/small_default.json")
	if err != nil {
		t.Fatalf("LoadConfig testdata: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("testdata config failed validation: %v", err)
	}
	if cfg.Sites != 3 {
		t.Errorf("Sites = %d, want 3", cfg.Sites)
	}
	if cfg.Boundary != "open" {
		t.Errorf("Boundary = %q, want %q", cfg.Boundary, "open")
	}
	if cfg.InitialState.Seed != 42 {
		t.Errorf("Seed = %d, want 42", cfg.InitialState.Seed)
	}
	if math.Abs(cfg.Time.Dt-0.01) > 1e-15 {
		t.Errorf("Dt = %g, want 0.01", cfg.Time.Dt)
	}
	if cfg.Time.Steps != 100 {
		t.Errorf("Steps = %d, want 100", cfg.Time.Steps)
	}
}

// TestRejectUnsupportedModel checks that a config with an invalid model is rejected.
func TestRejectUnsupportedModel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Model = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for model=\"invalid\", got nil")
	}
}

// TestRejectUnsupportedBoundary checks that a config with an invalid boundary is rejected.
func TestRejectUnsupportedBoundary(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Boundary = "reflecting"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for boundary=\"reflecting\", got nil")
	}
}

// TestRejectUnsupportedInitialState checks that a config with an unsupported
// initial_state.type is rejected.
func TestRejectUnsupportedInitialState(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InitialState.Type = "ground_state"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for initial_state.type=\"ground_state\", got nil")
	}
}

// TestRejectTooManySites checks boundary conditions on the site count.
func TestRejectTooManySites(t *testing.T) {
	cfg := DefaultConfig()

	cfg.Sites = 11
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for sites=11, got nil")
	}

	cfg.Sites = 1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for sites=1, got nil")
	}

	// Boundary values that should pass.
	cfg.Sites = 2
	if err := cfg.Validate(); err != nil {
		t.Errorf("sites=2 should be valid, got: %v", err)
	}
	cfg.Sites = 10
	if err := cfg.Validate(); err != nil {
		t.Errorf("sites=10 should be valid, got: %v", err)
	}
}
