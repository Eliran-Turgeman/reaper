package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Eliran-Turgeman/repear/internal/cache"
	"github.com/Eliran-Turgeman/repear/internal/config"
	"github.com/Eliran-Turgeman/repear/internal/diagnostics"
	diffpkg "github.com/Eliran-Turgeman/repear/internal/diff"
	gitpkg "github.com/Eliran-Turgeman/repear/internal/git"
	"github.com/Eliran-Turgeman/repear/internal/jev"
	"github.com/Eliran-Turgeman/repear/internal/runner"
	"github.com/spf13/cobra"
)

type checkOptions struct {
	all      bool
	staged   bool
	ref      string
	task     string
	format   string
	verbose  bool
	debug    bool
	noCache  bool
	provider string
	model    string
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
			if options.format != "text" && options.format != "json" {
				return fmt.Errorf("--format must be text or json")
			}
			return runCheck(command.Context(), command, app, options, paths)
		},
	}
	flags := command.Flags()
	flags.BoolVar(&options.all, "all", false, "evaluate all tracked files instead of only changes")
	flags.BoolVar(&options.staged, "staged", false, "inspect the staged diff")
	flags.StringVar(&options.ref, "diff", "", "compare current code against a Git reference")
	flags.StringVar(&options.task, "task", "", "task that motivated the code change")
	flags.StringVar(&options.format, "format", "text", "output format: text or json")
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
	if err != nil {
		return err
	}
	if err := applyProviderOverrides(&cfg, options.provider, options.model); err != nil {
		return err
	}
	task := options.task
	if task == "" {
		task = app.Getenv("REAPER_TASK")
	}
	collector := gitpkg.CommandCollector{Dir: cwd}
	rawDiff, root, err := collector.Diff(ctx, gitpkg.Options{
		All: options.all, Staged: options.staged, Ref: options.ref, Paths: paths,
	})
	if err != nil {
		return err
	}
	units, err := diffpkg.Units(root, rawDiff, 6)
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
		Config: cfg, Cache: cacheStore, Version: Version, Verbose: log,
		Debug: options.debug, Audit: options.all,
	}
	if engine.WorkCount(units, task) > 0 {
		client, err := jev.NewProviderClient(cfg.Provider, jev.HTTPOptions{
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
	if options.verbose || options.debug {
		log("checks=%d requests=%d cache_hits=%d duration=%s",
			report.Summary.SemanticChecks, report.Summary.JevRequests,
			report.Summary.CacheHits, time.Since(started).Round(time.Millisecond))
	}
	if options.format == "json" {
		err = diagnostics.WriteJSON(command.OutOrStdout(), report)
	} else {
		err = diagnostics.WriteText(command.OutOrStdout(), report)
	}
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if !report.Passed {
		return &ExitError{Code: 1, Err: errors.New("blocking semantic violations found")}
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
