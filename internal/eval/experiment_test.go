package eval

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

func TestExperimentsChangeOnlyChosenInputAndRecordActualRequest(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "test", Rule: "removed-validation", Task: "Keep validation", Expected: "positive", Rationale: "GOLD LABEL MUST NEVER LEAK"}, BeforeFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"}, AfterFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { store(n) }\n"}}
	var baseState, baseQuestions string
	for _, variant := range []string{"current", "questions", "context", "both"} {
		experiment, err := LoadExperiment("../../benchmarks/experiments/auth-validation-v2/" + variant + ".json")
		if err != nil {
			t.Fatal(err)
		}
		client := &benchmarkEvaluator{score: .7}
		result, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", experiment)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Requests) != 1 || len(client.requests) != 1 {
			t.Fatalf("missing request: %+v", result)
		}
		request := client.requests[0]
		if result.Requests[0].SHA256 != fingerprint(request) {
			t.Fatal("recorded pre-transform request")
		}
		if strings.Contains(request.State, c.Rationale) {
			t.Fatal("label evidence leaked")
		}
		if variant == "current" {
			baseState, baseQuestions = fingerprint(request.State), fingerprint(request.Questions)
		}
		wantContextChange := variant == "context" || variant == "both"
		wantQuestionChange := variant == "questions" || variant == "both"
		if (fingerprint(request.State) != baseState) != wantContextChange || (fingerprint(request.Questions) != baseQuestions) != wantQuestionChange {
			t.Fatalf("%s changed the wrong input", variant)
		}
		for _, signal := range result.Observations[0].Signals {
			for _, q := range request.Questions {
				if q.ID == c.Rule+":"+signal.ID && signal.Evidence != q.Instructions {
					t.Fatal("signal describes the wrong question")
				}
			}
		}
	}
}

func TestSignalExperimentRecordsRawPermissionAndPreservesProductionRules(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "permission", Rule: "removed-validation", Task: "Accept all integers", Expected: "negative"}, BeforeFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"}, AfterFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { store(n) }\n"}}
	e := &Experiment{Version: 1, Name: "policy", Context: "current", Signals: map[string][]rules.Signal{c.Rule: {{ID: "loss", Instructions: "Is input enforcement lost?"}, {ID: "permission", Instructions: "Does the task explicitly require accepting the previously rejected inputs?", Negate: true}}}}
	if err := e.validate(); err != nil {
		t.Fatal(err)
	}
	measured, err := runGitCaseExperiment(context.Background(), &benchmarkEvaluator{score: .9}, c, config.Defaults(), "isolated", e)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(measured.Score-.1) > 1e-9 || measured.Requests[0].Scores[c.Rule+":permission"] != .9 || measured.Observations[0].Composition != "minimum-of-oriented-signals-v1" {
		t.Fatal(measured)
	}
	if !measured.Observations[0].Signals[1].Negated {
		t.Fatal("raw permission score polarity not recorded")
	}
	unchanged, err := runGitCaseExperiment(context.Background(), &benchmarkEvaluator{score: .9}, c, config.Defaults(), "isolated", nil)
	if err != nil || unchanged.Score != .9 || unchanged.Observations[0].RuleVersion != 1 {
		t.Fatal("experiment mutated production rules", unchanged, err)
	}
	for _, signals := range [][]rules.Signal{
		{}, {{ID: "x", Instructions: "Q?", Negate: true}}, {{ID: "x", Instructions: "Q?"}, {ID: "x", Instructions: "Duplicate?"}}, {{ID: "x:y", Instructions: "Q?"}},
	} {
		e.Signals[c.Rule] = signals
		if e.validate() == nil {
			t.Fatal("invalid signal experiment accepted", signals)
		}
	}
}

