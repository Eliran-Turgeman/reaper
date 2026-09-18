package cmd

import (
	"fmt"

	"github.com/Eliran-Turgeman/repear/internal/rules"
	"github.com/spf13/cobra"
)

func newRules(_ App) *cobra.Command {
	return &cobra.Command{
		Use:   "rules",
		Short: "List built-in semantic rules",
		Args:  cobra.NoArgs,
		Run: func(command *cobra.Command, _ []string) {
			for _, rule := range rules.All() {
				fmt.Fprintf(command.OutOrStdout(), "%-24s %-7s %.2f  %s\n",
					rule.ID, rule.DefaultSeverity, rule.DefaultThreshold, rule.Description)
			}
		},
	}
}
