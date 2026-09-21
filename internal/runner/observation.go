package runner

import "github.com/Eliran-Turgeman/reaper/internal/diagnostics"

type PermissionObservation struct {
	Signal  string                        `json:"signal,omitempty"`
	Signals []PermissionSignalObservation `json:"signals,omitempty"`
	Score   float64                       `json:"score"`
	Outcome string                        `json:"outcome"`
}

type PermissionSignalObservation struct {
	Signal string  `json:"signal"`
	Score  float64 `json:"score"`
}

// Observation records a rule evaluation whether or not it produces a finding.
// Observe callbacks run serially after evaluation, in stable job/rule order.
type Observation struct {
	Composition     string                    `json:"composition,omitempty"`
	Decision        string                    `json:"decision,omitempty"`
	EvidenceStatus  string                    `json:"evidence_status,omitempty"`
	EvidenceReasons []string                  `json:"evidence_reasons,omitempty"`
	Permission      *PermissionObservation    `json:"permission,omitempty"`
	RuleVersion     int                       `json:"rule_version,omitempty"`
	Rule            string                    `json:"rule"`
	File            string                    `json:"file"`
	StartLine       int                       `json:"start_line"`
	EndLine         int                       `json:"end_line"`
	Score           float64                   `json:"score"`
	Signals         []diagnostics.SignalScore `json:"signals"`
}