func TestPermissionExperimentKeepsFactsIndependentAndClassifiesUncertainty(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "permission-v2", Rule: "removed-validation", Task: "Simplify input handling", Expected: "positive"}, BeforeFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"}, AfterFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { store(n) }\n"}}
	e := &Experiment{
		Version: 1, Name: "permission-v2", Context: "current",
		Permission: &rules.PermissionPolicy{AllowedAtLeast: .8, DisallowedAtMost: .2},
		Signals: map[string][]rules.Signal{c.Rule: {
			{ID: "previous-validation", Instructions: "Was validation present?"},
			{ID: "unguarded-operation", Instructions: "Is the operation now unguarded?"},
			{ID: "task-permits-change", Instructions: "Does the task explicitly permit this?", Role: rules.SignalRolePermission},
		}},
	}
	client := &benchmarkEvaluator{scores: map[string]float64{
		c.Rule + ":previous-validation": .98,
		c.Rule + ":unguarded-operation": .97,
		c.Rule + ":task-permits-change": .5,
	}}
	measured, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", e)
	if err != nil {
		t.Fatal(err)
	}
	observation := measured.Observations[0]
	if measured.Score != .97 || observation.Permission == nil || observation.Permission.Outcome != "uncertain" || observation.Permission.Score != .5 {
		t.Fatalf("permission classification changed factual score: %+v", measured)
	}
	if observation.Composition != "minimum-of-factual-signals-with-separate-permission-v1" {
		t.Fatal(observation.Composition)
	}
}

func TestPermissionExperimentRequiresEveryPermissionCondition(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "permission-v3", Rule: "removed-validation", Task: "Accept zero but reject negatives", Expected: "positive"}, BeforeFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { if n <= 0 { panic(n) }; store(n) }\n"}, AfterFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { store(n) }\n"}}
	e := &Experiment{
		Version: 1, Name: "permission-v3", Context: "current",
		Permission: &rules.PermissionPolicy{AllowedAtLeast: .8, DisallowedAtMost: .2},
		Signals: map[string][]rules.Signal{c.Rule: {
			{ID: "loss", Instructions: "Was validation lost?"},
			{ID: "specific-scope", Instructions: "Does the task authorize the named scope?", Role: rules.SignalRolePermission},
			{ID: "complete-scope", Instructions: "Does it authorize every newly accepted input?", Role: rules.SignalRolePermission},
		}},
	}
	client := &benchmarkEvaluator{scores: map[string]float64{
		c.Rule + ":loss":           .97,
		c.Rule + ":specific-scope": .94,
		c.Rule + ":complete-scope": .12,
	}}
	measured, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", e)
	if err != nil {
		t.Fatal(err)
	}
	permission := measured.Observations[0].Permission
	if permission == nil || permission.Score != .12 || permission.Outcome != "disallowed" || len(permission.Signals) != 2 {
		t.Fatalf("permission conditions were not composed conservatively: %+v", permission)
	}
	if measured.Observations[0].Composition != "minimum-of-factual-signals-with-minimum-permission-v2" {
		t.Fatal(measured.Observations[0].Composition)
	}
}

func TestPermissionExperimentCanUseSeparateRequest(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "permission-separated", Rule: "removed-validation", Task: "Accept negatives", Expected: "negative"}, BeforeFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"}, AfterFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { store(n) }\n"}}
	e := &Experiment{
		Version: 1, Name: "permission-separated", Context: "current", PermissionEval: "separate-request-v1",
		Permission: &rules.PermissionPolicy{AllowedAtLeast: .8, DisallowedAtMost: .2},
		Signals: map[string][]rules.Signal{c.Rule: {
			{ID: "loss", Instructions: "Was validation lost?"},
			{ID: "scope", Instructions: "Was this scope authorized?", Role: rules.SignalRolePermission},
			{ID: "complete", Instructions: "Was the whole change authorized?", Role: rules.SignalRolePermission},
		}},
	}
	client := &benchmarkEvaluator{score: .9}
	measured, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", e)
	if err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 2 || len(measured.Requests) != 2 {
		t.Fatalf("permission questions were not isolated: requests=%d records=%d", len(client.requests), len(measured.Requests))
	}
	if len(client.requests[0].Questions) != 1 || len(client.requests[1].Questions) != 2 {
		t.Fatalf("unexpected question groups: %+v", client.requests)
	}
}

