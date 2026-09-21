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
		{[]string{"--quality-policy", "policy.json"}, "requires configured Git benchmarking"},
		{[]string{"--benchmark-dir", "cases", "--benchmark-experiment", "spec.json"}, "patch experiments require --benchmark-mode git"},
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

func TestReleasePolicyFailsBeforeProviderWithoutIndependentReview(t *testing.T) {
	var output bytes.Buffer
	command := NewWith(App{Out: &output, ErrOut: &output, Getenv: func(string) string { return "" }})
	command.SetArgs([]string{"eval", "--benchmark-dir", "../benchmarks/validation", "--benchmark-mode", "git", "--quality-policy", "../benchmarks/release-policy.json"})
	err := command.Execute()
	if ExitCode(err) != 1 || !strings.Contains(err.Error(), "independent corpus review is missing") {
		t.Fatalf("gate did not fail before provider setup: %v", err)
	}
	if !strings.Contains(output.String(), "independent corpus review is missing") {
		t.Fatal("exit-1 failure must explain the unmet release requirement")
	}
}
