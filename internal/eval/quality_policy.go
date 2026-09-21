package eval

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

type QualityRequirement struct {
	MinimumRecall            float64 `json:"minimum_recall"`
	MaximumFalsePositiveRate float64 `json:"maximum_false_positive_rate"`
	MinimumPositives         int     `json:"minimum_positives"`
	MinimumNegatives         int     `json:"minimum_negatives"`
}

// CorpusReview is a human attestation, bound to the exact loaded corpus.
// Software checks the binding; it cannot independently verify reviewer identity.
type CorpusReview struct {
	Reviewer     string `json:"reviewer"`
	Evidence     string `json:"evidence"`
	Independent  bool   `json:"independent"`
	CorpusSHA256 string `json:"corpus_sha256"`
}

type QualityPolicy struct {
	Version     int                           `json:"version"`
	CorpusSplit string                        `json:"corpus_split"`
	Rules       map[string]QualityRequirement `json:"rules"`
	Review      *CorpusReview                 `json:"review"`
}

func (p QualityPolicy) CheckConfig(cfg config.Config) error {
	for id, rc := range cfg.Rules {
		if rc.Enabled != nil && *rc.Enabled && rc.Severity == string(rules.SeverityError) {
			if _, ok := p.Rules[id]; !ok {
				return fmt.Errorf("quality policy is missing configured blocking rule %s", id)
			}
		}
	}
	return nil
}

func LoadQualityPolicy(file string) (QualityPolicy, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return QualityPolicy{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var policy QualityPolicy
	if err := decoder.Decode(&policy); err != nil {
		return QualityPolicy{}, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return QualityPolicy{}, fmt.Errorf("quality policy must contain one JSON object")
	}
	return policy, policy.validate()
}

func (p QualityPolicy) validate() error {
	if p.Version != 1 {
		return fmt.Errorf("quality policy requires version 1")
	}
	if p.CorpusSplit != "hidden-test" {
		return fmt.Errorf("quality policy requires corpus_split hidden-test")
	}
	for _, rule := range rules.All() {
		if rule.DefaultSeverity == rules.SeverityError {
			if _, ok := p.Rules[rule.ID]; !ok {
				return fmt.Errorf("quality policy is missing blocking rule %s", rule.ID)
			}
		}
	}
	for id, r := range p.Rules {
		_, known := rules.Get(id)
		if !known || math.IsNaN(r.MinimumRecall) || math.IsNaN(r.MaximumFalsePositiveRate) || r.MinimumRecall < 0 || r.MinimumRecall > 1 || r.MaximumFalsePositiveRate < 0 || r.MaximumFalsePositiveRate > 1 || r.MinimumPositives < 1 || r.MinimumNegatives < 1 {
			return fmt.Errorf("invalid quality requirement for %s", id)
		}
	}
	return nil
}

// Ready fails before inference when independent review has not been supplied.
func (p QualityPolicy) Ready() error {
	if err := p.validate(); err != nil {
		return err
	}
	if p.Review == nil || !p.Review.Independent || strings.TrimSpace(p.Review.Reviewer) == "" || strings.TrimSpace(p.Review.Evidence) == "" {
		return fmt.Errorf("quality gate: independent corpus review is missing; synthetic development results cannot authorize a release")
	}
	hash, err := hex.DecodeString(p.Review.CorpusSHA256)
	if err != nil || len(hash) != 32 {
		return fmt.Errorf("quality gate: review requires a valid corpus SHA-256")
	}
	return nil
}

func CheckQualityPolicy(w io.Writer, report Report, policy QualityPolicy) error {
	if err := policy.Ready(); err != nil {
		return err
	}
	if report.Mode != "git" || report.Grouping != "configured" || report.Provenance == nil || report.Provenance.ExperimentSHA256 != "" {
		return fmt.Errorf("quality gate requires a complete configured Git benchmark of production rules, without experiment overrides")
	}
	if report.Provenance.CorpusSHA256 != policy.Review.CorpusSHA256 {
		return fmt.Errorf("quality gate: reviewed corpus fingerprint mismatch")
	}
	seen := map[string]bool{}
	for _, c := range report.Cases {
		if c.ID == "" || seen[c.ID] || c.Evaluated == nil || c.Split != policy.CorpusSplit || c.Decision == "insufficient-evidence" || (c.Expected != "positive" && c.Expected != "negative") || math.IsNaN(c.Score) || c.Score < 0 || c.Score > 1 {
			return fmt.Errorf("quality gate: invalid or duplicate scored case %s", c.ID)
		}
		seen[c.ID] = true
	}
	thresholds := map[string]float64{}
	for _, r := range report.Rules {
		thresholds[r.Rule] = r.Threshold
	}
	failed := false
	for _, rule := range rules.All() {
		required, ok := policy.Rules[rule.ID]
		if !ok {
			continue
		}
		threshold, exists := thresholds[rule.ID]
		if !exists {
			return fmt.Errorf("quality gate: missing required rule %s", rule.ID)
		}
		m := Metrics(rule.ID, report.Cases, threshold)
		positives, negatives := m.TruePositive+m.FalseNegative, m.TrueNegative+m.FalsePositive
		fmt.Fprintf(w, "%s recall=%.4f (min %.4f) false_positive_rate=%.4f (max %.4f) positives=%d (min %d) negatives=%d (min %d)\n", rule.ID, m.Recall, required.MinimumRecall, m.FalsePositiveRate, required.MaximumFalsePositiveRate, positives, required.MinimumPositives, negatives, required.MinimumNegatives)
		if positives < required.MinimumPositives || negatives < required.MinimumNegatives || m.Recall < required.MinimumRecall || m.FalsePositiveRate > required.MaximumFalsePositiveRate {
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("absolute release-quality requirements are not met")
	}
	return nil
}
