package eval

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewPackIsDeterministicAndBlindsGoldLabels(t *testing.T) {
	dir := t.TempDir()
	data := `[
	  {"id":"positive-secret","rule":"removed-validation","expected":"positive","task":"Keep validation","before_files":{"x.go":"package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"},"after_files":{"x.go":"package x\nfunc Save(n int) { store(n) }\n"},"rationale":"GOLD LABEL SECRET","provenance":"test","split":"dev"},
	  {"id":"negative-secret","rule":"removed-validation","expected":"negative","task":"Accept negative values","before_files":{"x.go":"package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"},"after_files":{"x.go":"package x\nfunc Save(n int) { store(n) }\n"},"rationale":"OTHER GOLD SECRET","provenance":"test","split":"dev"}
	]`
	if err := os.WriteFile(filepath.Join(dir, "cases.json"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := BuildReviewPack(dir, ReviewPackOptions{Seed: "pilot", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildReviewPack(dir, ReviewPackOptions{Seed: "pilot", Limit: 1})
	if err != nil || fingerprint(first) != fingerprint(second) {
		t.Fatal("review selection is not deterministic", err)
	}
	var out bytes.Buffer
	if err := WriteReviewPack(&out, first); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, secret := range []string{"GOLD LABEL SECRET", "OTHER GOLD SECRET", `"expected"`, `"rationale"`, `"provenance"`} {
		if strings.Contains(text, secret) {
			t.Fatalf("review pack leaked %q", secret)
		}
	}
	var decoded ReviewPack
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil || decoded.CompleteCorpus || decoded.SelectedCases != 1 {
		t.Fatalf("invalid review pack: %+v %v", decoded, err)
	}
}
