package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestEvalRejectsInvalidBenchmarkModesBeforeProviderSetup(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--benchmark-experiment", "spec.json"}, "requires --benchmark-dir and --benchmark-mode git"},
		{[]string{"--benchmark-mode", "unknown"}, "must be snippet or git"},
		{[]string{"--benchmark-mode", "git"}, "require --benchmark-dir"},
		{[]string{"--benchmark-dir", "cases", "--benchmark-grouping", "isolated"}, "requires --benchmark-mode git"},
		{[]string{"--benchmark-dir", "cases", "--benchmark-mode", "git", "--benchmark-grouping", "unknown"}, "must be configured or isolated"},
	} {
		var output bytes.Buffer
		command := NewWith(App{Out: &output, ErrOut: &output, Getenv: func(string) string { return "" }})
		command.SetArgs(append([]string{"eval"}, tc.args...))
		if err := command.Execute(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("args %v: got %v, want %s", tc.args, err, tc.want)
		}
	}
}
