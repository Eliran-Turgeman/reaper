package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Eliran-Turgeman/reaper/internal/baseline"
	"github.com/Eliran-Turgeman/reaper/internal/cache"
	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	diffpkg "github.com/Eliran-Turgeman/reaper/internal/diff"
	gitpkg "github.com/Eliran-Turgeman/reaper/internal/git"
	"github.com/Eliran-Turgeman/reaper/internal/provider"
	"github.com/Eliran-Turgeman/reaper/internal/runner"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
	"github.com/spf13/cobra"
)

type checkOptions struct {
	baseline      string
	taskFile      string
	taskFromPR    bool
	config        string
	failOnWarning bool
	all           bool
	staged        bool
	ref           string
	task          string
	format        string
	verbose       bool
	debug         bool
	noCache       bool
	provider      string
	model         string
}

func newCheck(app App) *cobra.Command {
	options := checkOptions{}
	command := &cobra.Command{
		Use:   "check [paths...]",
		Short: "Check changed code for semantic violations",
		Args:  cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, paths []string) error {
			if options.all && (options.staged || options.ref != "") {
				return errors.New("--all cannot be used with --staged or --diff")
			}
			if options.staged && options.ref != "" {
				return errors.New("--staged and --diff cannot be used together")
			}
			if options.format != "text" && options.format != "json" && options.format != "sarif" && options.format != "agent" {
				return fmt.Errorf("--format must be text, json, sarif, or agent")
			}
			return runCheck(command.Context(), command, app, options, paths)
		},
	}
	flags := command.Flags()
	flags.StringVar(&options.baseline, "baseline", "", "baseline of accepted existing findings")
	flags.StringVar(&options.taskFile, "task-file", "", "read task context from a UTF-8 file")
	flags.BoolVar(&options.taskFromPR, "task-from-pr", false, "derive task from GITHUB_EVENT_PATH pull request title and body")
	flags.StringVar(&options.config, "config", "", "explicit configuration file")
	flags.BoolVar(&options.failOnWarning, "fail-on-warning", false, "exit 1 when warnings are found")
	flags.BoolVar(&options.all, "all", false, "evaluate all tracked files instead of only changes")
	flags.BoolVar(&options.staged, "staged", false, "inspect the staged diff")
	flags.StringVar(&options.ref, "diff", "", "compare current code against a Git reference")
	flags.StringVar(&options.task, "task", "", "task that motivated the code change")
	flags.StringVar(&options.format, "format", "text", "output format: text, json, sarif, or agent")
	flags.BoolVarP(&options.verbose, "verbose", "v", false, "show evaluation details")
	flags.BoolVar(&options.debug, "debug", false, "show raw confidence for every evaluation")
	flags.BoolVar(&options.noCache, "no-cache", false, "disable result caching")
	flags.StringVar(&options.provider, "provider", "", "evaluation provider: typesafe or openrouter")
	flags.StringVar(&options.model, "model", "", "provider model override")
	return command
}

