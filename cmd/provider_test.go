package cmd

import (
	"testing"

	"github.com/Eliran-Turgeman/repear/internal/config"
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
