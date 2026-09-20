package runner

import "github.com/Eliran-Turgeman/reaper/internal/diagnostics"

// Observation records a rule evaluation whether or not it produces a finding.
// Observe callbacks run serially after evaluation, in stable job/rule order.
type Observation struct {
	Rule      string                    `json:"rule"`
	File      string                    `json:"file"`
	StartLine int                       `json:"start_line"`
	EndLine   int                       `json:"end_line"`
	Score     float64                   `json:"score"`
	Signals   []diagnostics.SignalScore `json:"signals"`
}
