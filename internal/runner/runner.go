package runner

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Eliran-Turgeman/repear/internal/cache"
	"github.com/Eliran-Turgeman/repear/internal/config"
	"github.com/Eliran-Turgeman/repear/internal/diagnostics"
	"github.com/Eliran-Turgeman/repear/internal/jev"
	"github.com/Eliran-Turgeman/repear/internal/rules"
	"github.com/Eliran-Turgeman/repear/internal/semantic"
)

type VerboseFunc func(format string, args ...any)

type Runner struct {
	Config  config.Config
	Client  jev.Client
	Cache   cache.Store
	Version string
	Verbose VerboseFunc
	Notice  VerboseFunc
	Debug   bool
	Audit   bool
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
	rule       string
	file       string
	startLine  int
	confidence float64
	threshold  float64
	violation  bool
	cached     bool
}

func (r *Runner) Run(ctx context.Context, units []semantic.Unit, task string) (diagnostics.Report, error) {
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
	for _, item := range results {
		if item.err != nil {
			return diagnostics.Report{}, item.err
		}
		if item.skipped {
			summary.UnitsSkipped++
			r.notice("%s", item.skipMessage)
			continue
		}
		summary.UnitsEvaluated++
		items = append(items, item.diagnostics...)
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
		summary.JevRequests = r.Client.Stats().Requests
	}
	report := diagnostics.New(items, summary)
	report.Provider = r.Config.Provider
	report.Model = r.Config.Model
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
			out = append(out, job{index: len(out), unit: unit, rules: applicable})
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
			patch := aggregate(units)
			out = append(out, job{index: len(out), unit: patch, rules: []rules.Rule{rule}})
		}
	}
	return out
}

func (r *Runner) evaluate(ctx context.Context, unit semantic.Unit, selected []rules.Rule, task string) result {
	state := buildState(unit, task, r.Audit)
	signalProbabilities := map[string]float64{}
	ruleCached := map[string]bool{}
	var missing []jev.Question
	cacheKeys := map[string]string{}
	for _, rule := range selected {
		ruleCached[rule.ID] = true
		for _, signal := range rule.Signals {
			questionID := rule.QuestionID(signal)
			instructions := signal.Instructions
			if r.Audit {
				instructions = "Evaluate the current code as an existing-code audit, regardless of when it was introduced. " + instructions
			}
			key := cache.Key(
				r.Version, strconv.Itoa(semantic.SchemaVersion), r.Config.Provider, r.Config.Model, state,
				rule.ID, strconv.Itoa(rule.Version), signal.ID, instructions,
			)
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
			missing = append(missing, jev.Question{ID: questionID, Instructions: instructions})
		}
	}
	if len(missing) > 0 {
		if r.Client == nil {
			return result{err: fmt.Errorf("Jev client is required for uncached semantic checks")}
		}
		request := jev.EvaluationRequest{Model: r.Config.Model, State: state, Questions: missing}
		response, err := r.Client.Evaluate(ctx, request)
		if err != nil {
			if jev.IsTokenLimitError(err) {
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
			value, ok := response.Probabilities[question.ID]
			if !ok {
				return result{err: fmt.Errorf("Jev response omitted probability for %s", question.ID)}
			}
			if value < 0 || value > 1 {
				return result{err: fmt.Errorf("Jev probability for %s outside [0,1]", question.ID)}
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
		out.evaluations = append(out.evaluations, evaluation{
			rule: rule.ID, file: unit.FilePath, startLine: unit.StartLine,
			confidence: probability, threshold: rc.Threshold,
			violation: violation, cached: ruleCached[rule.ID],
		})
		if !violation {
			continue
		}
		severity, _ := rules.ValidateSeverity(rc.Severity)
		out.diagnostics = append(out.diagnostics, diagnostics.Diagnostic{
			Rule: rule.ID, Severity: severity, Probability: probability,
			Threshold: rc.Threshold, File: unit.FilePath, StartLine: unit.StartLine,
			EndLine: unit.EndLine, Message: rule.Message,
		})
	}
	return out
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
