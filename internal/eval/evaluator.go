package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/Eliran-Turgeman/reaper/internal/runner"
	"gopkg.in/yaml.v3"
)

type Example struct {
	ID        string `yaml:"id"`
	Language  string `yaml:"language"`
	Task      string `yaml:"task,omitempty"`
	OldCode   string `yaml:"old_code,omitempty"`
	Code      string `yaml:"code"`
	Expected  string `yaml:"expected"`
	Rationale string `yaml:"rationale"`
}

type RuleReport struct {
	Rule                 string  `json:"rule"`
	Examples             int     `json:"examples"`
	TruePositive         int     `json:"true_positive"`
	FalsePositive        int     `json:"false_positive"`
	TrueNegative         int     `json:"true_negative"`
	FalseNegative        int     `json:"false_negative"`
	AbstainedPositive    int     `json:"abstained_positive,omitempty"`
	AbstainedNegative    int     `json:"abstained_negative,omitempty"`
	PermissionExamples   int     `json:"permission_examples,omitempty"`
	PermissionAllowed    int     `json:"permission_allowed,omitempty"`
	PermissionDisallowed int     `json:"permission_disallowed,omitempty"`
	PermissionUncertain  int     `json:"permission_uncertain,omitempty"`
	PermissionMismatch   int     `json:"permission_mismatch,omitempty"`
	Precision            float64 `json:"precision"`
	Recall               float64 `json:"recall"`
	FalsePositiveRate    float64 `json:"false_positive_rate"`
	Threshold            float64 `json:"threshold"`
	AveragePositiveScore float64 `json:"average_positive_score"`
	AverageNegativeScore float64 `json:"average_negative_score"`
}

type Report struct {
	Provenance  *Provenance   `json:"provenance,omitempty"`
	Mode        string        `json:"mode,omitempty"`
	Grouping    string        `json:"grouping,omitempty"`
	Calibration []Calibration `json:"calibration,omitempty"`
	Cases       []ScoredCase  `json:"cases,omitempty"`
	Version     int           `json:"version"`
	Provider    string        `json:"provider"`
	Model       string        `json:"model"`
	Rules       []RuleReport  `json:"rules"`
}

func Load(dir, id string) ([]Example, error) {
	data, err := os.ReadFile(filepath.Join(dir, id+".yaml"))
	if err != nil {
		return nil, fmt.Errorf("read eval corpus for %s: %w", id, err)
	}
	var examples []Example
	if err := yaml.Unmarshal(data, &examples); err != nil {
		return nil, fmt.Errorf("parse eval corpus for %s: %w", id, err)
	}
	for i, example := range examples {
		if example.ID == "" || example.Language == "" || example.Code == "" || example.Rationale == "" {
			return nil, fmt.Errorf("%s example %d is missing a required field", id, i+1)
		}
		if example.Expected != "positive" && example.Expected != "negative" {
			return nil, fmt.Errorf("%s example %s has invalid expected label %q", id, example.ID, example.Expected)
		}
	}
	return examples, nil
}

func Run(ctx context.Context, client decision.Evaluator, dir, provider, model, onlyRule string, thresholdOverride *float64, thresholds map[string]float64) (Report, error) {
	return RunExamplesExperiment(ctx, client, dir, provider, model, onlyRule, thresholdOverride, thresholds, nil)
}

