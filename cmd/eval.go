package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Eliran-Turgeman/repear/internal/config"
	evalpkg "github.com/Eliran-Turgeman/repear/internal/eval"
	"github.com/Eliran-Turgeman/repear/internal/jev"
	"github.com/Eliran-Turgeman/repear/internal/rules"
	"github.com/spf13/cobra"
)

type evalOptions struct {
	rule         string
	threshold    float64
	thresholdSet bool
	format       string
	dir          string
	provider     string
	model        string
}

func newEval(app App) *cobra.Command {
	options := evalOptions{}
	command := &cobra.Command{
		Use:   "eval",
		Short: "Run the labeled semantic-rule benchmark",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if options.format != "text" && options.format != "json" {
				return fmt.Errorf("--format must be text or json")
			}
			if options.thresholdSet && (options.threshold < 0 || options.threshold > 1) {
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
			client, err := jev.NewProviderClient(cfg.Provider, jev.HTTPOptions{
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
			report, err := evalpkg.Run(command.Context(), client, dir, cfg.Provider, cfg.Model, options.rule, override, thresholds)
			if err != nil {
				return err
			}
			if options.format == "json" {
				return evalpkg.WriteJSON(command.OutOrStdout(), report)
			}
			return evalpkg.WriteText(command.OutOrStdout(), report)
		},
	}
	flags := command.Flags()
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
