package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	repogit "github.com/Eliran-Turgeman/reaper/internal/git"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
)

var introducedSymbol = regexp.MustCompile(`\b(?:interface|class|type|trait)\s+([A-Za-z_][A-Za-z_0-9]*)`)

func (r *Runner) repositoryContext(ctx context.Context, unit semantic.Unit, rule rules.Rule) (string, error) {
	if rule.Context != "repository-search" || r.Root == "" {
		return "", nil
	}
	matches := introducedSymbol.FindAllStringSubmatch(unit.NewContent, 4)
	if len(matches) == 0 {
		return "No named introduced type found for targeted repository search; absence of results is not proof of no usage.", nil
	}
	var names []string
	for _, match := range matches {
		names = append(names, regexp.QuoteMeta(match[1]))
	}
	sort.Strings(names)
	pattern := regexp.MustCompile(`\b(?:` + strings.Join(names, "|") + `)\b`)
	var files []string
	if r.IndexSnapshot != nil {
		files = r.IndexSnapshot.Files()
	} else {
		cmd := exec.CommandContext(ctx, "git", "-C", r.Root, "ls-files", "-z")
		cmd.Env = r.GitEnv
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("list repository context files: %w", err)
		}
		files = strings.Split(string(output), "\x00")
	}
	sort.Strings(files)
	var b strings.Builder
	hits := 0
	for _, file := range files {
		if !semantic.IsSupportedSource(file) || semantic.IsMinifiedSource(file) || !r.matches(rule, file) {
			continue
		}
		data, err := r.contextSource(ctx, file, 1<<20)
		if errors.Is(err, repogit.ErrSourceTooLarge) {
			continue
		}
		if err != nil {
			return "", err
		}
		for i, line := range strings.Split(string(data), "\n") {
			if !pattern.MatchString(line) {
				continue
			}
			if hits >= 40 || b.Len()+len(line) > 12000 {
				b.WriteString("\nSearch truncated; do not infer absence of other uses.\n")
				return b.String(), nil
			}
			fmt.Fprintf(&b, "%s:%d: %s\n", file, i+1, line)
			hits++
		}
	}
	fmt.Fprintf(&b, "Search used tracked, supported, nonexcluded regular files <=1 MiB and named type references; it is lexical evidence, not proof of exhaustive symbol resolution.\n")
	return b.String(), nil
}

func (r *Runner) contextSource(ctx context.Context, file string, limit int64) ([]byte, error) {
	if r.IndexSnapshot != nil {
		return r.IndexSnapshot.Read(ctx, file, limit)
	}
	full := filepath.Join(r.Root, filepath.FromSlash(file))
	info, err := os.Lstat(full)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, nil
	}
	if info.Size() > limit {
		return nil, repogit.ErrSourceTooLarge
	}
	source, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer source.Close()
	data, err := io.ReadAll(io.LimitReader(source, limit+1))
	if int64(len(data)) > limit {
		return nil, repogit.ErrSourceTooLarge
	}
	return data, err
}
