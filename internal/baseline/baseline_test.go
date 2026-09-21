package baseline

import (
	"encoding/json"
	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"os"
	"path/filepath"
	"testing"
)

func TestBaselineMovementAndNewFindings(t *testing.T) {
	dir := t.TempDir()
	d := diagnostics.Diagnostic{Rule: "silent-failure-fallback", File: "x.go", StartLine: 4, EndLine: 5, Fingerprint: "stable", Severity: rules.SeverityError}
	report := diagnostics.New([]diagnostics.Diagnostic{d}, diagnostics.Summary{})
	file, err := Create(report, "legacy behavior")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(file)
	path := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	d.StartLine, d.EndLine = 40, 41
	newFinding := d
	newFinding.Fingerprint = "new"
	result, err := Apply(dir, path, diagnostics.New([]diagnostics.Diagnostic{d, newFinding}, diagnostics.Summary{}))
	if err != nil || len(result.Diagnostics) != 1 || result.Summary.Suppressed != 1 || result.ExitCode() != 1 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestInlineSuppressionNeedsReason(t *testing.T) {
	dir := t.TempDir()
	for _, reason := range []string{"", " -- compatibility contract"} {
		if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("// reaper: ignore silent-failure-fallback"+reason+"\nreturn nil\n"), 0600); err != nil {
			t.Fatal(err)
		}
		report := diagnostics.New([]diagnostics.Diagnostic{{Rule: "silent-failure-fallback", File: "x.go", StartLine: 2, EndLine: 2, Severity: rules.SeverityError}}, diagnostics.Summary{})
		result, err := Apply(dir, "", report)
		if reason == "" {
			if err == nil {
				t.Fatal("accepted reasonless suppression")
			}
		} else if err != nil || result.Summary.Suppressed != 1 || result.ExitCode() != 0 {
			t.Fatalf("%+v %v", result, err)
		}
	}
}
