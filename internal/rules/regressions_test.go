package rules

import (
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
	"testing"
)

func TestNewRegressionsRequireProductionChange(t *testing.T) {
	for _, id := range []string{"removed-authorization-check", "removed-validation", "swallowed-cancellation"} {
		rule, ok := Get(id)
		if !ok || rule.Pack != "regressions" || rule.AuditSkipReason == "" {
			t.Fatal(id)
		}
		for _, unit := range []semantic.Unit{{}, {IsTest: true, ExistingModified: true}} {
			if applies, _ := rule.Applicable(unit, ""); applies {
				t.Fatalf("%s applies without production change", id)
			}
		}
		if applies, _ := rule.Applicable(semantic.Unit{ExistingModified: true}, ""); !applies {
			t.Fatal(id)
		}
	}
}
