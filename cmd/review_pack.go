package cmd

import (
	"fmt"
	"os"

	evalpkg "github.com/Eliran-Turgeman/reaper/internal/eval"
	"github.com/spf13/cobra"
)

func newReviewPack() *cobra.Command {
	var benchmarkDir, output, seed string
	var limit int
	command := &cobra.Command{
		Use:   "review-pack",
		Short: "Export a blinded benchmark package for independent review",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if benchmarkDir == "" {
				return fmt.Errorf("--benchmark-dir is required")
			}
			pack, err := evalpkg.BuildReviewPack(benchmarkDir, evalpkg.ReviewPackOptions{Seed: seed, Limit: limit})
			if err != nil {
				return err
			}
			if output == "" || output == "-" {
				return evalpkg.WriteReviewPack(command.OutOrStdout(), pack)
			}
			file, err := os.OpenFile(output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			if err := evalpkg.WriteReviewPack(file, pack); err != nil {
				_ = file.Close()
				return err
			}
			return file.Close()
		},
	}
	flags := command.Flags()
	flags.StringVar(&benchmarkDir, "benchmark-dir", "", "Git benchmark corpus containing cases.json")
	flags.StringVar(&output, "output", "-", "output JSON path, or - for stdout")
	flags.StringVar(&seed, "seed", "review-v1", "stable blinded case-order seed")
	flags.IntVar(&limit, "limit", 0, "maximum cases for a pilot pack; 0 exports the complete corpus")
	return command
}
