package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Eliran-Turgeman/repear/internal/rules"
	"gopkg.in/yaml.v3"
)

const FileName = ".reaper.yaml"

const (
	ProviderTypeSafe   = "typesafe"
	ProviderOpenRouter = "openrouter"
)

type RuleConfig struct {
	Enabled      *bool    `yaml:"enabled,omitempty"`
	Threshold    float64  `yaml:"threshold,omitempty"`
	Severity     string   `yaml:"severity,omitempty"`
	Include      []string `yaml:"include,omitempty"`
	Exclude      []string `yaml:"exclude,omitempty"`
	thresholdSet bool
}

func (c *RuleConfig) UnmarshalYAML(node *yaml.Node) error {
	type rawRuleConfig struct {
		Enabled   *bool    `yaml:"enabled,omitempty"`
		Threshold *float64 `yaml:"threshold,omitempty"`
		Severity  string   `yaml:"severity,omitempty"`
		Include   []string `yaml:"include,omitempty"`
		Exclude   []string `yaml:"exclude,omitempty"`
	}
	var raw rawRuleConfig
	allowed := map[string]bool{
		"enabled": true, "threshold": true, "severity": true, "include": true, "exclude": true,
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if !allowed[node.Content[i].Value] {
			return fmt.Errorf("field %s not found in type config.RuleConfig", node.Content[i].Value)
		}
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}
	c.Enabled, c.Severity, c.Include, c.Exclude = raw.Enabled, raw.Severity, raw.Include, raw.Exclude
	if raw.Threshold != nil {
		c.Threshold = *raw.Threshold
		c.thresholdSet = true
	}
	return nil
}

type CacheConfig struct {
	Enabled *bool  `yaml:"enabled,omitempty"`
	Dir     string `yaml:"dir,omitempty"`
}

type Config struct {
	Version        int                   `yaml:"version"`
	Provider       string                `yaml:"provider"`
	Model          string                `yaml:"model"`
	BaseURL        string                `yaml:"base_url,omitempty"`
	RequestTimeout time.Duration         `yaml:"-"`
	TimeoutText    string                `yaml:"request_timeout,omitempty"`
	Concurrency    int                   `yaml:"concurrency,omitempty"`
	Exclude        []string              `yaml:"exclude,omitempty"`
	Cache          CacheConfig           `yaml:"cache,omitempty"`
	Rules          map[string]RuleConfig `yaml:"rules"`
}

func Defaults() Config {
	enabled := true
	cacheEnabled := true
	cfg := Config{
		Version: 1, Provider: ProviderTypeSafe, Model: DefaultModel(ProviderTypeSafe),
		RequestTimeout: 30 * time.Second,
		TimeoutText:    "30s", Concurrency: 4,
		Exclude: []string{"vendor/**", "generated/**", "dist/**", "node_modules/**", "**/*.lock"},
		Cache:   CacheConfig{Enabled: &cacheEnabled},
		Rules:   map[string]RuleConfig{},
	}
	for _, rule := range rules.All() {
		cfg.Rules[rule.ID] = RuleConfig{
			Enabled: &enabled, Threshold: rule.DefaultThreshold, Severity: string(rule.DefaultSeverity),
			thresholdSet: true,
		}
	}
	return cfg
}

func Load(start string) (Config, string, error) {
	cfg := Defaults()
	path, found, err := Find(start)
	if err != nil {
		return Config{}, "", err
	}
	if !found {
		return cfg, "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, path, fmt.Errorf("read config: %w", err)
	}
	var raw Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil {
		return Config{}, path, fmt.Errorf("parse config: %w", err)
	}
	if raw.Provider != "" && raw.Model == "" {
		cfg.Model = DefaultModel(raw.Provider)
	}
	merge(&cfg, raw)
	if err := cfg.Validate(); err != nil {
		return Config{}, path, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, path, nil
}

func Find(start string) (string, bool, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false, err
	}
	for {
		path := filepath.Join(current, FileName)
		_, err := os.Stat(path)
		if err == nil {
			return path, true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}
		current = parent
	}
}

func merge(dst *Config, src Config) {
	if src.Version != 0 {
		dst.Version = src.Version
	}
	if src.Provider != "" {
		dst.Provider = src.Provider
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.BaseURL != "" {
		dst.BaseURL = src.BaseURL
	}
	if src.TimeoutText != "" {
		dst.TimeoutText = src.TimeoutText
	}
	if src.Concurrency != 0 {
		dst.Concurrency = src.Concurrency
	}
	if src.Exclude != nil {
		dst.Exclude = src.Exclude
	}
	if src.Cache.Enabled != nil {
		dst.Cache.Enabled = src.Cache.Enabled
	}
	if src.Cache.Dir != "" {
		dst.Cache.Dir = src.Cache.Dir
	}
	for id, value := range src.Rules {
		base := dst.Rules[id]
		if value.Enabled != nil {
			base.Enabled = value.Enabled
		}
		if value.thresholdSet {
			base.Threshold = value.Threshold
		}
		if value.Severity != "" {
			base.Severity = value.Severity
		}
		if value.Include != nil {
			base.Include = value.Include
		}
		if value.Exclude != nil {
			base.Exclude = value.Exclude
		}
		dst.Rules[id] = base
	}
}

func (c *Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported version %d", c.Version)
	}
	if c.Provider != ProviderTypeSafe && c.Provider != ProviderOpenRouter {
		return fmt.Errorf("provider must be %q or %q, got %q", ProviderTypeSafe, ProviderOpenRouter, c.Provider)
	}
	if c.Model == "" {
		return errors.New("model is required")
	}
	timeout, err := time.ParseDuration(c.TimeoutText)
	if err != nil || timeout <= 0 {
		return fmt.Errorf("request_timeout must be a positive duration")
	}
	c.RequestTimeout = timeout
	if c.Concurrency < 1 || c.Concurrency > 32 {
		return fmt.Errorf("concurrency must be between 1 and 32")
	}
	for id, rc := range c.Rules {
		if _, ok := rules.Get(id); !ok {
			return fmt.Errorf("unknown rule %q", id)
		}
		if rc.Threshold < 0 || rc.Threshold > 1 {
			return fmt.Errorf("rule %s threshold must be between 0 and 1", id)
		}
		if _, err := rules.ValidateSeverity(rc.Severity); err != nil {
			return fmt.Errorf("rule %s: %w", id, err)
		}
	}
	return nil
}

func (c Config) YAML() ([]byte, error) {
	return yaml.Marshal(c)
}

func DefaultModel(provider string) string {
	if provider == ProviderOpenRouter {
		return "typesafe/jev-1.13"
	}
	return "jev-latest"
}

func APIKeyEnv(provider string) string {
	if provider == ProviderOpenRouter {
		return "OPENROUTER_API_KEY"
	}
	return "TYPESAFE_API_KEY"
}
