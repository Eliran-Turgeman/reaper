package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Eliran-Turgeman/reaper/internal/cache"
	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
)

type VerboseFunc func(format string, args ...any)

type Runner struct {
	// StateFormat is an experimental representation override; empty preserves legacy text.
	StateFormat string
	Observe     func(Observation)
	GitEnv      []string
	Root        string
	Config      config.Config
	Client      decision.Evaluator
	Cache       cache.Store
	Version     string
	Verbose     VerboseFunc
	Notice      VerboseFunc
	Debug       bool
	Audit       bool
}

type job struct {
	index int
	unit  semantic.Unit
	rules []rules.Rule
}

type result struct {
	diagnostics []diagnostics.Diagnostic
	evaluations []evaluation
	checks      int
	cacheHits   int
	skipped     bool
	skipMessage string
	err         error
}

type evaluation struct {
	observation     Observation
	contextEvidence string
	rule            string
	file            string
	startLine       int
	confidence      float64
	threshold       float64
	violation       bool
	cached          bool
}

func (r *Runner) Run(ctx context.Context, units []semantic.Unit, task string) (diagnostics.Report, error) {
	started := time.Now()
	before := decision.Stats{}
	if r.Client != nil {
		before = r.Client.Stats()
	}
	jobs := r.jobs(units, task, true)
	results := make([]result, len(jobs))
	limit := make(chan struct{}, r.Config.Concurrency)
	var wg sync.WaitGroup
	for _, work := range jobs {
		wg.Add(1)
		go func(work job) {
			defer wg.Done()
			select {
			case limit <- struct{}{}:
				defer func() { <-limit }()
			case <-ctx.Done():
				results[work.index].err = ctx.Err()
				return
			}
			results[work.index] = r.evaluate(ctx, work.unit, work.rules, task)
		}(work)
	}
	wg.Wait()

	summary := diagnostics.Summary{}
	var items []diagnostics.Diagnostic
	var evaluations []evaluation
	var skipped []diagnostics.SkippedUnit
	for i, item := range results {
		if item.err != nil || item.skipped {
			unit := jobs[i].unit
			reason := item.skipMessage
			if item.err != nil {
				reason = "evaluation failed: " + item.err.Error()
			}
			ids := make([]string, 0, len(jobs[i].rules))
			for _, rule := range jobs[i].rules {
				ids = append(ids, rule.ID)
			}
			skipped = append(skipped, diagnostics.SkippedUnit{File: unit.FilePath, StartLine: unit.StartLine, EndLine: unit.EndLine, Rules: ids, Reason: reason})
			summary.UnitsSkipped++
			r.notice("%s", reason)
			continue
		}
		summary.UnitsEvaluated++
		items = append(items, item.diagnostics...)
		if r.Observe != nil {
			for _, evaluated := range item.evaluations {
				r.Observe(evaluated.observation)
			}
		}
		evaluations = append(evaluations, item.evaluations...)
		summary.SemanticChecks += item.checks
		summary.CacheHits += item.cacheHits
	}
	if r.Debug {
		sort.Slice(evaluations, func(i, j int) bool {
			if evaluations[i].file != evaluations[j].file {
				return evaluations[i].file < evaluations[j].file
			}
			if evaluations[i].startLine != evaluations[j].startLine {
				return evaluations[i].startLine < evaluations[j].startLine
			}
			return evaluations[i].rule < evaluations[j].rule
		})
		for _, item := range evaluations {
			if item.contextEvidence != "" {
				r.log("context %s for %s:%d\n%s", item.rule, item.file, item.startLine, item.contextEvidence)
			}
			outcome := "pass"
			if item.violation {
				outcome = "violation"
			}
			source := "provider"
			if item.cached {
				source = "cache"
			}
			r.log("evaluation %s for %s:%d confidence=%s threshold=%s result=%s source=%s",
				item.rule, item.file, item.startLine,
				strconv.FormatFloat(item.confidence, 'g', -1, 64),
				strconv.FormatFloat(item.threshold, 'g', -1, 64),
				outcome, source)
		}
	}
	if r.Client != nil {
		after := r.Client.Stats()
		summary.Requests = after.Requests - before.Requests
		summary.JevRequests = summary.Requests
		summary.Retries = after.Retries - before.Retries
		summary.InputTokens = after.InputTokens - before.InputTokens
		summary.OutputTokens = after.OutputTokens - before.OutputTokens
		summary.UsageResponses = after.UsageResponses - before.UsageResponses
	}
	summary.UsageComplete = summary.UsageResponses == summary.Requests
	if summary.SemanticChecks > 0 {
		summary.CacheHitRate = float64(summary.CacheHits) / float64(summary.SemanticChecks)
	}
	summary.DurationMS = float64(time.Since(started).Microseconds()) / 1000
	if summary.Requests == 0 {
		zero := 0.0
		summary.EstimatedCostUSD = &zero
	} else if summary.UsageResponses > 0 && r.Config.Pricing.InputPerMillion != nil && r.Config.Pricing.OutputPerMillion != nil {
		cost := (float64(summary.InputTokens)**r.Config.Pricing.InputPerMillion + float64(summary.OutputTokens)**r.Config.Pricing.OutputPerMillion) / 1e6
		summary.EstimatedCostUSD = &cost
	}
	report := diagnostics.New(items, summary)
	if r.Client != nil {
		capabilities := r.Client.Capabilities()
		report.Capabilities = &capabilities
	}
	report.Provider = r.Config.Provider
	report.Model = r.Config.Model
	report.SetIncomplete(skipped, r.Config.IncompleteAnalysis)
	return report, nil
}

