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
	ScopePatch Scope = "patch"
)

type Signal struct {
	ID           string
	Instructions string
}

type Rule struct {
	ID                  string
	Description         string
	Signals             []Signal
	Message             string
	Scope               Scope
	DefaultThreshold    float64
	DefaultSeverity     Severity
	RequiresTaskContext bool
	AuditSkipReason     string
	Version             int
	Applicable          func(semantic.Unit, string) (bool, string)
}

func (r Rule) QuestionID(signal Signal) string {
	return r.ID + ":" + signal.ID
}

func (r Rule) Compose(probabilities map[string]float64) (float64, bool) {
	if len(r.Signals) == 0 {
		return 0, false
	}
	score := 1.0
	for _, signal := range r.Signals {
		value, ok := probabilities[r.QuestionID(signal)]
		if !ok {
			return 0, false
		}
		if value < score {
			score = value
		}
	}
	return score, true
}

var registry = map[string]Rule{
	"narrating-comment": {
		ID: "narrating-comment", Version: 1, Scope: ScopeHunk,
		Description:      "Detect changed comments that merely narrate adjacent code.",
		Message:          "Comment appears to merely narrate the adjacent implementation.",
		DefaultThreshold: 0.92, DefaultSeverity: SeverityWarning,
		Signals: []Signal{{
			ID:           "restates-visible-code",
			Instructions: "Return the probability that a comment added or modified in the changed code merely restates what the immediately adjacent code visibly does. Judge only the supplied changed region. Do not count public API documentation, required documentation comments, rationale, intent, constraints, invariants, domain information, warnings, compatibility notes, links, or explanations of non-obvious behavior.",
		}},
		Applicable: func(u semantic.Unit, _ string) (bool, string) {
			if !u.ContainsComments {
				return false, "changed region contains no comments"
			}
			return true, ""
		},
	},
	"unchanged-argument-forwarder": {
		ID: "unchanged-argument-forwarder", Version: 1, Scope: ScopeHunk,
		Description:      "Detect newly introduced callables that only forward an operation.",
		Message:          "New callable appears to forward the same operation and arguments without visible added behavior.",
		DefaultThreshold: 0.90, DefaultSeverity: SeverityWarning,
		Signals: []Signal{
			{
				ID:           "introduces-forwarder",
				Instructions: "Return the probability that the changed region introduces or materially changes a function, method, or callable whose visible implementation primarily invokes one other callable and returns or relays its result.",
			},
			{
				ID:           "preserves-operation-and-arguments",
				Instructions: "Return the probability that the visible forwarding callable exposes substantially the same operation and passes substantially the same arguments through unchanged.",
			},
			{
				ID:           "adds-no-visible-behavior",
				Instructions: "Return the probability that the visible forwarding callable adds no meaningful validation, authorization, policy, translation, composition, transaction handling, lifecycle management, observability, compatibility behavior, or error handling. Judge only behavior visible in the supplied region; do not decide whether the callable is architecturally justified elsewhere.",
			},
		},
		Applicable: alwaysApplicable,
	},
	"ceremonial-abstraction": {
		ID: "ceremonial-abstraction", Version: 1, Scope: ScopeHunk,
		Description:      "Detect new abstractions with visible ceremony and little local behavior.",
		Message:          "New abstraction adds visible ceremony while hiding little behavior in the supplied region.",
		DefaultThreshold: 0.90, DefaultSeverity: SeverityWarning,
		Signals: []Signal{
			{
				ID:           "introduces-abstraction",
				Instructions: "Return the probability that the changed region introduces a class, interface, wrapper, service, builder, request object, option object, operation object, or similarly named abstraction.",
			},
			{
				ID:           "adds-interface-ceremony",
				Instructions: "Return the probability that using the introduced abstraction visibly requires learning or invoking additional names, methods, construction steps, options, or protocol.",
			},
			{
				ID:           "hides-little-visible-behavior",
				Instructions: "Return the probability that the abstraction's visible implementation provides only trivial calculation, assignment, lookup, forwarding, or similarly small behavior without visible invariant enforcement, policy, translation, difficult dependency handling, or meaningful state management. Judge only the supplied region and do not decide repository-wide architectural value.",
			},
		},
		Applicable: alwaysApplicable,
	},
	"boolean-mode-parameter": {
		ID: "boolean-mode-parameter", Version: 1, Scope: ScopeHunk,
		Description:      "Detect new boolean parameters that select substantially different behavior.",
		Message:          "New boolean parameter appears to make callers select an internal behavior mode.",
		DefaultThreshold: 0.90, DefaultSeverity: SeverityWarning,
		Signals: []Signal{
			{
				ID:           "introduces-boolean-input",
				Instructions: "Return the probability that the changed region introduces a boolean parameter, option, or flag that callers must supply.",
			},
			{
				ID:           "selects-behavior-mode",
				Instructions: "Return the probability that the boolean visibly selects between meaningfully different execution modes rather than representing genuine boolean domain data. Do not count clear product policy, security choices, feature state, or naturally boolean values.",
			},
		},
		Applicable: alwaysApplicable,
	},
	"caller-managed-mechanics": {
		ID: "caller-managed-mechanics", Version: 1, Scope: ScopeHunk,
		Description:      "Detect low-level operational mechanics newly required from callers.",
		Message:          "Change appears to require callers to manage visible low-level operational mechanics.",
		DefaultThreshold: 0.90, DefaultSeverity: SeverityWarning,
		Signals: []Signal{
			{
				ID:           "introduces-caller-duty",
				Instructions: "Return the probability that the changed region requires callers to choose or perform visible low-level configuration, setup ordering, multi-step sequencing, cleanup, retry, serialization, internal error translation, representation filtering, or state-transition duties.",
			},
			{
				ID:           "duty-is-mechanical",
				Instructions: "Return the probability that the visible duty concerns mechanical operation of a lower-level component rather than caller-owned product policy, required caller data, explicit resource ownership, cross-module coordination, observable data semantics, or a security decision.",
			},
		},
		Applicable: alwaysApplicable,
	},
	"implementation-detail-exposure": {
		ID: "implementation-detail-exposure", Version: 1, Scope: ScopeHunk,
		Description:      "Detect implementation-specific details added to public boundaries.",
		Message:          "Changed public boundary appears to expose a visible implementation detail.",
		DefaultThreshold: 0.91, DefaultSeverity: SeverityWarning,
		Signals: []Signal{
			{
				ID:           "changes-external-boundary",
				Instructions: "Return the probability that the changed region adds or modifies a type, parameter, return value, field, key, exception, method, or required call sequence visibly exposed outside its immediate implementation.",
			},
			{
				ID:           "exposes-implementation-detail",
				Instructions: "Return the probability that the boundary visibly exposes a concrete dependency type, storage layout, mutable internal state, cache key, serialized bytes, protocol encoding, database row, process detail, internal identifier, algorithm phase, or other implementation-specific representation. Do not count intentional public contracts, standard interoperable formats, diagnostics, dependencies that are the module's purpose, or information callers visibly need.",
			},
		},
		Applicable: alwaysApplicable,
	},
	"named-special-case": {
		ID: "named-special-case", Version: 1, Scope: ScopeHunk,
		Description:      "Detect named product or workflow exceptions in reusable mechanisms.",
		Message:          "Reusable-looking mechanism contains a visible named special case.",
		DefaultThreshold: 0.91, DefaultSeverity: SeverityWarning,
		Signals: []Signal{
			{
				ID:           "changes-reusable-mechanism",
				Instructions: "Return the probability that the changed region is visibly part of a reusable, generic, shared, framework, infrastructure, utility, or mechanism-level component.",
			},
			{
				ID:           "branches-on-named-policy",
				Instructions: "Return the probability that the mechanism visibly branches on a named customer, endpoint, workflow, product, tenant, or one-off case. Do not count protocol-required cases, security boundaries, compatibility handling at a boundary, or behavior fundamental to the mechanism.",
			},
		},
		Applicable: alwaysApplicable,
	},
	"unused-extensibility-point": {
		ID: "unused-extensibility-point", Version: 1, Scope: ScopeHunk,
		Description:      "Detect extensibility machinery with no concrete use visible in the patch.",
		Message:          "New extensibility point has no distinct concrete use visible in the supplied change.",
		DefaultThreshold: 0.91, DefaultSeverity: SeverityWarning,
		Signals: []Signal{
			{
				ID:           "introduces-extensibility",
				Instructions: "Return the probability that the changed region introduces an interface, callback, hook, plugin point, generic parameter, strategy, registry, configurable factory, or similar extensibility mechanism.",
			},
			{
				ID:           "only-one-visible-behavior",
				Instructions: "Return the probability that the supplied patch context shows only one concrete implementation, registration, instantiation, or behavior for the new extensibility mechanism.",
			},
			{
				ID:           "task-does-not-request-extensibility",
				Instructions: "Return the probability that the supplied task, when present, does not explicitly require multiple implementations, configurability, plugins, substitution, testing seams, or another concrete extensibility need. If no task is supplied, return 0 rather than assuming the need is absent.",
			},
		},
		Applicable: func(_ semantic.Unit, task string) (bool, string) {
			if task == "" {
				return false, "task context is required for a local claim about unused extensibility"
			}
			return true, ""
		},
	},
	"silent-failure-fallback": {
		ID: "silent-failure-fallback", Version: 1, Scope: ScopeHunk,
		Description:      "Detect changed code that turns visible failures into success or defaults.",
		Message:          "Changed code appears to replace a visible failure with success or a default value.",
		DefaultThreshold: 0.92, DefaultSeverity: SeverityError,
		Signals: []Signal{
			{
				ID:           "intercepts-failure",
				Instructions: "Return the probability that the changed region catches, checks, discards, suppresses, or otherwise intercepts an error, exception, invalid value, failed parse, unknown case, or invariant violation.",
			},
			{
				ID:           "returns-success-or-default",
				Instructions: "Return the probability that the intercepted failure is visibly replaced with success, nil, zero, empty data, a default object, stale data, or continued execution without making the failure observable. Do not count explicit resilience behavior required by the task or visibly established contract.",
			},
		},
		Applicable: alwaysApplicable,
	},
	"weakened-test-assertion": {
		ID: "weakened-test-assertion", Version: 1, Scope: ScopeHunk,
		Description:      "Detect locally visible removal or broadening of test protection.",
		Message:          "Test change visibly removes, skips, or broadens an existing behavioral check.",
		DefaultThreshold: 0.93, DefaultSeverity: SeverityError,
		AuditSkipReason: "requires a before-and-after test change",
		Signals: []Signal{{
			ID:           "reduces-visible-protection",
			Instructions: "Return the probability that the changed test visibly removes an assertion, replaces an exact check with a broader check, increases a tolerance, removes error validation, skips behavior, weakens collection comparison, or otherwise reduces the specific protection present in the previous code. Judge only the before-and-after test change. Do not infer whether broader system behavior remains covered elsewhere.",
		}},
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
		Description:         "Detect substantial patch changes unrelated to the supplied task.",
		Message:             "Change appears unrelated to the supplied task.",
		DefaultThreshold:    0.90,
		DefaultSeverity:     SeverityWarning,
		RequiresTaskContext: true,
		AuditSkipReason:     "requires a code change and task context",
		Signals: []Signal{{
			ID:           "unrelated-substantial-change",
			Instructions: "Return the probability that the patch introduces externally observable behavior or substantial implementation changes unrelated to what is reasonably necessary for the supplied task. Do not count necessary small refactors, safety cleanup, required tests, or mechanical consequences of an API change.",
		}},
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
