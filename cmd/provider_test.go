package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/config"
)

func TestApplyProviderOverridesUsesProviderModelDefault(t *testing.T) {
	cfg := config.Defaults()
	if err := applyProviderOverrides(&cfg, config.ProviderOpenRouter, ""); err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != config.ProviderOpenRouter || cfg.Model != "typesafe/jev-1.13" {
		t.Fatalf("unexpected overrides: %#v", cfg)
	}
}

func TestCheckAllRejectsDiffModes(t *testing.T) {
	for _, args := range [][]string{
		{"check", "--all", "--staged"},
		{"check", "--all", "--diff", "main"},
	} {
		var output bytes.Buffer
		command := NewWith(App{Out: &output, ErrOut: &output, Getenv: func(string) string { return "" }})
		command.SetArgs(args)
		err := command.Execute()
		if err == nil || !strings.Contains(err.Error(), "--all cannot be used") {
			t.Fatalf("args %v returned %v, want --all conflict", args, err)
		}
	}
}

func TestApplyProviderOverridesHonorsExplicitModelAndRejectsProvider(t *testing.T) {
	cfg := config.Defaults()
	if err := applyProviderOverrides(&cfg, config.ProviderOpenRouter, "typesafe/jev-custom"); err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "typesafe/jev-custom" {
		t.Fatalf("explicit model was not preserved: %q", cfg.Model)
	}
	if err := applyProviderOverrides(&cfg, "unknown", ""); err == nil {
		t.Fatal("expected invalid provider error")
	}
}
