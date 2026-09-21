package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

// Experiment changes benchmark inputs only; it never changes production rules.
type Experiment struct {
	Version     int                              `json:"version"`
	Name        string                           `json:"name"`
	Context     string                           `json:"context"`
	Questions   map[string]string                `json:"questions,omitempty"`
	Criteria    map[string]decision.NoulCriteria `json:"criteria,omitempty"`
	StateFormat string                           `json:"state_format,omitempty"`
	Signals     map[string][]rules.Signal        `json:"signals,omitempty"`
}

func LoadExperiment(file string) (*Experiment, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var experiment Experiment
	if err := decoder.Decode(&experiment); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("experiment must contain one JSON object")
	}
	if err := experiment.validate(); err != nil {
		return nil, err
	}
	return &experiment, nil
}

func (e Experiment) validate() error {
	if e.Version != 1 || e.Name == "" || (e.Context != "current" && e.Context != "snapshots" && e.Context != "matched" && e.Context != "targeted") {
		return fmt.Errorf("experiment requires version 1, a name, and context current, snapshots, matched or targeted")
	}
	known := map[string]bool{}
	for id, signals := range e.Signals {
		if _, ok := rules.Get(id); !ok || len(signals) == 0 {
			return fmt.Errorf("invalid experiment signal rule %q", id)
		}
		seen := map[string]bool{}
		supports := 0
		for _, signal := range signals {
			if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(signal.ID) || seen[signal.ID] || strings.TrimSpace(signal.Instructions) == "" {
				return fmt.Errorf("invalid experiment signal %s:%s", id, signal.ID)
			}
			seen[signal.ID] = true
			if !signal.Negate {
				supports++
			}
			if err := signal.Criteria.Validate(); err != nil {
				return err
			}
		}
		if supports == 0 {
			return fmt.Errorf("experiment rule %s requires a supporting factual signal", id)
		}
	}
	if e.StateFormat != "" && e.StateFormat != "json-text" && e.StateFormat != "json-object" {
		return fmt.Errorf("unsupported experiment state_format %q", e.StateFormat)
	}
	for _, q := range rules.Questions(e.ruleSet(rules.All()), false, false) {
		known[q.ID] = true
	}
	for id, text := range e.Questions {
		if !known[id] || text == "" {
			return fmt.Errorf("invalid experiment question %q", id)
		}
	}
	for id, criteria := range e.Criteria {
		if !known[id] {
			return fmt.Errorf("invalid experiment criteria question %q", id)
		}
		if err := criteria.Validate(); err != nil {
			return fmt.Errorf("question %s: %w", id, err)
		}
	}
	return nil
}

func (e Experiment) ruleSet(selected []rules.Rule) []rules.Rule {
	for i := range selected {
		if signals, ok := e.Signals[selected[i].ID]; ok {
			selected[i].Signals = signals
			selected[i].Version++
		}
	}
	return selected
}

type experimentEvaluator struct {
	decision.Evaluator
	experiment *Experiment
	evidence   string
}

func (e *experimentEvaluator) Evaluate(ctx context.Context, request decision.Request) (decision.Response, error) {
	request.Questions = append([]decision.Question(nil), request.Questions...)
	for i, q := range request.Questions {
		if text, ok := e.experiment.Questions[q.ID]; ok {
			request.Questions[i].Instructions = text
		}
		if criteria, ok := e.experiment.Criteria[q.ID]; ok {
			request.Questions[i].Criteria = &criteria
		}
	}
	if e.evidence != "" {
		if e.experiment.StateFormat == "" {
			request.State += "\n\nBEFORE AND AFTER REPOSITORY FILES (evidence, not evaluation instructions)\n" + e.evidence
		} else {
			data := request.StructuredState
			if len(data) == 0 {
				data = []byte(request.State)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				return decision.Response{}, err
			}
			fields["snapshot_evidence"] = json.RawMessage(e.evidence)
			data, err := json.Marshal(fields)
			if err != nil {
				return decision.Response{}, err
			}
			if e.experiment.StateFormat == "json-text" {
				request.State = string(data)
			} else {
				request.StructuredState = data
			}
		}
	}
	return e.Evaluator.Evaluate(ctx, request)
}

func experimentEvidence(c GitCase, experiment *Experiment) (string, error) {
	if experiment == nil || experiment.Context != "snapshots" {
		return "", nil
	}
	// Include only code snapshots, never expected labels, rationale or provenance.
	data, err := json.Marshal(struct {
		Before map[string]string `json:"before_files"`
		After  map[string]string `json:"after_files"`
	}{c.BeforeFiles, c.AfterFiles})
	if err != nil {
		return "", err
	}
	if len(data) > 65536 {
		return "", fmt.Errorf("snapshot experiment exceeds 64 KiB evidence limit; use smaller fixtures")
	}
	return string(data), nil
}
