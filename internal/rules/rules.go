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
		Applicable:   alwaysApplicable,
	},
	"pass-through-layer": {
		ID: "pass-through-layer", Version: 1, Scope: ScopeHunk,
		Description:      "Detect layers that add indirection without a new abstraction.",
		Message:          "New layer appears to forward behavior without hiding meaningful complexity.",
		DefaultThreshold: 0.91, DefaultSeverity: SeverityWarning,
		Instructions: "Return the probability that the change introduces a method, class, wrapper, service, adapter, or architectural layer that mostly forwards the same operation and arguments to another layer without simplifying the interface, enforcing meaningful policy, translating representations, combining operations, or hiding substantial complexity. Do not count legitimate adapters, stable boundaries, dependency inversion with current consumers, authorization or validation layers, transaction boundaries, observability wrappers, or delegation that provides a meaningfully different abstraction.",
		Applicable:   alwaysApplicable,
	},
	"complexity-pushed-upward": {
		ID: "complexity-pushed-upward", Version: 1, Scope: ScopeHunk,
		Description:      "Detect implementation complexity pushed onto callers.",
		Message:          "Change appears to make callers manage complexity the module could own.",
		DefaultThreshold: 0.88, DefaultSeverity: SeverityWarning,
		Instructions: "Return the probability that the change makes callers, users, or higher-level modules understand or manage complexity that the changed module can reasonably handle itself. Consider new configuration knobs, boolean mode flags, lifecycle sequencing, cleanup duties, retry details, internal error distinctions, representation details, or repeated setup required from callers. Do not count choices that are genuine product policy, security decisions, resource ownership boundaries, required caller data, or variability with multiple current use cases. Prefer a simple interface even when that makes the implementation more complex.",
		Applicable:   alwaysApplicable,
	},
	"information-leakage": {
		ID: "information-leakage", Version: 1, Scope: ScopeHunk,
		Description:      "Detect internal design knowledge exposed across module boundaries.",
		Message:          "Change appears to expose an internal design decision to other modules.",
		DefaultThreshold: 0.89, DefaultSeverity: SeverityWarning,
		Instructions: "Return the probability that the change exposes internal implementation knowledge, representation, storage layout, protocol encoding, cache structure, dependency-specific type, or other design decision through a module boundary, causing consumers to depend on information that should remain hidden. Include back-door leakage through required call sequences or duplicated assumptions, not only public type signatures. Do not count intentional public contracts, standard interoperable formats, diagnostic or introspection APIs, dependency types that are the module's purpose, or representations callers genuinely need.",
		Applicable:   alwaysApplicable,
	},
	"special-general-mixture": {
		ID: "special-general-mixture", Version: 1, Scope: ScopeHunk,
		Description:      "Detect special-purpose policy embedded in general mechanisms.",
		Message:          "Special-case policy appears mixed into a general-purpose mechanism.",
		DefaultThreshold: 0.90, DefaultSeverity: SeverityWarning,
		Instructions: "Return the probability that the change embeds customer-specific, endpoint-specific, workflow-specific, product-specific, or one-off policy inside a general-purpose reusable mechanism, instead of keeping the special policy in its owning layer and the mechanism broadly applicable. Look for identity checks, named exceptions, domain branching, or special flags in generic infrastructure. Do not count behavior fundamental to the abstraction, protocol-required cases, security boundaries, compatibility handling localized at the boundary, or explicit policy hooks supplied by higher layers.",
		Applicable:   alwaysApplicable,
	},
	"shallow-module": {
		ID: "shallow-module", Version: 1, Scope: ScopeHunk,
		Description:      "Detect abstractions with costly interfaces and little hidden complexity.",
		Message:          "New abstraction appears shallow relative to the interface it introduces.",
		DefaultThreshold: 0.90, DefaultSeverity: SeverityWarning,
		Instructions: "Return the probability that the change introduces an abstraction whose interface complexity is not substantially lower than the implementation complexity it hides or the functionality it provides. Consider classes, interfaces, methods, option objects, builders, and wrappers that require significant learning, configuration, or ceremony for little benefit. Judge depth rather than raw code size. Do not count small well-named helpers that improve readability, narrow domain types that enforce invariants, adapters that hide difficult dependencies, or simple interfaces backed by meaningful policy or implementation complexity.",
		Applicable:   alwaysApplicable,
	},
	"defensive-fallback": {
		ID: "defensive-fallback", Version: 1, Scope: ScopeHunk,
		Description:      "Detect newly introduced fallbacks that hide failures.",
		Message:          "New fallback may hide an unexpected failure.",
		DefaultThreshold: 0.90, DefaultSeverity: SeverityError,
		Instructions: "Return the probability that newly introduced fallback behavior hides, suppresses, replaces, or silently recovers from a condition that should remain observable as an error, invalid state, or invariant violation. Do not count fallback behavior supported by the task, surrounding contract, established codebase behavior, or explicit resilience design.",
		Applicable:   alwaysApplicable,
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

func alwaysApplicable(semantic.Unit, string) (bool, string) {
	return true, ""
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