func (r *Runner) WorkCount(units []semantic.Unit, task string) int {
	return len(r.jobs(units, task, false))
}

func (r *Runner) jobs(units []semantic.Unit, task string, logSkips bool) []job {
	var out []job
	for _, unit := range units {
		var applicable []rules.Rule
		for _, rule := range rules.All() {
			if rule.Scope == rules.ScopePatch || !r.enabled(rule) || !r.matches(rule, unit.FilePath) {
				continue
			}
			if r.Audit && rule.AuditSkipReason != "" {
				if logSkips {
					r.log("skip %s for %s:%d: %s", rule.ID, unit.FilePath, unit.StartLine, rule.AuditSkipReason)
				}
				continue
			}
			ok, reason := rule.Applicable(unit, task)
			if !ok {
				if logSkips {
					r.log("skip %s for %s:%d: %s", rule.ID, unit.FilePath, unit.StartLine, reason)
				}
				continue
			}
			applicable = append(applicable, rule)
		}
		if len(applicable) > 0 {
			var local []rules.Rule
			for _, rule := range applicable {
				if rule.Context == "repository-search" && r.Root != "" {
					out = append(out, job{index: len(out), unit: unit, rules: []rules.Rule{rule}})
				} else {
					local = append(local, rule)
				}
			}
			if len(local) > 0 {
				out = append(out, job{index: len(out), unit: unit, rules: local})
			}
		}
	}
	if len(units) > 0 {
		for _, rule := range rules.All() {
			if rule.Scope != rules.ScopePatch || !r.enabled(rule) {
				continue
			}
			if r.Audit && rule.AuditSkipReason != "" {
				if logSkips {
					r.log("skip %s: %s", rule.ID, rule.AuditSkipReason)
				}
				continue
			}
			ok, reason := rule.Applicable(units[0], task)
			if !ok {
				if logSkips {
					r.log("skip %s: %s", rule.ID, reason)
				}
				continue
			}
			var included []semantic.Unit
			for _, unit := range units {
				if r.matches(rule, unit.FilePath) {
					included = append(included, unit)
				}
			}
			if len(included) == 0 {
				continue
			}
			patch := aggregate(included)
			out = append(out, job{index: len(out), unit: patch, rules: []rules.Rule{rule}})
		}
	}
	return out
}

