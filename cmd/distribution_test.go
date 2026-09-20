package cmd

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDistributionIncludesLicense(t *testing.T) {
	for path, want := range map[string]string{
		"../LICENSE":          "Permission is hereby granted, free of charge",
		"../README.md":        "[MIT License](LICENSE)",
		"../.goreleaser.yaml": "- LICENSE",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("%s missing license metadata", path)
		}
	}
}

func TestCIKeepsReleaseSeparate(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"pull_request:", "branches: [main]", "go test ./...", "go vet ./...", "go build ./cmd/reaper", "ubuntu-latest", "windows-latest", "macos-latest"} {
		if !strings.Contains(string(data), required) {
			t.Errorf("CI missing %s", required)
		}
	}
	if strings.Contains(string(data), "contents: write") {
		t.Fatal("normal CI must not publish releases")
	}
}

func TestManualReleaseValidationCannotPublish(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		On   map[string]any `yaml:"on"`
		Jobs map[string]struct {
			Steps []struct {
				Uses string `yaml:"uses"`
				If   string `yaml:"if"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	if _, ok := workflow.On["workflow_dispatch"]; !ok {
		t.Fatal("release checks must support manual validation")
	}
	found := false
	for _, job := range workflow.Jobs {
		for _, step := range job.Steps {
			if strings.HasPrefix(step.Uses, "goreleaser/goreleaser-action@") {
				found = true
				if step.If != "github.event_name == 'push' && startsWith(github.ref, 'refs/tags/')" {
					t.Fatal("publishing must require a tag push, excluding manual runs even on tags")
				}
			}
		}
	}
	if !found {
		t.Fatal("release publishing step missing")
	}
}

func TestPrivacyDocumentationCoversSentDataAndStorage(t *testing.T) {
	data, err := os.ReadFile("../docs/privacy.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"file path", "language", "diff", "current code", "previous code", "surrounding code", "task context", "api.typesafe.ai", "openrouter.ai", ".git/reaper-cache", "Authorization", "--no-cache"} {
		if !strings.Contains(string(data), required) {
			t.Errorf("missing privacy disclosure %s", required)
		}
	}
}

func TestIntegrationsRequireTestsAndReruns(t *testing.T) {
	for _, path := range []string{"AGENTS.md", "codex/AGENTS.md", "claude/SKILL.md", "cursor/reaper.mdc"} {
		data, err := os.ReadFile("../integrations/" + path)
		if err != nil {
			t.Fatal(err)
		}
		for _, text := range []string{"tests", "reaper check --format agent", "rerun", "complete: false"} {
			if !strings.Contains(string(data), text) {
				t.Errorf("%s missing %s", path, text)
			}
		}
	}
}
