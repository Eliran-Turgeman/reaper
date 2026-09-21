package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

type App struct {
	Out    io.Writer
	ErrOut io.Writer
	Getenv func(string) string
}

func New() *cobra.Command {
	return NewWith(App{Out: os.Stdout, ErrOut: os.Stderr, Getenv: os.Getenv})
}

func NewWith(app App) *cobra.Command {
	root := &cobra.Command{
		Use:           "reaper",
		Short:         "Semantic lint for coding agents",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(app.Out)
	root.SetErr(app.ErrOut)
	root.AddCommand(
		newBaseline(),
		newFeedback(),
		newCheck(app),
		newInit(app),
		newRules(app),
		newEval(app),
		newReviewPack(),
		&cobra.Command{
			Use: "version", Short: "Print the Reaper version",
			Run: func(cmd *cobra.Command, _ []string) { fmt.Fprintln(cmd.OutOrStdout(), Version) },
		},
	)
	return root
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit *ExitError
	if errors.As(err, &exit) {
		return exit.Code
	}
	return 2
}
