package diff

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotSourceFailuresAndDeletion(t *testing.T) {
	failure := errors.New("snapshot unavailable")
	patch := "diff --git a/service.go b/service.go\n--- a/service.go\n+++ b/service.go\n@@ -1 +1 @@\n-old\n+new\n"
	if _, err := UnitsWithSource(patch, 6, func(string) ([]byte, error) { return nil, failure }); !errors.Is(err, failure) {
		t.Fatalf("source failure must stop extraction: %v", err)
	}
	deleted := "diff --git a/service.go b/service.go\n--- a/service.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-old\n"
	units, err := UnitsWithSource(deleted, 6, func(string) ([]byte, error) {
		t.Fatal("deleted source must not be read")
		return nil, failure
	})
	if err != nil || len(units) != 1 || units[0].SurroundingCode != "" || units[0].OldContent != "old" {
		t.Fatalf("deleted file extraction: %+v, %v", units, err)
	}
}

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
	if !unit.ContainsComments {
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

func TestUnitsSkipsUnsupportedDataFiles(t *testing.T) {
	input := `diff --git a/EmailCollector.Api/Services/EmailValidations/emails.txt b/EmailCollector.Api/Services/EmailValidations/emails.txt
new file mode 100644
--- /dev/null
+++ b/EmailCollector.Api/Services/EmailValidations/emails.txt
@@ -0,0 +1,2 @@
+one@example.com
+two@example.com
diff --git a/EmailCollector.Api/Services/EmailValidator.cs b/EmailCollector.Api/Services/EmailValidator.cs
new file mode 100644
--- /dev/null
+++ b/EmailCollector.Api/Services/EmailValidator.cs
@@ -0,0 +1 @@
+class EmailValidator {}
diff --git a/wwwroot/js/site.min.js b/wwwroot/js/site.min.js
new file mode 100644
--- /dev/null
+++ b/wwwroot/js/site.min.js
@@ -0,0 +1 @@
+const minified=true;
`
	units, err := Units(t.TempDir(), input, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 1 || units[0].FilePath != "EmailCollector.Api/Services/EmailValidator.cs" {
		t.Fatalf("unexpected units: %#v", units)
	}
}
