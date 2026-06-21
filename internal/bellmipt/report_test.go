package bellmipt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeterministicOutputFixedSeed(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg1.Sites = 3
	cfg1.Time.Dt = 0.01
	cfg1.Time.Steps = 50

	tmpDir1, err := os.MkdirTemp("", "bellmipt-test-det1-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir1)

	res1 := Run(cfg1, tmpDir1)
	if res1.Error != nil {
		t.Fatalf("run 1 failed: %v", res1.Error)
	}

	cfg2 := DefaultConfig()
	cfg2.Sites = 3
	cfg2.Time.Dt = 0.01
	cfg2.Time.Steps = 50

	tmpDir2, err := os.MkdirTemp("", "bellmipt-test-det2-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir2)

	res2 := Run(cfg2, tmpDir2)
	if res2.Error != nil {
		t.Fatalf("run 2 failed: %v", res2.Error)
	}

	if res1.Report.Metrics.MaxEquivarianceL1Error != res2.Report.Metrics.MaxEquivarianceL1Error {
		t.Errorf("determinism check failed: metric MaxEquivarianceL1Error got %e and %e",
			res1.Report.Metrics.MaxEquivarianceL1Error, res2.Report.Metrics.MaxEquivarianceL1Error)
	}

	if res1.Report.Metrics.MaxHermitianError != res2.Report.Metrics.MaxHermitianError {
		t.Errorf("determinism check failed: metric MaxHermitianError got %e and %e",
			res1.Report.Metrics.MaxHermitianError, res2.Report.Metrics.MaxHermitianError)
	}
}

func TestReportContainsRequiredDebtStatus(t *testing.T) {
	required := RequiredDebtStatus()
	cfg := DefaultConfig()
	cfg.Sites = 2
	cfg.Time.Steps = 1

	tmpDir, err := os.MkdirTemp("", "bellmipt-test-debt-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	res := Run(cfg, tmpDir)
	if res.Error != nil {
		t.Fatal(res.Error)
	}

	for k, expectedVal := range required {
		val, exists := res.Report.DebtStatus[k]
		if !exists {
			t.Errorf("missing required debt status key %q", k)
		} else if val != expectedVal {
			t.Errorf("expected debt status value %q for key %q, got %q", expectedVal, k, val)
		}
	}
}

func TestReportContainsRequiredLimitations(t *testing.T) {
	required := RequiredLimitations()
	cfg := DefaultConfig()
	cfg.Sites = 2
	cfg.Time.Steps = 1

	tmpDir, err := os.MkdirTemp("", "bellmipt-test-lim-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	res := Run(cfg, tmpDir)
	if res.Error != nil {
		t.Fatal(res.Error)
	}

	if len(res.Report.Limitations) != len(required) {
		t.Errorf("expected %d limitations, got %d", len(required), len(res.Report.Limitations))
	}

	for i, expected := range required {
		if i < len(res.Report.Limitations) && res.Report.Limitations[i] != expected {
			t.Errorf("expected limitation %d to be %q, got %q", i, expected, res.Report.Limitations[i])
		}
	}
}

func TestNoForbiddenPromotionLanguage(t *testing.T) {
	// 1. Text with allowed negation sentences should pass
	textSafe := `
This checks Bell-rate algebra in a finite toy model only.
This is not a physics promotion.
This does not show Bell jumps are measurements.
No MIPT claim.
No holography claim.
No physics promotion.
No Bell-jumps-equal-measurements claim.
This does not implement MIPT.
This does not support any holography or black-hole claim.
toy_goal_passed, toy analysis, numerical check, equivariance audit, rate algebra
`
	auditSafe := AuditForbiddenLanguage(textSafe)
	if !auditSafe.Passed {
		t.Errorf("expected safe text to pass forbidden language audit, but got hits: %v", auditSafe.Hits)
	}

	// 2. Text with forbidden phrase should fail
	textForbidden := `
This establishes MIPT on a large scale.
`
	auditForbidden := AuditForbiddenLanguage(textForbidden)
	if auditForbidden.Passed {
		t.Errorf("expected text with forbidden phrase to fail, but it passed")
	}
	if len(auditForbidden.Hits) != 1 || auditForbidden.Hits[0] != "establishes mipt" {
		t.Errorf("expected hit for 'establishes mipt', got: %v", auditForbidden.Hits)
	}
}

func TestDefaultRunWritesInputReportJSONAndMarkdown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Sites = 3
	cfg.Time.Steps = 10

	tmpDir, err := os.MkdirTemp("", "bellmipt-test-run-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	res := Run(cfg, tmpDir)
	if res.Error != nil {
		t.Fatalf("Run failed: %v", res.Error)
	}

	if res.Report.GoalStatus != "toy_goal_passed" {
		t.Errorf("expected goal status toy_goal_passed, got %s", res.Report.GoalStatus)
	}

	// Check files exist
	files := []string{"input.json", "report.json", "report.md"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("file %s was not written", file)
			continue
		}
		if info.IsDir() {
			t.Errorf("%s is a directory, expected file", file)
		}
	}

	// Check that we can unmarshal the JSON output
	reportJSON, err := os.ReadFile(filepath.Join(tmpDir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var loadedReport Report
	if err := json.Unmarshal(reportJSON, &loadedReport); err != nil {
		t.Errorf("failed to unmarshal report.json: %v", err)
	}

	if loadedReport.ToyID != "BELL-MIPT-0001" {
		t.Errorf("loaded report has invalid toy_id: %s", loadedReport.ToyID)
	}

	// Check that markdown contains status and scope
	reportMD, err := os.ReadFile(filepath.Join(tmpDir, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	mdContent := string(reportMD)
	if !strings.Contains(mdContent, "Status") || !strings.Contains(mdContent, "Scope") {
		t.Errorf("report.md does not contain required headers")
	}
}
