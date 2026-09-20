package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Eliran-Turgeman/reaper/internal/cache"
	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/diff"
	repogit "github.com/Eliran-Turgeman/reaper/internal/git"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/Eliran-Turgeman/reaper/internal/runner"
)

// GitCase stores complete repository snapshots, or the legacy single-file patch.
// Missing paths in AfterFiles are deletions; empty strings are existing empty files.
type GitCase struct {
	PatchCase
	BeforeFiles map[string]string `json:"before_files,omitempty"`
	AfterFiles  map[string]string `json:"after_files,omitempty"`
}

func LoadGitCases(dir string) ([]GitCase, error) {
	data, err := os.ReadFile(filepath.Join(dir, "cases.json"))
	if err != nil {
		return nil, err
	}
	var cases []GitCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("empty Git benchmark corpus")
	}
	seen := map[string]bool{}
	for i := range cases {
		c := &cases[i]
		_, known := rules.Get(c.Rule)
		if c.ID == "" || seen[c.ID] || !known || c.Task == "" || c.Provenance == "" || c.Rationale == "" || (c.Expected != "positive" && c.Expected != "negative") || (c.Split != "dev" && c.Split != "train" && c.Split != "hidden-test") {
			return nil, fmt.Errorf("invalid or duplicate Git benchmark case %q", c.ID)
		}
		seen[c.ID] = true
		if c.BeforeFiles == nil && c.AfterFiles == nil {
			if c.File == "" || c.Before == "" || c.After == "" {
				return nil, fmt.Errorf("case %s requires snapshots or a before/after file", c.ID)
			}
			c.BeforeFiles = map[string]string{c.File: c.Before}
			c.AfterFiles = map[string]string{c.File: c.After}
		} else if c.BeforeFiles == nil || c.AfterFiles == nil || c.Before != "" || c.After != "" {
			return nil, fmt.Errorf("case %s must supply both snapshots without legacy before/after snippets", c.ID)
		}
		for _, files := range []map[string]string{c.BeforeFiles, c.AfterFiles} {
			folded := map[string]bool{}
			for name := range files {
				lower := strings.ToLower(name)
				if !safeFixturePath(name) || folded[lower] {
					return nil, fmt.Errorf("case %s has unsafe or conflicting path %q", c.ID, name)
				}
				folded[lower] = true
			}
		}
	}
	return cases, nil
}

func safeFixturePath(name string) bool {
	if !filepath.IsLocal(name) || path.Clean(name) != name || strings.ContainsAny(name, "\\:\x00\r\n") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.EqualFold(part, ".git") || strings.TrimRight(part, ". ") != part {
			return false
		}
	}
	return true
}

func fixtureGitEnv() []string {
	var env []string
	for _, item := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(item), "GIT_") {
			env = append(env, item)
		}
	}
	return append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0", "GIT_LITERAL_PATHSPECS=1")
}

// materializeGitCase snapshots the before tree in the index, then collects the
// real unstaged diff. No commits, hooks, source execution or network are needed.
func materializeGitCase(ctx context.Context, c GitCase) (string, string, func(), error) {
	root, err := os.MkdirTemp("", "reaper-benchmark-")
	if err != nil {
		return "", "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	fail := func(err error) (string, string, func(), error) { cleanup(); return "", "", nil, err }
	env := fixtureGitEnv()
	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("fixture git %v: %w: %s", args, err, out)
		}
		return nil
	}
	write := func(files map[string]string) error {
		for _, name := range sortedPaths(files) {
			if !safeFixturePath(name) {
				return fmt.Errorf("unsafe fixture path %q", name)
			}
			full := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
				return err
			}
			if err := os.WriteFile(full, []byte(files[name]), 0600); err != nil {
				return err
			}
		}
		return nil
	}
	if err := run("init", "--quiet", "--template="); err != nil {
		return fail(err)
	}
	for key, value := range map[string]string{"core.autocrlf": "false", "core.quotePath": "false", "diff.renames": "true"} {
		if err := run("config", key, value); err != nil {
			return fail(err)
		}
	}
	if err := write(c.BeforeFiles); err != nil {
		return fail(err)
	}
	for _, name := range sortedPaths(c.BeforeFiles) {
		if err := run("add", "--force", "--", name); err != nil {
			return fail(err)
		}
	}
	for _, name := range sortedPaths(c.BeforeFiles) {
		if _, exists := c.AfterFiles[name]; !exists {
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(name))); err != nil {
				return fail(err)
			}
		}
	}
	if err := write(c.AfterFiles); err != nil {
		return fail(err)
	}
	for _, name := range sortedPaths(c.AfterFiles) {
		if _, existed := c.BeforeFiles[name]; !existed {
			if err := run("add", "--force", "--intent-to-add", "--", name); err != nil {
				return fail(err)
			}
		}
	}
	collector := repogit.CommandCollector{Dir: root, Env: env}
	patch, _, err := collector.Diff(ctx, repogit.Options{})
	if err != nil {
		return fail(err)
	}
	return root, patch, cleanup, nil
}

