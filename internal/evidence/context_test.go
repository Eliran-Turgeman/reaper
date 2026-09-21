package evidence

import (
	"encoding/json"
	"github.com/Eliran-Turgeman/reaper/internal/diff"
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
