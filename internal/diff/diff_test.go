package diff

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnitsExtractsBeforeAfterAndContext(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "client.go")
	source := "package client\n\nfunc call() error {\n\t// Return the error.\n\treturn err\n}\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	input := `diff --git a/client.go b/client.go
index 1111111..2222222 100644
--- a/client.go
+++ b/client.go
@@ -1,5 +1,6 @@
 package client
 
 func call() error {
+	// Return the error.
 	return err
 }
`
	units, err := Units(root, input, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 1 {
		t.Fatalf("got %d units, want 1", len(units))
	}
	unit := units[0]
	if unit.FilePath != "client.go" || unit.StartLine != 1 || unit.EndLine != 6 {
		t.Fatalf("unexpected location: %#v", unit)
	}
	if !unit.ContainsComments || unit.Additions != 1 {
		t.Fatalf("unexpected applicability metadata: %#v", unit)
	}
	if unit.OldContent == unit.NewContent || unit.SurroundingCode == "" {
		t.Fatalf("expected distinct before/after and surrounding context")
	}
}

func TestParseRejectsMalformedHunk(t *testing.T) {
	_, err := Parse("diff --git a/a.go b/a.go\n@@ malformed @@\n")
	if err == nil {
		t.Fatal("expected malformed hunk error")
	}
}
