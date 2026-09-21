package diagnostics

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

type Diagnostic struct {
	ContextEvidence   string         `json:"context_evidence,omitempty"`
	Fingerprint       string         `json:"fingerprint"`
	SuppressionReason string         `json:"suppression_reason,omitempty"`
	Rule              string         `json:"rule"`
	Severity          rules.Severity `json:"severity"`
	Confidence        float64        `json:"confidence"`
	Signals           []SignalScore  `json:"signals"`
	Threshold         float64        `json:"threshold"`
	File              string         `json:"file"`
	StartLine         int            `json:"start_line"`
	EndLine           int            `json:"end_line"`
	Message           string         `json:"message"`
}

type SignalScore struct {
	Negated  bool    `json:"negated,omitempty"`
	ID       string  `json:"id"`
	Score    float64 `json:"score"`
	Evidence string  `json:"evidence"`
}

type Summary struct {
	Requests         int      `json:"requests"`
	Retries          int      `json:"retries"`
	InputTokens      int      `json:"input_tokens"`
	OutputTokens     int      `json:"output_tokens"`
	UsageResponses   int      `json:"usage_responses"`
	UsageComplete    bool     `json:"usage_complete"`
	DurationMS       float64  `json:"duration_ms"`
	CacheHitRate     float64  `json:"cache_hit_rate"`
	EstimatedCostUSD *float64 `json:"estimated_cost_usd"`
	Suppressed       int      `json:"suppressed"`
	Errors           int      `json:"errors"`
	Warnings         int      `json:"warnings"`
	UnitsEvaluated   int      `json:"units_evaluated"`
	UnitsSkipped     int      `json:"units_skipped,omitempty"`
	SemanticChecks   int      `json:"semantic_checks"`
	JevRequests      int      `json:"jev_requests"`
	CacheHits        int      `json:"cache_hits"`
}

type Report struct {
	Capabilities     *decision.Capabilities `json:"provider_capabilities,omitempty"`
	Suppressed       []Diagnostic           `json:"suppressed_findings"`
	Complete         bool                   `json:"complete"`
	Status           string                 `json:"status"`
	IncompletePolicy string                 `json:"incomplete_analysis"`
	Skipped          []SkippedUnit          `json:"skipped_units"`
	Version          int                    `json:"version"`
	Provider         string                 `json:"provider"`
	Model            string                 `json:"model"`
	Passed           bool                   `json:"passed"`
	Diagnostics      []Diagnostic           `json:"diagnostics"`
	Summary          Summary                `json:"summary"`
}

type SkippedUnit struct {
	File      string   `json:"file"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Rules     []string `json:"rules"`
	Reason    string   `json:"reason"`
}

func (r *Report) SetIncomplete(skipped []SkippedUnit, policy string) {
	r.IncompletePolicy = policy
	r.Skipped = skipped
	if r.Skipped == nil {
		r.Skipped = []SkippedUnit{}
	}
	r.Complete = len(skipped) == 0
	r.Passed = r.Complete && r.Summary.Errors == 0
	r.Status = "passed"
	if r.Summary.Errors > 0 {
		r.Status = "failed"
	}
	if !r.Complete {
		r.Status = "incomplete"
	}
}

func (r Report) ExitCode() int {
	if !r.Complete && r.IncompletePolicy != "warning" && r.IncompletePolicy != "ignore" {
		return 2
	}
	if r.Summary.Errors > 0 {
		return 1
	}
	return 0
}

func New(items []Diagnostic, summary Summary) Report {
	for i := range items {
		if items[i].Signals == nil {
			items[i].Signals = []SignalScore{}
		}
		sort.Slice(items[i].Signals, func(a, b int) bool { return items[i].Signals[a].ID < items[i].Signals[b].ID })
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].File != items[j].File {
			return items[i].File < items[j].File
		}
		if items[i].StartLine != items[j].StartLine {
			return items[i].StartLine < items[j].StartLine
		}
		return items[i].Rule < items[j].Rule
	})
	summary.Errors = 0
	summary.Warnings = 0
	for _, item := range items {
		if item.Severity == rules.SeverityError {
			summary.Errors++
		} else {
			summary.Warnings++
		}
	}
	if items == nil {
		items = []Diagnostic{}
	}
	r := Report{Version: 2, Diagnostics: items, Summary: summary}
	r.SetIncomplete(nil, "error")
	return r
}

func WriteJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func WriteText(w io.Writer, report Report) error {
	if !report.Complete {
		if _, err := fmt.Fprintf(w, "Analysis incomplete: %d units skipped (policy=%s).\n", len(report.Skipped), report.IncompletePolicy); err != nil {
			return err
		}
		for _, skipped := range report.Skipped {
			if _, err := fmt.Fprintf(w, "  %s:%d: %s\n", skipped.File, skipped.StartLine, skipped.Reason); err != nil {
				return err
			}
		}
	}
	for _, item := range report.Diagnostics {
		if _, err := fmt.Fprintf(w, "%s:%d-%d: %s [%s] confidence=%.2f\n  %s\n\n",
			item.File, item.StartLine, item.EndLine, item.Severity, item.Rule,
			item.Confidence, item.Message); err != nil {
			return err
		}
		for _, signal := range item.Signals {
			if _, err := fmt.Fprintf(w, "  %s=%.2f\n", signal.ID, signal.Score); err != nil {
				return err
			}
		}
	}
	total := len(report.Diagnostics)
	if total == 0 {
		if !report.Complete {
			_, err := fmt.Fprintln(w, "No violations found in the completed evaluations; remaining code was not fully evaluated.")
			return err
		}
		_, err := fmt.Fprintln(w, "No blocking semantic violations found.")
		return err
	}
	_, err := fmt.Fprintf(w, "Found %d semantic violations:\n  %d errors\n  %d warnings\n",
		total, report.Summary.Errors, report.Summary.Warnings)
	return err
}
