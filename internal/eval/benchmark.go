package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Eliran-Turgeman/reaper/internal/cache"
	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/Eliran-Turgeman/reaper/internal/runner"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
)

type PatchCase struct {
	ID         string `json:"id"`
	Task       string `json:"task"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Diff       string `json:"diff"`
	File       string `json:"file"`
	Rule       string `json:"rule"`
	Expected   string `json:"expected"`
	Rationale  string `json:"rationale"`
	Provenance string `json:"provenance"`
	Split      string `json:"split"`
}

type ScoredCase struct {
	Requests       []RequestRecord      `json:"requests,omitempty"`
	Evaluated      *bool                `json:"evaluated,omitempty"`
	CoverageReason string               `json:"coverage_reason,omitempty"`
	Observations   []runner.Observation `json:"observations,omitempty"`
	ID             string               `json:"id"`
	Rule           string               `json:"rule"`
	Expected       string               `json:"expected"`
	Score          float64              `json:"score"`
	Split          string               `json:"split"`
}

func LoadPatches(dir string) ([]PatchCase, error) {
	data, err := os.ReadFile(filepath.Join(dir, "cases.json"))
	if err != nil {
		return nil, err
	}
	var cases []PatchCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, c := range cases {
		_, ruleOK := rules.Get(c.Rule)
		if c.ID == "" || seen[c.ID] || c.Task == "" || c.Before == "" || c.After == "" || c.Diff == "" || !semantic.IsSupportedSource(c.File) || !ruleOK || c.Rationale == "" || c.Provenance == "" {
			return nil, fmt.Errorf("invalid or duplicate benchmark case %q", c.ID)
		}
		if c.Expected != "positive" && c.Expected != "negative" {
			return nil, fmt.Errorf("invalid label for %s", c.ID)
		}
		if c.Split != "train" && c.Split != "dev" && c.Split != "hidden-test" {
			return nil, fmt.Errorf("invalid split for %s", c.ID)
		}
		seen[c.ID] = true
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("empty benchmark corpus")
	}
	return cases, nil
}

func RunBenchmark(ctx context.Context, client decision.Evaluator, dir string, cfg config.Config) (Report, error) {
	cases, err := LoadPatches(dir)
	if err != nil {
		return Report{}, err
	}
	report := Report{Version: 1, Provider: cfg.Provider, Model: cfg.Model, Provenance: provenance("snippet-v1", cases, cfg)}
	for _, c := range cases {
		caseCfg := cfg
		caseCfg.Rules = map[string]config.RuleConfig{}
		enabled := true
		rc := cfg.Rules[c.Rule]
		rc.Enabled = &enabled
		rc.Threshold = 0
		caseCfg.Rules[c.Rule] = rc
		recorder := &recordingEvaluator{Evaluator: client}
		var observations []runner.Observation
		engine := runner.Runner{Config: caseCfg, Client: recorder, Cache: cache.Disabled{}, Observe: func(o runner.Observation) { observations = append(observations, o) }}
		unit := semantic.Unit{FilePath: c.File, Language: semantic.Language(c.File), OldContent: c.Before, NewContent: c.After, Diff: c.Diff, SurroundingCode: c.After, StartLine: 1, EndLine: strings.Count(c.After, "\n") + 1, ExistingModified: true, IsTest: semantic.IsTestFile(c.File), ContainsComments: semantic.ContainsComment(semantic.Language(c.File), c.After)}
		result, err := engine.Run(ctx, []semantic.Unit{unit}, c.Task)
		if err != nil {
			return Report{}, err
		}
		if !result.Complete {
			return Report{}, fmt.Errorf("benchmark %s incomplete: %v", c.ID, result.Skipped)
		}
		if len(result.Diagnostics) != 1 {
			return Report{}, fmt.Errorf("benchmark case %s does not exercise its rule", c.ID)
		}
		report.Cases = append(report.Cases, ScoredCase{ID: c.ID, Rule: c.Rule, Expected: c.Expected, Score: result.Diagnostics[0].Confidence, Split: c.Split, Requests: recorder.Records(), Observations: observations})
	}
	for id, rc := range cfg.Rules {
		metric := Metrics(id, report.Cases, rc.Threshold)
		if metric.Examples > 0 {
			report.Rules = append(report.Rules, metric)
		}
	}
	sort.Slice(report.Rules, func(i, j int) bool { return report.Rules[i].Rule < report.Rules[j].Rule })
	return report, nil
}

func Metrics(rule string, cases []ScoredCase, threshold float64) RuleReport {
	m := RuleReport{Rule: rule, Threshold: threshold}
	for _, c := range cases {
		if c.Rule != rule {
			continue
		}
		m.Examples++
		predicted := (c.Evaluated == nil || *c.Evaluated) && c.Score >= threshold
		if c.Expected == "positive" {
			m.AveragePositiveScore += c.Score
			if predicted {
				m.TruePositive++
			} else {
				m.FalseNegative++
			}
		} else {
			m.AverageNegativeScore += c.Score
			if predicted {
				m.FalsePositive++
			} else {
				m.TrueNegative++
			}
		}
	}
	m.Precision = ratio(m.TruePositive, m.TruePositive+m.FalsePositive)
	m.Recall = ratio(m.TruePositive, m.TruePositive+m.FalseNegative)
	m.FalsePositiveRate = ratio(m.FalsePositive, m.FalsePositive+m.TrueNegative)
	m.AveragePositiveScore = ratioFloat(m.AveragePositiveScore, m.TruePositive+m.FalseNegative)
	m.AverageNegativeScore = ratioFloat(m.AverageNegativeScore, m.TrueNegative+m.FalsePositive)
	return m
}
