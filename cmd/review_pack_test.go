package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewPackCommandRequiresNoProviderAndBlindsLabels(t *testing.T) {
	dir := t.TempDir()
	data := `[{"id":"case","rule":"removed-validation","expected":"positive","task":"Keep validation","before_files":{"x.go":"package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"},"after_files":{"x.go":"package x\nfunc Save(n int) { store(n) }\n"},"rationale":"SECRET GOLD","provenance":"test","split":"dev"}]`
	if err := os.WriteFile(filepath.Join(dir, "cases.json"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	command := NewWith(App{Out: &output, ErrOut: &output, Getenv: func(string) string { return "" }})
	command.SetArgs([]string{"review-pack", "--benchmark-dir", dir})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "SECRET GOLD") || !strings.Contains(output.String(), `"source_corpus_sha256"`) {
		t.Fatal("review pack leaked labels or missed corpus binding")
	}
}