func RunExamplesExperiment(ctx context.Context, client decision.Evaluator, dir, provider, model, onlyRule string, thresholdOverride *float64, thresholds map[string]float64, experiment *Experiment) (Report, error) {
	if experiment != nil {
		if err := experiment.validate(); err != nil {
			return Report{}, err
		}
		if experiment.Context != "current" {
			return Report{}, fmt.Errorf("example experiments require context current; repository evidence requires Git fixtures")
		}
	}
	selected := rules.All()
	if onlyRule != "" {
		rule, ok := rules.Get(onlyRule)
		if !ok {
			return Report{}, fmt.Errorf("unknown rule %q", onlyRule)
		}
		selected = []rules.Rule{rule}
	}
	if experiment != nil {
		selected = experiment.ruleSet(selected)
	}
	report := Report{Version: 1, Provider: provider, Model: model}
	corpus := map[string][]Example{}
	cfg := config.Config{Rules: map[string]config.RuleConfig{}}
	for _, rule := range selected {
		examples, err := Load(dir, rule.ID)
		if err != nil {
			return Report{}, err
		}
		threshold := thresholds[rule.ID]
		if thresholdOverride != nil {
			threshold = *thresholdOverride
		}
		corpus[rule.ID] = examples
		cfg.Rules[rule.ID] = config.RuleConfig{Threshold: threshold}
		result := RuleReport{Rule: rule.ID, Examples: len(examples), Threshold: threshold}
		var positiveTotal, negativeTotal float64
		var positives, negatives int
		for _, example := range examples {
			state := evalState(example)
			request := decision.Request{Model: model, State: state, Questions: rules.Questions([]rules.Rule{rule}, false, false)}
			if experiment != nil && experiment.StateFormat != "" {
				fields := map[string]string{"language": example.Language, "current_code": example.Code}
				if example.Task != "" {
					fields["task"] = example.Task
				}
				if example.OldCode != "" {
					fields["previous_code"] = example.OldCode
				}
				data, err := json.Marshal(fields)
				if err != nil {
					return Report{}, err
				}
				if experiment.StateFormat == "json-text" {
					request.State = string(data)
				} else {
					request.State = ""
					request.StructuredState = data
				}
			}
			recorder := &recordingEvaluator{Evaluator: client}
			var evaluator decision.Evaluator = recorder
			if experiment != nil {
				evaluator = &experimentEvaluator{Evaluator: recorder, experiment: experiment}
			}
			requests := []decision.Request{request}
			if experiment != nil && experiment.PermissionEval == "separate-request-v1" {
				permissionIDs := map[string]bool{}
				for _, signal := range rule.Signals {
					if signal.Role == rules.SignalRolePermission {
						permissionIDs[rule.QuestionID(signal)] = true
					}
				}
				factual, permission := request, request
				factual.Questions = nil
				permission.Questions = nil
				for _, question := range request.Questions {
					if permissionIDs[question.ID] {
						permission.Questions = append(permission.Questions, question)
					} else {
						factual.Questions = append(factual.Questions, question)
					}
				}
				requests = []decision.Request{factual, permission}
			}
			response := decision.Response{Scores: map[string]float64{}}
			for _, grouped := range requests {
				if len(grouped.Questions) == 0 {
					continue
				}
				part, err := decision.Evaluate(ctx, evaluator, grouped)
				if err != nil {
					return Report{}, fmt.Errorf("evaluate example %s: %w", example.ID, err)
				}
				for id, score := range part.Scores {
					response.Scores[id] = score
				}
				response.Calls = append(response.Calls, part.Calls...)
			}
			score, ok := rule.Compose(response.Scores)
			if !ok || score < 0 || score > 1 {
				return Report{}, fmt.Errorf("invalid probability for example %s", example.ID)
			}
			var signals []diagnostics.SignalScore
			decisionOutcome := "scored"
			for _, signal := range rule.Signals {
				instructions := signal.Instructions
				if experiment != nil {
					if text, ok := experiment.Questions[rule.QuestionID(signal)]; ok {
						instructions = text
					}
				}
				signals = append(signals, diagnostics.SignalScore{ID: signal.ID, Score: response.Scores[rule.QuestionID(signal)], Evidence: instructions, Negated: signal.Negate})
			}
			var permission *runner.PermissionObservation
			if experiment != nil && experiment.Permission != nil {
				permissionScore, permissionSignals, ok := rule.ComposePermission(response.Scores)
				if !ok {
					return Report{}, fmt.Errorf("cannot compose permission for example %s", example.ID)
				}
				permission = &runner.PermissionObservation{Score: permissionScore, Outcome: experiment.Permission.Classify(permissionScore)}
				for _, signalID := range permissionSignals {
					permission.Signals = append(permission.Signals, runner.PermissionSignalObservation{
						Signal: signalID,
						Score:  response.Scores[rule.ID+":"+signalID],
					})
				}
				if len(permission.Signals) == 1 {
					permission.Signal = permission.Signals[0].Signal
					permission.Signals = nil
				}
				if permission.Outcome == "allowed" && experiment.PermissionAction == "suppress-allowed" {
					decisionOutcome = "permission-allowed"
				}
			}
			report.Cases = append(report.Cases, ScoredCase{ID: example.ID, Rule: rule.ID, Expected: example.Expected, Score: score, Split: "dev", Decision: decisionOutcome, PermissionOutcome: permissionOutcome(permission), Requests: recorder.Records(), Observations: []runner.Observation{{Rule: rule.ID, Score: score, Signals: signals, Composition: rule.Composition(), RuleVersion: rule.Version, Decision: decisionOutcome, Permission: permission}}})
			predicted := score >= threshold && decisionOutcome != "permission-allowed"
			if example.Expected == "positive" {
				positives++
				positiveTotal += score
				if predicted {
					result.TruePositive++
				} else {
					result.FalseNegative++
				}

			} else {
				negatives++
				negativeTotal += score
				if predicted {
					result.FalsePositive++
				} else {
					result.TrueNegative++
				}
			}
		}
		result.Precision = ratio(result.TruePositive, result.TruePositive+result.FalsePositive)
		result.Recall = ratio(result.TruePositive, result.TruePositive+result.FalseNegative)
		result.FalsePositiveRate = ratio(result.FalsePositive, result.FalsePositive+result.TrueNegative)
		result.AveragePositiveScore = ratioFloat(positiveTotal, positives)
		result.AverageNegativeScore = ratioFloat(negativeTotal, negatives)
		report.Rules = append(report.Rules, result)
	}
	sort.Slice(report.Rules, func(i, j int) bool { return report.Rules[i].Rule < report.Rules[j].Rule })
	report.Provenance = provenance("seed-v1", corpus, cfg)
	if experiment != nil {
		report.Provenance.ExperimentSHA256 = fingerprint(experiment)
	}
	return report, nil
}

