package eval

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/config"
)

func TestQualityPolicyMustCoverNewlyConfiguredBlockingRules(t *testing.T) {
	_, p := qualityFixture(t)
	cfg := config.Defaults()
	rc := cfg.Rules["narrating-comment"]
	enabled := true
	rc.Enabled, rc.Severity = &enabled, "error"
	cfg.Rules["narrating-comment"] = rc
	if err := p.CheckConfig(cfg); err == nil {
		t.Fatal("new blocking rule bypassed policy")
	}
}

func qualityFixture(t *testing.T) (Report, QualityPolicy) {
	t.Helper()
	policy, err := LoadQualityPolicy("../../benchmarks/release-policy.json")
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.Repeat("a", 64)
	policy.Review = &CorpusReview{Reviewer: "independent test reviewer", Evidence: "test review record", Independent: true, CorpusSHA256: hash}
	report := Report{Version: 1, Mode: "git", Grouping: "configured", Provenance: &Provenance{CorpusSHA256: hash}}
	evaluated := true
	for id := range policy.Rules {
		report.Rules = append(report.Rules, RuleReport{Rule: id, Threshold: .9})
		for i := 0; i < 120; i++ {
			label, score := "negative", .1
			if i < 20 {
				label, score = "positive", .99
			}
			report.Cases = append(report.Cases, ScoredCase{ID: fmt.Sprintf("%s-%d", id, i), Rule: id, Expected: label, Score: score, Evaluated: &evaluated})
		}
	}
	return report, policy
}

func TestAbsoluteQualityGateCannotPassZeroRecallOrOnlyFalsePositives(t *testing.T) {
	for _, mode := range []string{"zero detections", "only false positives", "small sample", "false positives above target", "misses above target"} {
		t.Run(mode, func(t *testing.T) {
			r, p := qualityFixture(t)
			positives, negatives := 0, 0
			for i := range r.Cases {
				c := &r.Cases[i]
				if c.Expected == "positive" {
					positives++
				} else {
					negatives++
				}
				switch mode {
				case "zero detections":
					c.Score = 0
				case "only false positives":
					if c.Expected == "positive" {
						c.Score = 0
					} else {
						c.Score = 1
					}
				case "false positives above target":
					if c.Expected == "negative" && negatives <= 2 {
						c.Score = 1
					}
				case "misses above target":
					if c.Expected == "positive" && positives <= 5 {
						c.Score = 0
					}
				}
			}
			if mode == "small sample" {
				r.Cases = r.Cases[1:]
			}
			if err := CheckQualityPolicy(io.Discard, r, p); err == nil {
				t.Fatal("unfit release passed")
			}
		})
	}
}

func TestAbsoluteQualityGateAcceptsExactApprovedBoundaries(t *testing.T) {
	r, p := qualityFixture(t)
	for _, rule := range r.Rules {
		positives, negatives := 0, 0
		for i := range r.Cases {
			c := &r.Cases[i]
			if c.Rule != rule.Rule {
				continue
			}
			if c.Expected == "positive" {
				positives++
				if positives <= 4 {
					c.Score = 0
				}
			} else {
				negatives++
				if negatives == 1 {
					c.Score = 1
				}
			}
		}
	}
	if err := CheckQualityPolicy(io.Discard, r, p); err != nil {
		t.Fatal(err)
	}
}

func TestAbsoluteQualityGateRequiresIndependentReviewAndProductionInputs(t *testing.T) {
	for _, mode := range []string{"unreviewed", "self-reviewed", "stale review", "no provenance", "isolated", "experiment", "legacy cases", "missing blocking rule"} {
		t.Run(mode, func(t *testing.T) {
			r, p := qualityFixture(t)
			switch mode {
			case "unreviewed":
				p.Review = nil
			case "self-reviewed":
				p.Review.Independent = false
			case "stale review":
				p.Review.CorpusSHA256 = strings.Repeat("b", 64)
			case "no provenance":
				r.Provenance = nil
			case "isolated":
				r.Grouping = "isolated"
			case "experiment":
				r.Provenance.ExperimentSHA256 = "override"
			case "legacy cases":
				r.Cases[0].Evaluated = nil
			case "missing blocking rule":
				delete(p.Rules, "swallowed-cancellation")
			}
			if err := CheckQualityPolicy(io.Discard, r, p); err == nil {
				t.Fatal("unverified release passed")
			}
		})
	}
}
