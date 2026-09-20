package eval

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
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
