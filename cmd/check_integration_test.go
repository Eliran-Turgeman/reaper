package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIReportsIncompleteAndWarningPolicy(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git: %s %v", out, err)
		}
	}
	git("init")
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\nfunc Value() int { return 1 }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", "x.go")
	git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "fixture")
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\nfunc Value() int { return 2 }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	for _, tc := range []struct {
		policy                  string
		incomplete, failWarning bool
		code                    int
	}{{"error", true, false, 2}, {"warning", true, false, 0}, {"ignore", true, false, 0}, {"error", false, true, 1}, {"error", false, false, 0}} {
		t.Run(fmt.Sprintf("%s-%t-%t", tc.policy, tc.incomplete, tc.failWarning), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.incomplete {
					w.WriteHeader(400)
					fmt.Fprint(w, `{"error":"max_tokens_exceeded"}`)
					return
				}
				var request struct {
					Questions map[string]any `json:"questions"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				answers := map[string]any{}
				for id := range request.Questions {
					answers[id] = map[string]any{"type": "noul", "noul": .99}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"answers": answers})
			}))
			defer server.Close()
			cfg := fmt.Sprintf("base_url: %s\npacks: []\nincomplete_analysis: %s\ncache:\n  enabled: false\nrules:\n  silent-failure-fallback:\n    enabled: true\n    severity: warning\n", server.URL, tc.policy)
			if err := os.WriteFile(filepath.Join(dir, "explicit.yaml"), []byte(cfg), 0600); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			app := NewWith(App{Out: &out, ErrOut: &stderr, Getenv: func(string) string { return "test" }})
			args := []string{"check", "--config", "explicit.yaml", "--format", "json"}
			if tc.failWarning {
				args = append(args, "--fail-on-warning")
			}
			app.SetArgs(args)
			err := app.Execute()
			if ExitCode(err) != tc.code {
				t.Fatalf("exit %d want %d: %v %s", ExitCode(err), tc.code, err, stderr.String())
			}
			var report diagnostics.Report
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatal(err, out.String())
			}
			if report.Complete == tc.incomplete {
				t.Fatal("wrong completeness")
			}
			if tc.incomplete && (report.Passed || len(report.Skipped) != 1) {
				t.Fatal(report)
			}
			if tc.failWarning && report.Passed {
				t.Fatal("warning gate claimed pass")
			}
		})
	}
}
