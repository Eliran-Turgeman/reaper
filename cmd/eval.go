package cmd

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/Eliran-Turgeman/reaper/internal/config"
	evalpkg "github.com/Eliran-Turgeman/reaper/internal/eval"
	"github.com/Eliran-Turgeman/reaper/internal/provider"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/spf13/cobra"
)

type evalOptions struct {
	optimize          string
	minRecall         float64
	metricsBaseline   string
	tolerance         float64
	benchmarkDir      string
	benchmarkMode     string
	benchmarkGrouping string
	experimentFile    string
	qualityPolicyFile string
	rule              string
	threshold         float64
	thresholdSet      bool
	format            string
	dir               string
	provider          string
	model             string
}

func newEval(app App) *cobra.Command {
	options := evalOptions{}
	command := &cobra.Command{
		Use:   "eval",
		Short: "Run the labeled semantic-rule benchmark",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			var qualityPolicy *evalpkg.QualityPolicy
			if options.qualityPolicyFile != "" {
				if options.benchmarkDir == "" || options.benchmarkMode != "git" || options.benchmarkGrouping != "configured" || options.experimentFile != "" {
					return fmt.Errorf("--quality-policy requires configured Git benchmarking without experiment overrides")
				}
				policy, err := evalpkg.LoadQualityPolicy(options.qualityPolicyFile)
				if err != nil {
					return err
				}
				if err := policy.Ready(); err != nil {
					fmt.Fprintln(command.ErrOrStderr(), "reaper:", err)
					return &ExitError{Code: 1, Err: err}
				}
				qualityPolicy = &policy
			}
			var experiment *evalpkg.Experiment
			if options.experimentFile != "" {
				if options.benchmarkDir != "" && options.benchmarkMode != "git" {
					return fmt.Errorf("patch experiments require --benchmark-mode git")
				}
				var err error
				experiment, err = evalpkg.LoadExperiment(options.experimentFile)
				if err != nil {
					return err
				}
				if options.benchmarkDir == "" && experiment.Context != "current" {
					return fmt.Errorf("example experiments require context current")
				}
			}
			if options.benchmarkMode != "snippet" && options.benchmarkMode != "git" {
				return fmt.Errorf("--benchmark-mode must be snippet or git")
			}
			if options.benchmarkGrouping != "configured" && options.benchmarkGrouping != "isolated" {
				return fmt.Errorf("--benchmark-grouping must be configured or isolated")
			}
			if (command.Flags().Changed("benchmark-mode") || command.Flags().Changed("benchmark-grouping")) && options.benchmarkDir == "" {
				return fmt.Errorf("benchmark mode/grouping require --benchmark-dir")
			}
			if command.Flags().Changed("benchmark-grouping") && options.benchmarkMode != "git" {
				return fmt.Errorf("--benchmark-grouping requires --benchmark-mode git")
			}
			if options.optimize != "" && options.optimize != "precision" {
				return fmt.Errorf("--optimize must be precision")
			}
			if math.IsNaN(options.minRecall) || options.minRecall < 0 || options.minRecall > 1 {
				return fmt.Errorf("--min-recall must be between 0 and 1")
			}
			if math.IsNaN(options.tolerance) || options.tolerance < 0 || options.tolerance > 1 {
				return fmt.Errorf("--tolerance must be between 0 and 1")
			}
			if options.format != "text" && options.format != "json" {
				return fmt.Errorf("--format must be text or json")
			}
			if options.thresholdSet && (math.IsNaN(options.threshold) || options.threshold < 0 || options.threshold > 1) {
				return fmt.Errorf("--threshold must be between 0 and 1")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			cfg, _, err := config.Load(cwd)
			if err != nil {
				return err
			}
			if err := applyProviderOverrides(&cfg, options.provider, options.model); err != nil {
				return err
			}
			if qualityPolicy != nil {
				if err := qualityPolicy.CheckConfig(cfg); err != nil {
					return err
				}
			}
			client, err := provider.New(cfg.Provider, provider.Options{
				BaseURL: cfg.BaseURL, APIKey: app.Getenv(config.APIKeyEnv(cfg.Provider)),
				Timeout: cfg.RequestTimeout, MaxRetries: 2,
			})
			if err != nil {
				return err
			}
			thresholds := map[string]float64{}
			for _, rule := range rules.All() {
				thresholds[rule.ID] = cfg.Rules[rule.ID].Threshold
			}
			var override *float64
			if options.thresholdSet {
				override = &options.threshold
			}
			dir := options.dir
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(cwd, dir)
			}
			var report evalpkg.Report
			if options.benchmarkDir != "" {
				if options.rule != "" || options.thresholdSet {
					return fmt.Errorf("benchmark uses its labeled rules and configured thresholds")
				}
				if options.benchmarkMode == "git" {
					report, err = evalpkg.RunGitExperiment(command.Context(), client, options.benchmarkDir, cfg, options.benchmarkGrouping, experiment)
				} else {
					report, err = evalpkg.RunBenchmark(command.Context(), client, options.benchmarkDir, cfg)
				}
			} else {
				report, err = evalpkg.RunExamplesExperiment(command.Context(), client, dir, cfg.Provider, cfg.Model, options.rule, override, thresholds, experiment)
			}
			if err != nil {
				return err
			}
			if options.optimize != "" {
				report.Calibration, err = evalpkg.Calibrate(report, options.minRecall)
				if err != nil {
					return err
				}
			}
			if options.format == "json" {
				err = evalpkg.WriteJSON(command.OutOrStdout(), report)
			} else {
				err = evalpkg.WriteText(command.OutOrStdout(), report)
			}
			if err != nil {
				return err
			}
			if options.metricsBaseline != "" {
				if err := evalpkg.CheckBaseline(command.ErrOrStderr(), report, options.metricsBaseline, options.tolerance); err != nil {
					return &ExitError{Code: 1, Err: err}
				}
			}
			if qualityPolicy != nil {
				if err := evalpkg.CheckQualityPolicy(command.ErrOrStderr(), report, *qualityPolicy); err != nil {
					fmt.Fprintln(command.ErrOrStderr(), "reaper:", err)
					return &ExitError{Code: 1, Err: err}
				}
			}
			return nil
		},
	}
	flags := command.Flags()
	flags.StringVar(&options.optimize, "optimize", "", "report threshold candidates optimizing precision")
	flags.Float64Var(&options.minRecall, "min-recall", .6, "minimum recall for threshold candidates")
	flags.StringVar(&options.metricsBaseline, "metrics-baseline", "", "expected evaluation metrics JSON for a quality gate")
	flags.Float64Var(&options.tolerance, "tolerance", .02, "maximum precision/recall drop allowed by the metrics baseline")
	flags.StringVar(&options.benchmarkDir, "benchmark-dir", "", "evaluate labeled coding-agent patches from cases.json")
	flags.StringVar(&options.benchmarkMode, "benchmark-mode", "snippet", "benchmark input path: snippet or git")
	flags.StringVar(&options.benchmarkGrouping, "benchmark-grouping", "configured", "Git benchmark rule grouping: configured or isolated")
	flags.StringVar(&options.experimentFile, "benchmark-experiment", "", "JSON question/context experiment for examples or Git fixtures; does not change check defaults")
	flags.StringVar(&options.qualityPolicyFile, "quality-policy", "", "strict quality policy with independent corpus review")
	flags.StringVar(&options.rule, "rule", "", "evaluate one rule")
	flags.Float64Var(&options.threshold, "threshold", 0, "override the configured threshold")
	flags.StringVar(&options.format, "format", "text", "output format: text or json")
	flags.StringVar(&options.dir, "eval-dir", "evals", "evaluation corpus directory")
	flags.StringVar(&options.provider, "provider", "", "evaluation provider: typesafe or openrouter")
	flags.StringVar(&options.model, "model", "", "provider model override")
	command.PreRun = func(command *cobra.Command, _ []string) {
		options.thresholdSet = command.Flags().Changed("threshold")
	}
	return command
}