func (r *Runner) evaluate(ctx context.Context, unit semantic.Unit, selected []rules.Rule, task string) result {
	state := buildState(unit, task, r.Audit)
	retrieved := ""
	if len(selected) == 1 {
		var err error
		retrieved, err = r.repositoryContext(ctx, unit, selected[0])
		if err != nil {
			return result{err: err}
		}
		if retrieved != "" {
			state += "\n\nREPOSITORY SEARCH EVIDENCE\n" + retrieved
		}
	}
	request := decision.Request{Model: r.Config.Model, State: state}
	if r.StateFormat != "" {
		data, err := json.Marshal(stateFields(unit, task, r.Audit, retrieved))
		if err != nil {
			return result{err: err}
		}
		switch r.StateFormat {
		case "json-text":
			request.State = string(data)
		case "json-object":
			request.State = ""
			request.StructuredState = data
		default:
			return result{err: fmt.Errorf("unsupported state format %q", r.StateFormat)}
		}
		// Separate text and object states even when their serialized facts match.
		state = r.StateFormat + "\n" + string(data)
	}
	signalProbabilities := map[string]float64{}
	ruleCached := map[string]bool{}
	var missing []decision.Question
	cacheKeys := map[string]string{}
	questions := map[string]decision.Question{}
	for _, question := range rules.Questions(selected, r.Audit, retrieved != "") {
		questions[question.ID] = question
	}
	for _, rule := range selected {
		ruleCached[rule.ID] = true
		for _, signal := range rule.Signals {
			questionID := rule.QuestionID(signal)
			instructions := questions[questionID].Instructions
			key := cache.Key(
				r.Version, strconv.Itoa(semantic.SchemaVersion), r.Config.Provider, r.Config.Model, state,
				rule.ID, strconv.Itoa(rule.Version), signal.ID, instructions,
			)
			if criteria := questions[questionID].Criteria; criteria != nil {
				if err := criteria.Validate(); err != nil {
					return result{err: err}
				}
				key = cache.Key(key, "noul", criteria.True, criteria.False)
			}
			cacheKeys[questionID] = key
			value, ok, err := r.Cache.Get(key)
			if err != nil {
				return result{err: err}
			}
			if ok {
				signalProbabilities[questionID] = value
				continue
			}
			ruleCached[rule.ID] = false
			missing = append(missing, questions[questionID])
		}
	}
	if len(missing) > 0 {
		if r.Client == nil {
			return result{err: fmt.Errorf("an evaluator is required for uncached semantic checks")}
		}
		request.Questions = missing
		response, err := decision.Evaluate(ctx, r.Client, request)
		if err != nil {
			if decision.IsContextLimit(err) {
				return result{
					skipped: true,
					skipMessage: fmt.Sprintf(
						"skip %s:%d: provider token limit exceeded",
						unit.FilePath, unit.StartLine,
					),
				}
			}
			return result{err: fmt.Errorf("evaluate %s:%d: %w", unit.FilePath, unit.StartLine, err)}
		}
		for _, question := range missing {
			value, ok := response.Scores[question.ID]
			if !ok {
				return result{err: fmt.Errorf("evaluator response omitted score for %s", question.ID)}
			}
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
				return result{err: fmt.Errorf("evaluator score for %s outside [0,1]", question.ID)}
			}
			signalProbabilities[question.ID] = value
			if err := r.Cache.Put(cacheKeys[question.ID], value); err != nil {
				return result{err: err}
			}
		}
	}
	out := result{checks: len(selected)}
	for _, rule := range selected {
		if ruleCached[rule.ID] {
			out.cacheHits++
		}
		probability, ok := rule.Compose(signalProbabilities)
		if !ok {
			return result{err: fmt.Errorf("cannot compose probability for %s", rule.ID)}
		}
		rc := r.Config.Rules[rule.ID]
		violation := probability >= rc.Threshold
		signals := make([]diagnostics.SignalScore, 0, len(rule.Signals))
		for _, signal := range rule.Signals {
			signals = append(signals, diagnostics.SignalScore{ID: signal.ID, Score: signalProbabilities[rule.QuestionID(signal)], Evidence: signal.Instructions})
		}
		sort.Slice(signals, func(i, j int) bool { return signals[i].ID < signals[j].ID })
		out.evaluations = append(out.evaluations, evaluation{
			observation:     Observation{Rule: rule.ID, File: unit.FilePath, StartLine: unit.StartLine, EndLine: unit.EndLine, Score: probability, Signals: signals},
			contextEvidence: retrieved,
			rule:            rule.ID, file: unit.FilePath, startLine: unit.StartLine,
			confidence: probability, threshold: rc.Threshold,
			violation: violation, cached: ruleCached[rule.ID],
		})
		if !violation {
			continue
		}
		severity, _ := rules.ValidateSeverity(rc.Severity)
		out.diagnostics = append(out.diagnostics, diagnostics.Diagnostic{
			ContextEvidence: retrieved,
			Fingerprint:     findingFingerprint(rule.ID, unit),
			Rule:            rule.ID, Severity: severity, Confidence: probability, Signals: signals,
			Threshold: rc.Threshold, File: unit.FilePath, StartLine: unit.StartLine,
			EndLine: unit.EndLine, Message: rule.Message,
		})
	}
	return out
}

