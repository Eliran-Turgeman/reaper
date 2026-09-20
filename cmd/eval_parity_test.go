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
	"sync"
	"testing"
)

func TestGitBenchmarkAndCheckSendIdenticalRequests(t *testing.T) {
	var mu sync.Mutex
	var captured []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		canonical, _ := json.Marshal(request)
		mu.Lock()
		captured = append(captured, string(canonical))
		mu.Unlock()
		answers := map[string]any{}
		for id := range request["questions"].(map[string]any) {
			answers[id] = map[string]any{"type": "noul", "noul": .25}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"answers": answers})
	}))
	defer server.Close()
	root, corpus := t.TempDir(), t.TempDir()
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(name, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root, "-c", "core.autocrlf=false"}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	before := "package service\nfunc Read(fast bool) error {\n if !allowed() { return denied }\n return read()\n}\n"
	after := "package service\nfunc Read(fast bool) error {\n if fast { return read() }\n if !allowed() { return denied }\n return read()\n}\n"
	task := "Preserve authorization while optimizing reads"
	git("init", "--quiet")
	write(filepath.Join(root, "service.go"), before)
	git("add", "--", "service.go")
	write(filepath.Join(root, "service.go"), after)
	write(filepath.Join(root, ".reaper.yaml"), fmt.Sprintf("provider: typesafe\nbase_url: %s\ncache:\n  enabled: false\n", server.URL))
	cases := []map[string]any{{"id": "parity", "task": task, "rule": "removed-authorization-check", "expected": "positive", "rationale": "LABEL MUST NOT LEAK", "provenance": "unit test", "split": "dev", "before_files": map[string]string{"service.go": before}, "after_files": map[string]string{"service.go": after}}}
	data, _ := json.Marshal(cases)
	write(filepath.Join(corpus, "cases.json"), string(data))
	t.Chdir(root)
	run := func(args ...string) []string {
		t.Helper()
		var out bytes.Buffer
		command := NewWith(App{Out: &out, ErrOut: &out, Getenv: func(string) string { return "test" }})
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatalf("%v: %v %s", args, err, out.String())
		}
		mu.Lock()
		defer mu.Unlock()
		result := append([]string(nil), captured...)
		captured = nil
		sort.Strings(result)
		return result
	}
	check := run("check", "--task", task, "--no-cache", "--format", "json")
	benchmark := run("eval", "--benchmark-dir", corpus, "--benchmark-mode", "git", "--format", "json")
	if len(check) == 0 || !reflect.DeepEqual(check, benchmark) {
		t.Fatalf("CLI and benchmark requests differ:\ncheck=%v\nbenchmark=%v", check, benchmark)
	}
}
