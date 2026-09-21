package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIncompleteAnalysisPolicy(t *testing.T) {
	for _, policy := range []string{"error", "warning", "ignore", "invalid"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, FileName), []byte("incomplete_analysis: "+policy), 0600); err != nil {
			t.Fatal(err)
		}
		cfg, _, err := Load(dir)
		if policy == "invalid" {
			if err == nil {
				t.Fatal("accepted invalid policy")
			}
			continue
		}
		if err != nil || cfg.IncompleteAnalysis != policy {
			t.Fatalf("%s: %v %+v", policy, err, cfg)
		}
	}
}

func TestPacksAndExplicitRulePrecedence(t *testing.T) {
	cfg := Defaults()
	if !*cfg.Rules["silent-failure-fallback"].Enabled || *cfg.Rules["narrating-comment"].Enabled {
		t.Fatal("unexpected default packs")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("packs: [style]\nrules:\n  silent-failure-fallback:\n    enabled: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !*cfg.Rules["narrating-comment"].Enabled || !*cfg.Rules["silent-failure-fallback"].Enabled || *cfg.Rules["weakened-test-assertion"].Enabled {
		t.Fatal("pack or explicit override ignored")
	}
	cfg.Packs = []string{"typo"}
	if cfg.Validate() == nil {
		t.Fatal("accepted unknown pack")
	}
}

func TestLoadMergesOverridesIncludingZeroThreshold(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`version: 1
model: jev-1.13.0
request_timeout: 2s
concurrency: 2
rules:
  narrating-comment:
    enabled: false
    threshold: 0
    severity: error
`)
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, path, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if path == "" || cfg.Model != "jev-1.13.0" || cfg.RequestTimeout != 2*time.Second {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	rule := cfg.Rules["narrating-comment"]
	if *rule.Enabled || rule.Threshold != 0 || rule.Severity != "error" {
		t.Fatalf("override not applied: %#v", rule)
	}
	if _, ok := cfg.Rules["silent-failure-fallback"]; !ok {
		t.Fatal("default rules were not preserved")
	}
}

func TestLoadRejectsUnknownAndInvalidFields(t *testing.T) {
	for name, data := range map[string]string{
		"unknown":  "version: 1\nmodel: jev-latest\nmystery: true\n",
		"nested":   "version: 1\nmodel: jev-latest\nrules:\n  narrating-comment:\n    typo: true\n",
		"timeout":  "version: 1\nmodel: jev-latest\nrequest_timeout: nope\n",
		"rule":     "version: 1\nmodel: jev-latest\nrules:\n  made-up:\n    enabled: true\n",
		"provider": "version: 1\nprovider: mystery\nmodel: jev-latest\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, _, err := Load(dir); err == nil {
				t.Fatal("expected invalid configuration error")
			}
		})
	}
}

func TestLoadSelectsProviderSpecificModelDefault(t *testing.T) {
	dir := t.TempDir()
	data := []byte("version: 1\nprovider: openrouter\n")
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != ProviderOpenRouter || cfg.Model != "typesafe/jev-1.13" {
		t.Fatalf("unexpected provider defaults: %#v", cfg)
	}
	if APIKeyEnv(cfg.Provider) != "OPENROUTER_API_KEY" {
		t.Fatalf("unexpected credential environment variable")
	}
}
