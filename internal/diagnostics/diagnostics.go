package diagnostics

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/Eliran-Turgeman/repear/internal/rules"
)

type Diagnostic struct {
	Rule        string         `json:"rule"`
	Severity    rules.Severity `json:"severity"`
	Probability float64        `json:"probability"`
	Threshold   float64        `json:"threshold"`
	File        string         `json:"file"`
	StartLine   int            `json:"start_line"`
	EndLine     int            `json:"end_line"`
	Message     string         `json:"message"`
}

type Summary struct {
	Errors         int `json:"errors"`
	Warnings       int `json:"warnings"`
	UnitsEvaluated int `json:"units_evaluated"`
	SemanticChecks int `json:"semantic_checks"`
	JevRequests    int `json:"jev_requests"`
	CacheHits      int `json:"cache_hits"`
}

type Report struct {
	Version     int          `json:"version"`
	Provider    string       `json:"provider"`
	Model       string       `json:"model"`
	Passed      bool         `json:"passed"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Summary     Summary      `json:"summary"`
}

func New(items []Diagnostic, summary Summary) Report {
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
	return Report{Version: 1, Passed: summary.Errors == 0, Diagnostics: items, Summary: summary}
}

func WriteJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func WriteText(w io.Writer, report Report) error {
	for _, item := range report.Diagnostics {
		if _, err := fmt.Fprintf(w, "%s:%d-%d: %s [%s] confidence=%.2f\n  %s\n\n",
			item.File, item.StartLine, item.EndLine, item.Severity, item.Rule,
			item.Probability, item.Message); err != nil {
			return err
		}
	}
	total := len(report.Diagnostics)
	if total == 0 {
		_, err := fmt.Fprintln(w, "No blocking semantic violations found.")
		return err
	}
	_, err := fmt.Fprintf(w, "Found %d semantic violations:\n  %d errors\n  %d warnings\n",
		total, report.Summary.Errors, report.Summary.Warnings)
	return err
}
