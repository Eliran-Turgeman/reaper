package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

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
	if e.StateFormat != "" && e.StateFormat != "json-text" && e.StateFormat != "json-object" {
		return fmt.Errorf("unsupported experiment state_format %q", e.StateFormat)
	}
	for _, q := range rules.Questions(rules.All(), false, false) {
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
