package cmd

import (
	"encoding/json"
	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	"github.com/Eliran-Turgeman/reaper/internal/feedback"
	"github.com/spf13/cobra"
	"os"
)

func newFeedback() *cobra.Command {
	var path, input string
	read := func() (diagnostics.Report, error) {
		data, err := os.ReadFile(input)
		if err != nil {
			return diagnostics.Report{}, err
		}
		var report diagnostics.Report
		err = json.Unmarshal(data, &report)
		return report, err
	}
	command := &cobra.Command{Use: "feedback <finding-id> <useful|false-positive>", Short: "Record local finding feedback", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		report, err := read()
		if err != nil {
			return err
		}
		return feedback.Record(path, report, args[0], args[1])
	}}
	command.PersistentFlags().StringVar(&path, "file", ".reaper-feedback.jsonl", "local feedback file")
	command.PersistentFlags().StringVar(&input, "report", "", "JSON finding report")
	_ = command.MarkPersistentFlagRequired("report")
	command.AddCommand(&cobra.Command{Use: "metrics", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		report, err := read()
		if err != nil {
			return err
		}
		metrics, err := feedback.Metrics(path, report)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(metrics)
	}})
	return command
}
