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
	All    bool
	Staged bool
	Ref    string
	Paths  []string
}

type CommandCollector struct {
	Dir string
	// Env optionally isolates Git configuration and repository environment.
	Env []string
}

func (c CommandCollector) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = c.Env
	return cmd
}

func (c CommandCollector) Diff(ctx context.Context, opts Options) (string, string, error) {
	rootBytes, err := c.command(ctx, "-C", c.Dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", "", errors.New("current directory is not inside a Git repository")
	}
	root := strings.TrimSpace(string(rootBytes))
	args := []string{"-C", c.Dir, "--no-pager", "diff", "--no-ext-diff", "--unified=6"}
	switch {
	case opts.All:
		emptyTree, err := c.emptyTree(ctx)
		if err != nil {
			return "", root, err
		}
		args = append(args, emptyTree)
	case opts.Staged:
		args = append(args, "--cached")
	case opts.Ref != "":
		args = append(args, opts.Ref)
	}
	if len(opts.Paths) > 0 {
		args = append(args, "--")
		args = append(args, opts.Paths...)
	}
	cmd := c.command(ctx, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", root, fmt.Errorf("git diff: %s", strings.TrimSpace(string(output)))
	}
	return string(output), root, nil
}

func (c CommandCollector) emptyTree(ctx context.Context) (string, error) {
	cmd := c.command(ctx, "-C", c.Dir, "hash-object", "-t", "tree", "--stdin")
	cmd.Stdin = strings.NewReader("")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("create empty Git tree: %s", strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

// IndexSource returns the staged blob using a repository-relative path.
func (c CommandCollector) IndexSource(ctx context.Context, path string) ([]byte, error) {
	data, err := c.command(ctx, "-C", c.Dir, "show", ":"+path).Output()
	if err != nil {
		return nil, fmt.Errorf("read staged blob %s: %w", path, err)
	}
	return data, nil
}
