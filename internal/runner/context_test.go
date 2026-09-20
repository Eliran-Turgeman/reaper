package runner

import (
	"context"
	"github.com/Eliran-Turgeman/reaper/internal/cache"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTargetedContextHonorsTrackedFilesAndExclusions(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init").CombinedOutput(); err != nil {
		t.Fatalf("%s %v", out, err)
	}
	for name, code := range map[string]string{"use.go": "var x Plugin", "other.go": "var y Unrelated", "secret.go": "var secret Plugin"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(code), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("git", "-C", dir, "add", "use.go", "other.go").CombinedOutput(); err != nil {
		t.Fatalf("%s %v", out, err)
	}
	r := Runner{Config: allRulesConfig(), Root: dir}
	rule, _ := rules.Get("unused-extensibility-point")
	evidence, err := r.repositoryContext(context.Background(), semantic.Unit{NewContent: "type Plugin interface {}"}, rule)
	if err != nil || !strings.Contains(evidence, "use.go:1") || strings.Contains(evidence, "secret.go") || strings.Contains(evidence, "other.go") {
		t.Fatal(evidence, err)
	}
	r.Config.Exclude = append(r.Config.Exclude, "use.go")
	for id, rc := range r.Config.Rules {
		enabled := id == "unused-extensibility-point" || id == "silent-failure-fallback"
		rc.Enabled = &enabled
		r.Config.Rules[id] = rc
	}
	client := &mockClient{}
	r.Client = client
	r.Cache = cache.Disabled{}
	if _, err := r.Run(context.Background(), []semantic.Unit{{FilePath: "new.go", NewContent: "type Plugin interface {}"}}, "Add a plugin boundary"); err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 2 {
		t.Fatal("context rules were not isolated")
	}
	for _, request := range client.requests {
		retrieved := strings.Contains(request.State, "REPOSITORY SEARCH EVIDENCE")
		for _, question := range request.Questions {
			if retrieved != strings.HasPrefix(question.ID, "unused-extensibility-point:") {
				t.Fatal("context leaked to local rule")
			}
		}
	}
	evidence, err = r.repositoryContext(context.Background(), semantic.Unit{NewContent: "type Plugin interface {}"}, rule)
	if err != nil || strings.Contains(evidence, "use.go:1") {
		t.Fatal(evidence, err)
	}
}
