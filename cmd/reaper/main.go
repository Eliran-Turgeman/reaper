package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Eliran-Turgeman/reaper/cmd"
)

func main() {
	root := cmd.New()
	if err := root.Execute(); err != nil {
		var exit *cmd.ExitError
		if !errors.As(err, &exit) || exit.Code != 1 {
			fmt.Fprintln(root.ErrOrStderr(), "reaper:", err)
		}
		os.Exit(cmd.ExitCode(err))
	}
}
