package cmd

import (
	"fmt"

	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/spf13/cobra"
)

func newRules(_ App) *cobra.Command {
	var pack string
	command := &cobra.Command{
		Use:   "rules",
		Short: "List built-in semantic rules",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if pack != "" && !rules.ValidPack(pack) {
				return fmt.Errorf("unknown pack %q", pack)
			}
			for _, rule := range rules.All() {
				if pack != "" && rule.Pack != pack {
					continue
				}
				fmt.Fprintf(command.OutOrStdout(), "%-24s %-13s %-7s %.2f  %s\n",
					rule.ID, rule.Pack, rule.DefaultSeverity, rule.DefaultThreshold, rule.Description)
			}
			return nil
		},
	}
	command.Flags().StringVar(&pack, "pack", "", "filter by rule pack")
	return command
}
