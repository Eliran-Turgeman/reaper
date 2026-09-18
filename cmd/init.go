package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/Eliran-Turgeman/repear/internal/config"
	"github.com/spf13/cobra"
)

func newInit(_ App) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a starter .reaper.yaml",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			data, err := config.Defaults().YAML()
			if err != nil {
				return fmt.Errorf("encode starter config: %w", err)
			}
			file, err := os.OpenFile(config.FileName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
			if errors.Is(err, os.ErrExist) {
				return fmt.Errorf("%s already exists", config.FileName)
			}
			if err != nil {
				return fmt.Errorf("create %s: %w", config.FileName, err)
			}
			if _, err := file.Write(data); err != nil {
				file.Close()
				return fmt.Errorf("write %s: %w", config.FileName, err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("close %s: %w", config.FileName, err)
			}
			fmt.Fprintf(command.OutOrStdout(), "Created %s\n", config.FileName)
			return nil
		},
	}
}
