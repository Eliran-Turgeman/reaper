package feedback

import (
	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	"path/filepath"
	"testing"
)

func TestFeedbackUpdatesWithoutDoubleCounting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feedback.jsonl")
	report := diagnostics.New([]diagnostics.Diagnostic{{Rule: "r", Fingerprint: "id"}}, diagnostics.Summary{})
	if err := Record(path, report, "id", "useful"); err != nil {
		t.Fatal(err)
	}
	if err := Record(path, report, "id", "false-positive"); err != nil {
		t.Fatal(err)
	}
	metrics, err := Metrics(path, report)
	if err != nil || len(metrics) != 1 || metrics[0].Findings != 1 || metrics[0].Accepted != 0 || metrics[0].FalsePositive != 1 {
		t.Fatal(metrics, err)
	}
	if err := Record(path, report, "missing", "useful"); err == nil {
		t.Fatal("unknown ID accepted")
	}
}
