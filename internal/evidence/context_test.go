package evidence

import (
	"encoding/json"
	"github.com/Eliran-Turgeman/reaper/internal/diff"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
	"strings"
	"testing"
)

func TestTargetedContextHandlesShadowingLimitsAndParseFailure(t *testing.T) {
	h := diff.Hunk{OldStart: 3, NewStart: 3, Lines: []string{"- old()", "+ ensurePermission()"}}
	for _, tc := range []struct {
		name, source, helper    string
		wantHelper, wantLimited bool
	}{
		{"direct", "package service\nfunc Read() {\n ensurePermission()\n}\n", "package service\nfunc ensurePermission() { panic(\"HELPER_BODY\") }\n", true, false},
		{"shadowed", "package service\nfunc Read(ensurePermission func()) {\n ensurePermission()\n}\n", "package service\nfunc ensurePermission() { panic(\"HELPER_BODY\") }\n", false, false},
		{"method", "package service\nfunc Read() {\n actor.ensurePermission()\n}\n", "package service\nfunc ensurePermission() { panic(\"HELPER_BODY\") }\n", false, false},
		{"oversized", "package service\nfunc Read() {\n ensurePermission()\n}\n", "package service\nfunc ensurePermission() { /*" + strings.Repeat("x", targetedEvidenceLimit) + "*/ panic(\"HELPER_BODY\") }\n", false, true},
		{"broken", "package service\nfunc Read() {\n ensurePermission(\n", "package service\nfunc ensurePermission() {}\n", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Context{}
			collectSide(&e, map[string]string{"service.go": tc.source, "policy.go": tc.helper}, "service.go", h, false, true)
			data, _ := json.Marshal(e)
			if strings.Contains(string(data), "HELPER_BODY") != tc.wantHelper {
				t.Fatalf("wrong resolution: %s", data)
			}
			if len(data) > targetedEvidenceLimit || strings.Contains(string(data), "size limit") != tc.wantLimited {
				t.Fatalf("bad size handling: %d %s", len(data), data)
			}
			if tc.name == "broken" && !strings.Contains(string(data), "did not parse") {
				t.Fatal("missing parse failure notice")
			}
		})
	}
}

func TestMatchedContextLocatesInsertionOnBothSides(t *testing.T) {
	h := diff.Hunk{OldStart: 1, NewStart: 1, Lines: []string{" package service", " func Read() {", "+ bypass()", "  read()", " }"}}
	for _, before := range []bool{true, false} {
		first, last := changedLines(h, before)
		if first != 3 || last != 3 {
			t.Fatalf("wrong changed location: %d..%d", first, last)
		}
	}
}

func TestAssessDistinguishesCompleteAndInsufficientEvidence(t *testing.T) {
	complete, err := json.Marshal(Context{Snippets: []codeEvidence{
		{Side: "before", Kind: "changed-function"},
		{Side: "after", Kind: "changed-function"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := Assess(complete)
	if err != nil || !coverage.Complete {
		t.Fatalf("complete evidence rejected: %+v %v", coverage, err)
	}
	incomplete, err := json.Marshal(Context{Limitations: []string{"before: changed file did not parse"}})
	if err != nil {
		t.Fatal(err)
	}
	coverage, err = Assess(incomplete)
	if err != nil || coverage.Complete || len(coverage.Reasons) < 2 {
		t.Fatalf("insufficient evidence accepted: %+v %v", coverage, err)
	}
	unresolved, err := json.Marshal(Context{Snippets: []codeEvidence{
		{Side: "before", Kind: "changed-function"},
		{Side: "after", Kind: "changed-function"},
	}, Limitations: []string{"after: unresolved direct calls (possibly builtins/conversions): [check]"}})
	if err != nil {
		t.Fatal(err)
	}
	coverage, err = Assess(unresolved)
	if err != nil || coverage.Complete {
		t.Fatalf("ambiguous helper evidence accepted: %+v %v", coverage, err)
	}
	unchanged, err := json.Marshal(Context{Snippets: []codeEvidence{
		{Side: "before", Kind: "changed-function"},
		{Side: "after", Kind: "changed-function"},
	}, Limitations: []string{"unchanged unresolved direct calls (possibly builtins/conversions): [traceStep(0)]"}})
	if err != nil {
		t.Fatal(err)
	}
	coverage, err = Assess(unchanged)
	if err != nil || !coverage.Complete {
		t.Fatalf("unchanged unresolved call forced abstention: %+v %v", coverage, err)
	}
}

func TestTargetedContextDistinguishesChangedAndUnchangedUnresolvedCalls(t *testing.T) {
	unresolved := func(after string) Context {
		units := []semantic.Unit{{FilePath: "service.go", StartLine: 2, Language: "go"}}
		beforeFiles := map[string]string{"service.go": "package service\nfunc Read() {\n oldGuard()\n traceStep(0)\n}\n"}
		afterFiles := map[string]string{"service.go": after}
		if err := Add(beforeFiles, afterFiles, "diff --git a/service.go b/service.go\n--- a/service.go\n+++ b/service.go\n@@ -2,4 +2,4 @@\n func Read() {\n- oldGuard()\n+ newGuard()\n  traceStep(0)\n }\n", units, true); err != nil {
			t.Fatal(err)
		}
		var report Context
		if err := json.Unmarshal(units[0].RelatedEvidence, &report); err != nil {
			t.Fatal(err)
		}
		return report
	}
	report := unresolved("package service\nfunc Read() {\n newGuard()\n traceStep(0)\n}\n")
	data, _ := json.Marshal(report)
	coverage, err := Assess(data)
	if err != nil || coverage.Complete {
		t.Fatalf("changed unresolved calls were accepted: %+v %v", coverage, err)
	}
	if !strings.Contains(strings.Join(report.Limitations, "\n"), "unchanged unresolved direct calls") {
		t.Fatalf("unchanged call was not distinguished: %+v", report.Limitations)
	}
}

func TestTargetedContextAssociatesAdjacentFunctionComment(t *testing.T) {
	h := diff.Hunk{OldStart: 2, NewStart: 2, Lines: []string{
		"-// Existing policy.",
		"+// Shared policy.",
		" func ensurePermission() error { return nil }",
	}}
	for _, before := range []bool{true, false} {
		e := Context{}
		source := "package service\n// Shared policy.\nfunc ensurePermission() error { return nil }\n"
		collectSide(&e, map[string]string{"policy.go": source}, "policy.go", h, before, true)
		if len(e.Snippets) != 1 || e.Snippets[0].Kind != "changed-function" {
			t.Fatalf("adjacent function was not selected: %+v", e)
		}
	}
}
