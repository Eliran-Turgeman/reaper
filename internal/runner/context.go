package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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
	output, err := exec.CommandContext(ctx, "git", "-C", r.Root, "ls-files", "-z").Output()
	if err != nil {
		return "", fmt.Errorf("list repository context files: %w", err)
	}
	files := strings.Split(string(output), "\x00")
	sort.Strings(files)
	var b strings.Builder
	hits := 0
	for _, file := range files {
		if !semantic.IsSupportedSource(file) || semantic.IsMinifiedSource(file) || !r.matches(rule, file) {
			continue
		}
		full := filepath.Join(r.Root, filepath.FromSlash(file))
		info, err := os.Lstat(full)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if !info.Mode().IsRegular() || info.Size() > 1<<20 {
			continue
		}
		data, err := os.ReadFile(full)
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
