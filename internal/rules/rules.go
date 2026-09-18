package rules

import (
	"fmt"
	"sort"

	"github.com/Eliran-Turgeman/repear/internal/semantic"
)

type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Scope string

const (
	ScopeHunk  Scope = "hunk"
	ScopeFile  Scope = "file"
	ScopePatch Scope = "patch"
)

type Rule struct {
	ID                  string
	Description         string
	Instructions        string
	Message             string
	Scope               Scope
	DefaultThreshold    float64
	DefaultSeverity     Severity
	RequiresTaskContext bool
	Version             int
	Applicable          func(semantic.Unit, string) (bool, string)
}

var registry = map[string]Rule{
	"redundant-comment": {
		ID: "redundant-comment", Version: 1, Scope: ScopeHunk,
		Description:      "Detect comments that merely narrate adjacent code.",
		Message:          "Comment appears to merely narrate the adjacent implementation.",
		DefaultThreshold: 0.92, DefaultSeverity: SeverityWarning,
		Instructions: "Return the probability that a comment in the changed code merely restates what immediately adjacent code visibly does, without adding rationale, intent, constraints, invariants, domain information, warnings, or explanation of non-obvious behavior. Do not count public API documentation, required documentation comments, why-comments, security or concurrency rationale, compatibility constraints, links, or non-obvious domain explanations.",
		Applicable: func(u semantic.Unit, _ string) (bool, string) {
			if !u.ContainsComments {
				return false, "changed region contains no comments"
			}
			return true, ""
		},
	},
	"speculative-generality": {
		ID: "speculative-generality", Version: 1, Scope: ScopeHunk,
		Description:      "Detect unjustified abstraction or extensibility.",
		Message:          "New generality appears unjustified by the task or demonstrated usage.",
		DefaultThreshold: 0.88, DefaultSeverity: SeverityWarning,
		Instructions: "Return the probability that the change introduces abstraction, configurability, extensibility, indirection, generic infrastructure, or architectural machinery not justified by the task or demonstrated usage. Do not penalize abstraction itself, necessary boundaries, established patterns, or generality with current concrete consumers.",
		Applicable: func(u semantic.Unit, _ string) (bool, string) {
			if u.Additions < 4 {
				return false, "too few substantive additions"
			}
			return true, ""
		},
	},
	"defensive-fallback": {
		ID: "defensive-fallback", Version: 1, Scope: ScopeHunk,
		Description:      "Detect newly introduced fallbacks that hide failures.",
		Message:          "New fallback may hide an unexpected failure.",
		DefaultThreshold: 0.90, DefaultSeverity: SeverityError,
		Instructions: "Return the probability that newly introduced fallback behavior hides, suppresses, replaces, or silently recovers from a condition that should remain observable as an error, invalid state, or invariant violation. Do not count fallback behavior supported by the task, surrounding contract, established codebase behavior, or explicit resilience design.",
		Applicable: func(u semantic.Unit, _ string) (bool, string) {
			if u.Additions == 0 {
				return false, "changed region has no additions"
			}
			return true, ""
		},
	},
	"weakened-test": {
		ID: "weakened-test", Version: 1, Scope: ScopeHunk,
		Description:      "Detect test changes that reduce defect detection.",
		Message:          "Test modification may weaken behavioral protection.",
		DefaultThreshold: 0.92, DefaultSeverity: SeverityError,
		Instructions: "Return the probability that modification of an existing test reduces its ability to detect incorrect behavior instead of legitimately adapting to an intended behavior change. Consider removed assertions, broader checks, increased tolerances, removed error validation, or skipped behavior. Use old and new versions and task context. Do not count legitimate updates caused by intentional behavior changes.",
		Applicable: func(u semantic.Unit, _ string) (bool, string) {
			if !u.IsTest {
				return false, "not a test file"
			}
			if !u.ExistingModified {
				return false, "not a modified existing test"
			}
			return true, ""
		},
	},
	"scope-creep": {
		ID: "scope-creep", Version: 1, Scope: ScopePatch,
		Description:         "Detect substantial changes unrelated to the task.",
		Message:             "Change appears unrelated to the supplied task.",
		DefaultThreshold:    0.90,
		DefaultSeverity:     SeverityWarning,
		RequiresTaskContext: true,
		Instructions:        "Return the probability that the patch introduces externally observable behavior or substantial implementation changes unrelated to what is reasonably necessary for the supplied task. Do not count necessary small refactors, safety cleanup, required tests, or mechanical consequences of an API change.",
		Applicable: func(_ semantic.Unit, task string) (bool, string) {
			if task == "" {
				return false, "task context is required"
			}
			return true, ""
		},
	},
}

func All() []Rule {
	out := make([]Rule, 0, len(registry))
	for _, rule := range registry {
		out = append(out, rule)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func Get(id string) (Rule, bool) {
	rule, ok := registry[id]
	return rule, ok
}

func ValidateSeverity(value string) (Severity, error) {
	severity := Severity(value)
	if severity != SeverityWarning && severity != SeverityError {
		return "", fmt.Errorf("severity must be warning or error, got %q", value)
	}
	return severity, nil
}