func findingFingerprint(rule string, unit semantic.Unit) string {
	var changed []string
	for _, line := range strings.Split(unit.Diff, "\n") {
		if (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-")) && !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
			changed = append(changed, line[:1]+strings.Join(strings.Fields(line[1:]), " "))
		}
	}
	if len(changed) == 0 {
		changed = strings.Fields(unit.NewContent)
	}
	return cache.Key(rule, unit.FilePath, strings.Join(changed, "\n"))
}

func (r *Runner) enabled(rule rules.Rule) bool {
	rc, ok := r.Config.Rules[rule.ID]
	return ok && rc.Enabled != nil && *rc.Enabled
}

func (r *Runner) matches(rule rules.Rule, file string) bool {
	rc := r.Config.Rules[rule.ID]
	for _, glob := range append(r.Config.Exclude, rc.Exclude...) {
		if excludeMatch(glob, file) {
			return false
		}
	}
	if len(rc.Include) == 0 {
		return true
	}
	for _, glob := range rc.Include {
		if globMatch(glob, file) {
			return true
		}
	}
	return false
}

func (r *Runner) log(format string, args ...any) {
	if r.Verbose != nil {
		r.Verbose(format, args...)
	}
}

func (r *Runner) notice(format string, args ...any) {
	if r.Notice != nil {
		r.Notice(format, args...)
	}
}

func buildState(unit semantic.Unit, task string, audit bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "FILE\n%s\n\nLANGUAGE\n%s\n", unit.FilePath, unit.Language)
	if audit {
		b.WriteString("\nMODE\nEXISTING CODE AUDIT\n")
	}
	if task != "" {
		fmt.Fprintf(&b, "\nTASK\n%s\n", task)
	}
	if !audit {
		fmt.Fprintf(&b, "\nDIFF\n%s", unit.Diff)
	}
	fmt.Fprintf(&b, "\n\nCURRENT CODE\n%s", unit.NewContent)
	if unit.OldContent != "" {
		fmt.Fprintf(&b, "\n\nPREVIOUS CODE\n%s", unit.OldContent)
	}
	if unit.SurroundingCode != "" && unit.SurroundingCode != unit.NewContent {
		fmt.Fprintf(&b, "\n\nSURROUNDING CONTEXT\n%s", unit.SurroundingCode)
	}
	return b.String()
}

func aggregate(units []semantic.Unit) semantic.Unit {
	sorted := append([]semantic.Unit(nil), units...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].FilePath != sorted[j].FilePath {
			return sorted[i].FilePath < sorted[j].FilePath
		}
		return sorted[i].StartLine < sorted[j].StartLine
	})
	var diffs []string
	for _, unit := range sorted {
		diffs = append(diffs, "FILE "+unit.FilePath+"\n"+unit.Diff)
	}
	return semantic.Unit{
		FilePath: "<patch>", Language: "diff", Diff: strings.Join(diffs, "\n\n"),
		StartLine: 1, EndLine: 1,
	}
}

func excludeMatch(pattern, file string) bool {
	if globMatch(pattern, file) {
		return true
	}
	if strings.HasPrefix(strings.ReplaceAll(pattern, "\\", "/"), "**/") {
		return false
	}
	return globMatch("**/"+pattern, file)
}

func globMatch(pattern, file string) bool {
	pattern = path.Clean(strings.ReplaceAll(pattern, "\\", "/"))
	file = path.Clean(strings.ReplaceAll(file, "\\", "/"))
	var expression strings.Builder
	expression.WriteByte('^')
	for i := 0; i < len(pattern); {
		if i+2 < len(pattern) && pattern[i:i+3] == "**/" {
			expression.WriteString("(?:.*/)?")
			i += 3
			continue
		}
		if i+1 < len(pattern) && pattern[i:i+2] == "**" {
			expression.WriteString(".*")
			i += 2
			continue
		}
		switch pattern[i] {
		case '*':
			expression.WriteString("[^/]*")
		case '?':
			expression.WriteString("[^/]")
		default:
			expression.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		}
		i++
	}
	expression.WriteByte('$')
	matched, err := regexp.MatchString(expression.String(), file)
	return err == nil && matched
}