func TestPermissionExperimentSuppressesOnlyAllowedOutcome(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "permission-action", Rule: "removed-validation", Task: "Accept negative values", Expected: "negative"}, BeforeFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"}, AfterFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { store(n) }\n"}}
	e := &Experiment{
		Version: 1, Name: "permission-action", Context: "current", PermissionAction: "suppress-allowed",
		Permission: &rules.PermissionPolicy{AllowedAtLeast: .8, DisallowedAtMost: .2},
		Signals: map[string][]rules.Signal{c.Rule: {
			{ID: "previous-validation", Instructions: "Was validation present?"},
			{ID: "unguarded-operation", Instructions: "Is the operation now unguarded?"},
			{ID: "task-permits-change", Instructions: "Does the task explicitly permit this?", Role: rules.SignalRolePermission},
		}},
	}
	client := &benchmarkEvaluator{scores: map[string]float64{
		c.Rule + ":previous-validation": .98,
		c.Rule + ":unguarded-operation": .97,
		c.Rule + ":task-permits-change": .9,
	}}
	measured, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", e)
	if err != nil {
		t.Fatal(err)
	}
	if measured.Score != .97 || measured.Decision != "permission-allowed" || measured.PermissionOutcome != "allowed" {
		t.Fatalf("allowed permission did not suppress independently: %+v", measured)
	}
	metrics := Metrics(c.Rule, []ScoredCase{measured}, .95)
	if metrics.TrueNegative != 1 || metrics.FalsePositive != 0 {
		t.Fatalf("suppressed permission counted as finding: %+v", metrics)
	}
}

func TestStructuredExperimentPreservesFactsAndRecordsCriteria(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "structured", Rule: "removed-validation", Task: "Keep validation", Expected: "positive", Rationale: "PRIVATE GOLD LABEL"}, BeforeFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"}, AfterFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { store(n) }\n"}}
	var textState string
	for _, format := range []string{"json-text", "json-object"} {
		client := &benchmarkEvaluator{score: .7}
		e := &Experiment{Version: 1, Name: "format", Context: "snapshots", StateFormat: format, Criteria: map[string]decision.NoulCriteria{"removed-validation:removes-validation": {True: "A previous check is bypassed.", False: "Previous checks still apply."}}}
		result, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", e)
		if err != nil {
			t.Fatal(err)
		}
		request := client.requests[0]
		if format == "json-text" {
			textState = request.State
		} else if string(request.StructuredState) != textState || result.Requests[0].StateSHA256 != fingerprint(request.StructuredState) {
			t.Fatal("representation changed the supplied facts")
		}
		if strings.Contains(textState, c.Rationale) {
			t.Fatal("label leaked into evidence")
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(textState), &fields); err != nil || fields["snapshot_evidence"] == nil {
			t.Fatal("missing structured snapshot evidence", err)
		}
		if result.Requests[0].Questions[0].Criteria == nil || result.Requests[0].SHA256 != fingerprint(request) {
			t.Fatal("criteria missing from actual request provenance")
		}
	}
}

