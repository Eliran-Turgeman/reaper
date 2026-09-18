package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Collector interface {
	Diff(ctx context.Context, opts Options) (string, string, error)
}

type Options struct {
	Staged bool
	Ref    string
	Paths  []string
}

type CommandCollector struct {
	Dir string
}

func (c CommandCollector) Diff(ctx context.Context, opts Options) (string, string, error) {
	rootBytes, err := exec.CommandContext(ctx, "git", "-C", c.Dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", "", errors.New("current directory is not inside a Git repository")
	}
	root := strings.TrimSpace(string(rootBytes))
	args := []string{"-C", c.Dir, "--no-pager", "diff", "--no-ext-diff", "--unified=6"}
	switch {
	case opts.Staged:
		args = append(args, "--cached")
	case opts.Ref != "":
		args = append(args, opts.Ref)
	}
	if len(opts.Paths) > 0 {
		args = append(args, "--")
		args = append(args, opts.Paths...)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", root, fmt.Errorf("git diff: %s", strings.TrimSpace(string(output)))
	}
	return string(output), root, nil
}
