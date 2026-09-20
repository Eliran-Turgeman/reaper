package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTaskPrecedenceAndPRSanitizing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(path, []byte(`{"pull_request":{"title":"Fix retries","body":"Keep errors.<!-- private -->\n<!-- generated:start -->bot content<!-- generated:end -->"},"comment":{"body":"unrelated"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == "GITHUB_EVENT_PATH" {
			return path
		}
		return ""
	}
	task, source, err := resolveTask(checkOptions{taskFromPR: true}, false, getenv)
	if err != nil || task != "Fix retries\n\nKeep errors." || source != "GitHub PR title and body" {
		t.Fatalf("%q %q %v", task, source, err)
	}
	task, source, err = resolveTask(checkOptions{task: "explicit", taskFromPR: true, taskFile: "missing"}, true, getenv)
	if err != nil || task != "explicit" || source != "--task" {
		t.Fatal(task, source, err)
	}
	task, source, err = resolveTask(checkOptions{}, false, getenv)
	if err != nil || task != "" || source != "none" {
		t.Fatal(task, source, err)
	}
	if _, _, err := resolveTask(checkOptions{taskFile: "missing"}, false, getenv); err == nil {
		t.Fatal("missing task file ignored")
	}
}