func TestExampleExperimentsKeepInputsIsolated(t *testing.T) {
	thresholds := map[string]float64{"removed-validation": .95}
	var baselineState, baselineQuestions string
	for _, variant := range []string{"baseline", "wording", "json-text", "json-object", "criteria"} {
		e := &Experiment{Version: 1, Name: variant, Context: "current"}
		if variant != "baseline" {
			e.Questions = map[string]string{"removed-validation:removes-validation": "Does the change bypass an old input check?"}
		}
		if variant == "json-text" || variant == "json-object" {
			e.StateFormat = variant
		}
		if variant == "criteria" {
			e.Criteria = map[string]decision.NoulCriteria{"removed-validation:removes-validation": {True: "Old input enforcement is lost.", False: "Old input enforcement remains."}}
		}
		client := &benchmarkEvaluator{score: .7}
		report, err := RunExamplesExperiment(context.Background(), client, "../../evals", "typesafe", "m", "removed-validation", nil, thresholds, e)
		if err != nil {
			t.Fatal(err)
		}
		request := client.requests[0]
		if variant == "baseline" {
			baselineState, baselineQuestions = fingerprint(request.State), fingerprint(request.Questions)
		}
		if (variant == "wording" || variant == "criteria") && fingerprint(request.State) != baselineState {
			t.Fatal("question variant changed state")
		}
		if variant != "baseline" && fingerprint(request.Questions) == baselineQuestions {
			t.Fatal("question override missing")
		}
		if report.Cases[0].Requests[0].SHA256 != fingerprint(request) || report.Provenance.ExperimentSHA256 != fingerprint(e) {
			t.Fatal("missing actual experiment provenance")
		}
		if variant == "json-object" && len(request.StructuredState) == 0 {
			t.Fatal("missing native state")
		}
	}
	if _, err := RunExamplesExperiment(context.Background(), &benchmarkEvaluator{}, "../../evals", "typesafe", "m", "removed-validation", nil, thresholds, &Experiment{Version: 1, Name: "bad", Context: "targeted"}); err == nil {
		t.Fatal("accepted nonexistent repository evidence")
	}
}

func TestExperimentRejectsUnknownQuestionsAndOversizedContext(t *testing.T) {
	for _, data := range []string{`{"version":1,"name":"x","context":"current","questions":{"unknown":"text"}}`, `{"version":1,"name":"x","context":"current","threshold":0}`, `{"version":1,"name":"x","context":"current"} {}`, `{"version":1,"name":"x","context":"current","state_format":"yaml"}`, `{"version":1,"name":"x","context":"current","criteria":{"unknown":{"true":"yes","false":"no"}}}`, `{"version":1,"name":"x","context":"current","criteria":{"removed-validation:removes-validation":{"true":"yes"}}}`} {
		file := filepath.Join(t.TempDir(), "experiment.json")
		if err := os.WriteFile(file, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadExperiment(file); err == nil {
			t.Fatal("invalid experiment accepted")
		}
	}

	if _, err := experimentEvidence(GitCase{BeforeFiles: map[string]string{"x.go": strings.Repeat("x", 65537)}}, &Experiment{Context: "snapshots"}); err == nil {
		t.Fatal("oversized context silently accepted")
	}
}

func TestContextFixturesSupplyPreviouslyInvisibleHelperBodies(t *testing.T) {
	cases, err := LoadGitCases("../../benchmarks/context-dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 4 {
		t.Fatal("missing paired context fixtures")
	}
	for _, c := range cases {
		for _, mode := range []string{"current", "snapshots"} {
			client := &benchmarkEvaluator{score: .5}
			_, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", &Experiment{Version: 1, Name: "context test", Context: mode})
			if err != nil {
				t.Fatal(err)
			}
			if len(client.requests) != 1 {
				t.Fatal("fixture must produce one changed-unit request")
			}
			hasHelper := strings.Contains(client.requests[0].State, "func ensurePermission(") || strings.Contains(client.requests[0].State, "func validateAmount(")
			if hasHelper != (mode == "snapshots") {
				t.Fatalf("%s/%s supplied incorrect helper evidence", c.ID, mode)
			}
		}
	}
}

func TestFollowUpExperimentSpecificationsLoad(t *testing.T) {
	for _, file := range []string{
		"../../benchmarks/experiments/permission-decision-v2/policy-targeted.json",
		"../../benchmarks/experiments/permission-decision-v2/policy-contracts.json",
		"../../benchmarks/experiments/permission-decision-v2/policy-assertion-dev.json",
		"../../benchmarks/experiments/factual-predicates-v2/discarded-errors.json",
		"../../benchmarks/experiments/factual-predicates-v2/cancellation-ownership.json",
		"../../benchmarks/experiments/factual-predicates-v2/assertion-specificity.json",
	} {
		if _, err := LoadExperiment(file); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
}
