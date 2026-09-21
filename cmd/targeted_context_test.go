package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestTargetedCheckUsesBoundedSnapshotHelpersOnlyForSelectedRules(t *testing.T) {
	var mu sync.Mutex
	var captured []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		data, _ := json.Marshal(request)
		mu.Lock()
		captured = append(captured, string(data))
		mu.Unlock()
		answers := map[string]any{}
		for id := range request["questions"].(map[string]any) {
			if strings.HasPrefix(id, "removed-validation:") || strings.HasPrefix(id, "removed-authorization-check:") {
				if _, ok := request["state"].(map[string]any); !ok {
					t.Error("targeted state was not a native object")
				}
			}
			if strings.Contains(string(data), "func allow(") && !strings.HasPrefix(id, "removed-validation:") && !strings.HasPrefix(id, "removed-authorization-check:") {
				t.Error("helper evidence leaked to unrelated rule", id)
			}
			answers[id] = map[string]any{"type": "noul", "noul": .2}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"answers": answers})
	}))
	defer server.Close()
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root, "-c", "core.autocrlf=false"}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
	write := func(name, code string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(code), 0600); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "--quiet")
	write("service.go", "package service\nfunc Save(n int) error { if err := validate(n); err != nil { return err }; return store(n) }\n")
	write("helpers.go", "package service\nfunc validate(n int) error { if n < 0 { return invalid }; return nil }\nfunc allow(n int) error { return nil }\nfunc unrelated() { panic(\"NOISE_MARKER\") }\n")
	git("add", "service.go", "helpers.go")
	git("-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "before")
	write("service.go", "package service\nfunc Save(n int) error { if err := allow(n); err != nil { return err }; return store(n) }\n")
	configuration := fmt.Sprintf("provider: typesafe\nbase_url: %s\ncache:\n  enabled: false\n", server.URL)
	write(".reaper.yaml", configuration)
	t.Chdir(root)
	run := func(extra ...string) []string {
		t.Helper()
		var output bytes.Buffer
		command := NewWith(App{Out: &output, ErrOut: &output, Getenv: func(string) string { return "test" }})
		command.SetArgs(append([]string{"check", "--experimental-context", "targeted-go", "--task", "Keep rejecting negative values", "--format", "json"}, extra...))
		if err := command.Execute(); err != nil {
			t.Fatalf("%v %s", err, output.String())
		}
		mu.Lock()
		defer mu.Unlock()
		result := append([]string(nil), captured...)
		captured = nil
		sort.Strings(result)
		return result
	}
	working := run()
	joined := strings.Join(working, "\n")
	if !strings.Contains(joined, "func validate(") || !strings.Contains(joined, "func allow(") || strings.Contains(joined, "NOISE_MARKER") {
		t.Fatal("wrong targeted helper evidence", joined)
	}
	git("add", "service.go")
	write("helpers.go", "package service\nfunc allow(n int) error { panic(\"UNSTAGED_MARKER\") }\n")
	if staged := run("--staged"); !reflect.DeepEqual(staged, working) {
		t.Fatal("unstaged helper changed staged evidence", staged)
	}
	if err := os.Remove(filepath.Join(root, "helpers.go")); err != nil {
		t.Fatal(err)
	}
	if staged := run("--staged"); !reflect.DeepEqual(staged, working) {
		t.Fatal("unstaged helper deletion changed staged evidence", staged)
	}
	write(".reaper.yaml", configuration+"exclude:\n  - helpers.go\n")
	excluded := strings.Join(run("--staged"), "\n")
	if strings.Contains(excluded, "func allow(") || !strings.Contains(excluded, "unresolved direct calls") {
		t.Fatal("excluded helper read or missing coverage limitation", excluded)
	}
}
