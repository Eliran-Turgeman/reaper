package eval

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestValidationCorpusRemainsFrozenAndSeparate(t *testing.T) {
	const dir = "../../benchmarks/validation"
	cases, err := LoadPatches(dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dir + "/protocol.json")
	if err != nil {
		t.Fatal(err)
	}
	var protocol struct {
		Hash       string                        `json:"cases_sha256"`
		Candidates map[string]map[string]float64 `json:"candidates"`
	}
	if err := json.Unmarshal(data, &protocol); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(dir + "/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	// The protocol fingerprints canonical LF text across Git checkout settings.
	canonical := strings.ReplaceAll(string(data), "\r\n", "\n")
	if fmt.Sprintf("%x", sha256.Sum256([]byte(canonical))) != protocol.Hash {
		t.Fatal("validation cases changed after the protocol was frozen")
	}
	development, err := LoadPatches("../../benchmarks/patches")
	if err != nil {
		t.Fatal(err)
	}
	seenIDs, seenPatches := map[string]bool{}, map[string]bool{}
	for _, c := range development {
		seenIDs[c.ID] = true
		seenPatches[c.Before+"\x00"+c.After] = true
	}
	counts := map[string][2]int{}
	for _, c := range cases {
		if seenIDs[c.ID] || seenPatches[c.Before+"\x00"+c.After] {
			t.Fatalf("validation case %s duplicates the calibration corpus", c.ID)
		}
		if len(protocol.Candidates[c.Rule]) != 2 {
			t.Fatalf("missing preselected candidates for %s", c.Rule)
		}
		n := counts[c.Rule]
		if c.Expected == "positive" {
			n[0]++
		} else {
			n[1]++
		}
		counts[c.Rule] = n
	}
	if len(counts) != 4 {
		t.Fatal("validation must cover the four patch-benchmark rules")
	}
	for rule, count := range counts {
		if count != [2]int{3, 5} {
			t.Fatalf("%s needs the frozen three regressions and five hard negatives: %v", rule, count)
		}
	}
}

func TestValidationResultsMatchFrozenCases(t *testing.T) {
	const dir = "../../benchmarks/validation"
	cases, err := LoadPatches(dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dir + "/results.json")
	if err != nil {
		t.Fatal(err)
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(dir + "/protocol.json")
	if err != nil {
		t.Fatal(err)
	}
	var protocol struct {
		Provider   string                        `json:"provider"`
		Model      string                        `json:"model"`
		Candidates map[string]map[string]float64 `json:"candidates"`
	}
	if err := json.Unmarshal(data, &protocol); err != nil {
		t.Fatal(err)
	}
	if report.Provider != protocol.Provider || report.Model != protocol.Model || len(report.Cases) != len(cases) || len(report.Rules) != len(protocol.Candidates) {
		t.Fatal("validation report does not match the frozen experiment")
	}
	byID := map[string]PatchCase{}
	for _, c := range cases {
		byID[c.ID] = c
	}
	for _, scored := range report.Cases {
		c, ok := byID[scored.ID]
		if !ok || c.Rule != scored.Rule || c.Expected != scored.Expected || c.Split != scored.Split || math.IsNaN(scored.Score) || scored.Score < 0 || scored.Score > 1 {
			t.Fatalf("invalid validation score: %+v", scored)
		}
		delete(byID, scored.ID)
	}
	seen := map[string]bool{}
	for _, recorded := range report.Rules {
		if seen[recorded.Rule] || protocol.Candidates[recorded.Rule] == nil {
			t.Fatalf("unexpected or duplicate rule: %s", recorded.Rule)
		}
		seen[recorded.Rule] = true
		if actual := Metrics(recorded.Rule, report.Cases, recorded.Threshold); !reflect.DeepEqual(actual, recorded) {
			t.Fatalf("recorded metrics for %s disagree with scores", recorded.Rule)
		}
		for _, family := range []string{"patch", "seed"} {
			threshold := protocol.Candidates[recorded.Rule][family]
			m := Metrics(recorded.Rule, report.Cases, threshold)
			t.Logf("%s %s threshold=%.2f TP=%d FP=%d TN=%d FN=%d precision=%.3f recall=%.3f", recorded.Rule, family, threshold, m.TruePositive, m.FalsePositive, m.TrueNegative, m.FalseNegative, m.Precision, m.Recall)
		}
	}
}
