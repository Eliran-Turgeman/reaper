package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Eliran-Turgeman/reaper/internal/baseline"
	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	"github.com/spf13/cobra"
)

func newBaseline() *cobra.Command {
	var input, output, reason string
	command := &cobra.Command{Use: "baseline", Short: "Manage accepted existing findings"}
	create := &cobra.Command{Use: "create", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		data, err := os.ReadFile(input)
		if err != nil {
			return err
		}
		var report diagnostics.Report
		if err := json.Unmarshal(data, &report); err != nil {
			return err
		}
		file, err := baseline.Create(report, reason)
		if err != nil {
			return err
		}
		data, err = json.MarshalIndent(file, "", "  ")
		if err != nil {
			return err
		}
		f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, writeErr := f.Write(append(data, '\n'))
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created %s with %d entries\n", output, len(file.Entries))
		return nil
	}}
	create.Flags().StringVar(&input, "report", "", "JSON report from reaper check --format json")
	create.Flags().StringVar(&output, "output", ".reaper-baseline.json", "new baseline file (must not exist)")
	create.Flags().StringVar(&reason, "reason", "", "reason for accepting existing findings")
	_ = create.MarkFlagRequired("report")
	command.AddCommand(create)
	return command
}