func runCheck(ctx context.Context, command *cobra.Command, app App, options checkOptions, paths []string) error {
	started := time.Now()
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	cfg, configPath, err := config.Load(cwd)
	if options.config != "" {
		cfg, configPath, err = config.LoadFile(options.config)
	}
	if err != nil {
		return err
	}
	if err := applyProviderOverrides(&cfg, options.provider, options.model); err != nil {
		return err
	}
	task, taskSource, err := resolveTask(options, command.Flags().Changed("task"), app.Getenv)
	if err != nil {
		return err
	}
	collector := gitpkg.CommandCollector{Dir: cwd}
	rawDiff, root, err := collector.Diff(ctx, gitpkg.Options{
		All: options.all, Staged: options.staged, Ref: options.ref, Paths: paths,
	})
	if err != nil {
		return err
	}
	var units []semantic.Unit
	if options.staged {
		index := gitpkg.CommandCollector{Dir: root}
		units, err = diffpkg.UnitsWithSource(rawDiff, 6, func(path string) ([]byte, error) {
			return index.IndexSource(ctx, path)
		})
	} else {
		units, err = diffpkg.Units(root, rawDiff, 6)
	}
	if err != nil {
		return fmt.Errorf("extract semantic units: %w", err)
	}
	log := func(format string, args ...any) {}
	if options.verbose || options.debug {
		log = func(format string, args ...any) {
			fmt.Fprintf(command.ErrOrStderr(), "reaper: "+format+"\n", args...)
		}
		mode := "changes"
		if options.all {
			mode = "all"
		}
		log("config=%s provider=%s model=%s mode=%s units=%d",
			displayConfig(configPath), cfg.Provider, cfg.Model, mode, len(units))
		log("task_source=%s", taskSource)
	}

	cacheStore := cache.Store(cache.Disabled{})
	if !options.noCache && cfg.Cache.Enabled != nil && *cfg.Cache.Enabled {
		dir := cfg.Cache.Dir
		if dir == "" {
			dir = filepath.Join(root, ".git", "reaper-cache")
		} else if !filepath.IsAbs(dir) {
			dir = filepath.Join(root, dir)
		}
		cacheStore = &cache.FileStore{Dir: dir}
	}
	engine := &runner.Runner{
		Root:   root,
		Config: cfg, Cache: cacheStore, Version: Version, Verbose: log,
		Debug: options.debug, Audit: options.all,
		Notice: func(format string, args ...any) {
			fmt.Fprintf(command.ErrOrStderr(), "reaper: "+format+"\n", args...)
		},
	}
	if engine.WorkCount(units, task) > 0 {
		client, err := provider.New(cfg.Provider, provider.Options{
			BaseURL: cfg.BaseURL, APIKey: app.Getenv(config.APIKeyEnv(cfg.Provider)),
			Timeout: cfg.RequestTimeout, MaxRetries: 2,
		})
		if err != nil {
			return err
		}
		engine.Client = client
	}
	report, err := engine.Run(ctx, units, task)
	if err != nil {
		return err
	}
	report, err = baseline.Apply(root, options.baseline, report)
	if err != nil {
		return err
	}
	if report.Complete && options.failOnWarning && report.Summary.Warnings > 0 {
		report.Passed = false
		report.Status = "failed"
	}
	if options.verbose || options.debug {
		log("checks=%d requests=%d cache_hits=%d duration=%s",
			report.Summary.SemanticChecks, report.Summary.JevRequests,
			report.Summary.CacheHits, time.Since(started).Round(time.Millisecond))
		cost := "unknown"
		if report.Summary.EstimatedCostUSD != nil {
			cost = fmt.Sprintf("%.6f", *report.Summary.EstimatedCostUSD)
		}
		log("input_tokens=%d output_tokens=%d usage_complete=%t cache_hit_rate=%.3f estimated_cost_usd=%s", report.Summary.InputTokens, report.Summary.OutputTokens, report.Summary.UsageComplete, report.Summary.CacheHitRate, cost)
	}
	if options.format == "agent" {
		err = diagnostics.WriteAgent(command.OutOrStdout(), report, task)
	} else if options.format == "sarif" {
		err = diagnostics.WriteSARIF(command.OutOrStdout(), report)
	} else if options.format == "json" {
		err = diagnostics.WriteJSON(command.OutOrStdout(), report)
	} else {
		err = diagnostics.WriteText(command.OutOrStdout(), report)
	}
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if code := report.ExitCode(); code != 0 {
		message := "blocking semantic violations found"
		if code == 2 {
			message = "analysis incomplete"
		}
		return &ExitError{Code: code, Err: errors.New(message)}
	}
	if options.failOnWarning && report.Summary.Warnings > 0 {
		return &ExitError{Code: 1, Err: errors.New("semantic warnings found")}
	}
	return nil
}

func applyProviderOverrides(cfg *config.Config, provider, model string) error {
	if provider != "" {
		cfg.Provider = provider
		if model == "" {
			cfg.Model = config.DefaultModel(provider)
		}
	}
	if model != "" {
		cfg.Model = model
	}
	return cfg.Validate()
}

func displayConfig(path string) string {
	if path == "" {
		return "<defaults>"
	}
	return path
}