func sortedPaths(files map[string]string) []string {
	paths := make([]string, 0, len(files))
	for name := range files {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	return paths
}

func RunGitBenchmark(ctx context.Context, client decision.Evaluator, dir string, cfg config.Config, grouping string) (Report, error) {
	return RunGitExperiment(ctx, client, dir, cfg, grouping, nil)
}

func RunGitExperiment(ctx context.Context, client decision.Evaluator, dir string, cfg config.Config, grouping string, experiment *Experiment) (Report, error) {
	if experiment != nil {
		if err := experiment.validate(); err != nil {
			return Report{}, err
		}
	}
	if grouping != "configured" && grouping != "isolated" {
		return Report{}, fmt.Errorf("benchmark grouping must be configured or isolated")
	}
	cases, err := LoadGitCases(dir)
	if err != nil {
		return Report{}, err
	}
	report := Report{Version: 1, Provider: cfg.Provider, Model: cfg.Model, Mode: "git", Grouping: grouping, Provenance: provenance("git-v2", cases, cfg)}
	if experiment != nil {
		report.Provenance.ExperimentSHA256 = fingerprint(experiment)
	}
	for _, c := range cases {
		scored, err := runGitCaseExperiment(ctx, client, c, cfg, grouping, experiment)
		if err != nil {
			return Report{}, fmt.Errorf("benchmark %s: %w", c.ID, err)
		}
		report.Cases = append(report.Cases, scored)
	}
	for _, rule := range rules.All() {
		m := Metrics(rule.ID, report.Cases, cfg.Rules[rule.ID].Threshold)
		if m.Examples > 0 {
			report.Rules = append(report.Rules, m)
		}
	}
	return report, nil
}

func runGitCase(ctx context.Context, client decision.Evaluator, c GitCase, cfg config.Config, grouping string) (ScoredCase, error) {
	return runGitCaseExperiment(ctx, client, c, cfg, grouping, nil)
}

func runGitCaseExperiment(ctx context.Context, client decision.Evaluator, c GitCase, cfg config.Config, grouping string, experiment *Experiment) (ScoredCase, error) {
	evidence, err := experimentEvidence(c, experiment)
	if err != nil {
		return ScoredCase{}, err
	}
	root, patch, cleanup, err := materializeGitCase(ctx, c)
	if err != nil {
		return ScoredCase{}, err
	}
	defer cleanup()
	units, err := diff.Units(root, patch, 6)
	if err != nil {
		return ScoredCase{}, err
	}
	if experiment != nil && (experiment.Context == "matched" || experiment.Context == "targeted") {
		if err := addTargetedContext(c, patch, units, experiment.Context == "targeted"); err != nil {
			return ScoredCase{}, err
		}
	}
	if grouping == "isolated" {
		rc := cfg.Rules[c.Rule]
		enabled := true
		rc.Enabled = &enabled
		cfg.Rules = map[string]config.RuleConfig{c.Rule: rc}
	}
	evaluated := false
	recorder := &recordingEvaluator{Evaluator: client}
	var evaluator decision.Evaluator = recorder
	if experiment != nil {
		evaluator = &experimentEvaluator{Evaluator: recorder, experiment: experiment, evidence: evidence}
	}
	scored := ScoredCase{ID: c.ID, Rule: c.Rule, Expected: c.Expected, Split: c.Split, Evaluated: &evaluated}
	engine := runner.Runner{Root: root, GitEnv: fixtureGitEnv(), Config: cfg, Client: evaluator, Cache: cache.Disabled{}, Observe: func(o runner.Observation) {
		if experiment != nil {
			for i, signal := range o.Signals {
				if text, ok := experiment.Questions[o.Rule+":"+signal.ID]; ok {
					o.Signals[i].Evidence = text
				}
			}
		}
		if o.Rule == c.Rule {
			evaluated = true
			scored.Observations = append(scored.Observations, o)
			if o.Score > scored.Score {
				scored.Score = o.Score
			}
		}
	}}
	result, err := engine.Run(ctx, units, c.Task)
	if err != nil {
		return ScoredCase{}, err
	}
	if !result.Complete {
		return ScoredCase{}, fmt.Errorf("analysis incomplete: %v", result.Skipped)
	}
	if !evaluated {
		scored.CoverageReason = "no evaluation of labeled rule after extraction, configuration and applicability checks"
	}
	scored.Requests = recorder.Records()
	return scored, nil
}