func permissionOutcome(permission *runner.PermissionObservation) string {
	if permission == nil {
		return ""
	}
	return permission.Outcome
}

func WriteJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func WriteText(w io.Writer, report Report) error {
	for _, c := range report.Calibration {
		if c.Candidate == nil {
			fmt.Fprintf(w, "%s: no threshold meets the recall constraint\n", c.Rule)
		} else {
			fmt.Fprintf(w, "%s candidate threshold=%.2f precision=%.3f recall=%.3f (report only)\n", c.Rule, c.Candidate.Threshold, c.Candidate.Precision, c.Candidate.Recall)
		}
	}
	fmt.Fprintf(w, "Provider: %s\nModel: %s\n\n", report.Provider, report.Model)
	for i, result := range report.Rules {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "Rule: %s\n\n", result.Rule)
		fmt.Fprintf(w, "Examples:          %d\n", result.Examples)
		fmt.Fprintf(w, "True positive:     %d\n", result.TruePositive)
		fmt.Fprintf(w, "False positive:    %d\n", result.FalsePositive)
		fmt.Fprintf(w, "True negative:     %d\n", result.TrueNegative)
		fmt.Fprintf(w, "False negative:    %d\n\n", result.FalseNegative)
		if result.AbstainedPositive+result.AbstainedNegative > 0 {
			fmt.Fprintf(w, "Insufficient evidence: %d positive, %d negative\n\n", result.AbstainedPositive, result.AbstainedNegative)
		}
		if result.PermissionExamples > 0 {
			fmt.Fprintf(w, "Permission outcomes: %d allowed, %d disallowed, %d uncertain, %d mismatched\n\n", result.PermissionAllowed, result.PermissionDisallowed, result.PermissionUncertain, result.PermissionMismatch)
		}
		fmt.Fprintf(w, "Precision:         %.1f%%\n", result.Precision*100)
		fmt.Fprintf(w, "Recall:            %.1f%%\n", result.Recall*100)
		fmt.Fprintf(w, "False positive:    %.1f%%\n", result.FalsePositiveRate*100)
		fmt.Fprintf(w, "Threshold:         %.2f\n", result.Threshold)
		fmt.Fprintf(w, "Average positive:  %.3f\n", result.AveragePositiveScore)
		fmt.Fprintf(w, "Average negative:  %.3f\n", result.AverageNegativeScore)
	}
	return nil
}

func evalState(example Example) string {
	var b strings.Builder
	fmt.Fprintf(&b, "LANGUAGE\n%s\n", example.Language)
	if example.Task != "" {
		fmt.Fprintf(&b, "\nTASK\n%s\n", example.Task)
	}
	if example.OldCode != "" {
		fmt.Fprintf(&b, "\nPREVIOUS CODE\n%s\n", example.OldCode)
	}
	fmt.Fprintf(&b, "\nCURRENT CODE\n%s", example.Code)
	return b.String()
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func ratioFloat(numerator float64, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / float64(denominator)
}
