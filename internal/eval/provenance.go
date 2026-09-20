package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"sync"

	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

// Provenance binds an experiment to its inputs, independently of its scores.
type Provenance struct {
	ExperimentSHA256 string `json:"experiment_sha256,omitempty"`
	Protocol         string `json:"protocol"`
	CorpusSHA256     string `json:"corpus_sha256"`
	RulesSHA256      string `json:"rules_sha256"`
	ConfigSHA256     string `json:"config_sha256"`
}

// RequestRecord describes an actual evaluator call (after capability splitting).
// State is hashed, not copied: it can be reproduced from the fixture and protocol.
// Returned model identity is unavailable in the normalized evaluator interface.
type RequestRecord struct {
	SHA256      string              `json:"sha256"`
	StateSHA256 string              `json:"state_sha256"`
	Model       string              `json:"requested_model"`
	Questions   []decision.Question `json:"questions"`
	Scores      map[string]float64  `json:"scores"`
}

func fingerprint(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		// All callers use fixed JSON-serializable input records, not user objects.
		panic(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func provenance(protocol string, corpus any, cfg config.Config) *Provenance {
	type ruleVersion struct {
		ID, Context, Scope string
		Version            int
		Questions          []decision.Question
	}
	var versions []ruleVersion
	for _, rule := range rules.All() {
		versions = append(versions, ruleVersion{rule.ID, rule.Context, string(rule.Scope), rule.Version, rules.Questions([]rules.Rule{rule}, false, false)})
	}
	// Paths, credentials, cache directories and request timing are not rule inputs.
	inputs := struct {
		Rules   map[string]config.RuleConfig
		Exclude []string
	}{cfg.Rules, cfg.Exclude}
	return &Provenance{Protocol: protocol, CorpusSHA256: fingerprint(corpus), RulesSHA256: fingerprint(versions), ConfigSHA256: fingerprint(inputs)}
}

type recordingEvaluator struct {
	decision.Evaluator
	mu      sync.Mutex
	records []RequestRecord
}

func (r *recordingEvaluator) Evaluate(ctx context.Context, request decision.Request) (decision.Response, error) {
	response, err := r.Evaluator.Evaluate(ctx, request)
	if err != nil {
		return response, err
	}
	scores := make(map[string]float64, len(response.Scores))
	for id, score := range response.Scores {
		scores[id] = score
	}
	record := RequestRecord{SHA256: fingerprint(request), StateSHA256: fingerprint(request.State), Model: request.Model, Questions: append([]decision.Question(nil), request.Questions...), Scores: scores}
	r.mu.Lock()
	r.records = append(r.records, record)
	r.mu.Unlock()
	return response, nil
}

func (r *recordingEvaluator) Records() []RequestRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append([]RequestRecord(nil), r.records...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].SHA256 < out[j].SHA256 })
	return out
}
