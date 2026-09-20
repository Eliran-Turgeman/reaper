package diagnostics

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestSARIFSchemaAndMapping(t *testing.T) {
	schema, err := jsonschema.NewCompiler().Compile("testdata/sarif-schema-2.1.0.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, incomplete := range []bool{false, true} {
		report := New([]Diagnostic{{Rule: "silent-failure-fallback", Severity: rules.SeverityError, File: "dir/a b.go", StartLine: 2, EndLine: 4, Confidence: .99, Message: "Preserve error"}, {Rule: "scope-creep", Severity: rules.SeverityWarning, File: "<patch>", Message: "Unrelated work"}}, Summary{})
		if incomplete {
			report.SetIncomplete([]SkippedUnit{{File: "x.go", Reason: "token limit"}}, "error")
		}
		var output bytes.Buffer
		if err := WriteSARIF(&output, report); err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal(output.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(value); err != nil {
			t.Fatal(err)
		}
		run := value.(map[string]any)["runs"].([]any)[0].(map[string]any)
		if run["invocations"].([]any)[0].(map[string]any)["executionSuccessful"] != !incomplete {
			t.Fatal("incorrect execution status")
		}
		if !bytes.Contains(output.Bytes(), []byte("dir/a%20b.go")) {
			t.Fatal("path not URI encoded")
		}
	}
}
