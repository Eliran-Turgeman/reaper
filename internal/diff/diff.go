package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Eliran-Turgeman/reaper/internal/semantic"
)

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

type FilePatch struct {
	OldPath string
	NewPath string
	Hunks   []Hunk
}

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []string
}

func Parse(input string) ([]FilePatch, error) {
	var files []FilePatch
	var current *FilePatch
	var hunk *Hunk
	for _, line := range strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			parts := strings.SplitN(strings.TrimPrefix(line, "diff --git "), " ", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("malformed diff header %q", line)
			}
			files = append(files, FilePatch{OldPath: trimPrefix(parts[0]), NewPath: trimPrefix(parts[1])})
			current = &files[len(files)-1]
			hunk = nil
		case current != nil && hunk == nil && strings.HasPrefix(line, "new file mode "):
			current.OldPath = "/dev/null"
		case current != nil && hunk == nil && strings.HasPrefix(line, "--- "):
			current.OldPath = trimPrefix(strings.TrimPrefix(line, "--- "))
		case current != nil && hunk == nil && strings.HasPrefix(line, "+++ "):
			current.NewPath = trimPrefix(strings.TrimPrefix(line, "+++ "))
		case strings.HasPrefix(line, "@@ "):
			if current == nil {
				return nil, fmt.Errorf("hunk before file header")
			}
			match := hunkHeader.FindStringSubmatch(line)
			if match == nil {
				return nil, fmt.Errorf("malformed hunk header %q", line)
			}
			current.Hunks = append(current.Hunks, Hunk{
				OldStart: parseInt(match[1]), OldCount: parseCount(match[2]),
				NewStart: parseInt(match[3]), NewCount: parseCount(match[4]),
			})
			hunk = &current.Hunks[len(current.Hunks)-1]
		case hunk != nil && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-")):
			if !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
				hunk.Lines = append(hunk.Lines, line)
			}
		}
	}
	return files, nil
}

func Units(root, input string, contextLines int) ([]semantic.Unit, error) {
	return UnitsWithSource(input, contextLines, func(path string) ([]byte, error) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if os.IsNotExist(err) {
			return nil, nil
		}
		return data, err
	})
}

// UnitsWithSource reads surrounding code from the same snapshot as the diff's
// after side. Deleted files have no after-side source and are not read.
func UnitsWithSource(input string, contextLines int, readSource func(string) ([]byte, error)) ([]semantic.Unit, error) {
	patches, err := Parse(input)
	if err != nil {
		return nil, err
	}
	var units []semantic.Unit
	for _, patch := range patches {
		path := patch.NewPath
		if path == "/dev/null" {
			path = patch.OldPath
		}
		if !semantic.IsSupportedSource(path) || semantic.IsMinifiedSource(path) {
			continue
		}
		var source []byte
		if patch.NewPath != "/dev/null" {
			source, err = readSource(path)
			if err != nil {
				return nil, fmt.Errorf("read context for %s: %w", path, err)
			}
		}
		sourceLines := splitLines(string(source))
		for _, hunk := range patch.Hunks {
			oldLines, newLines := hunkContent(hunk)
			start := max(1, hunk.NewStart-contextLines)
			end := min(len(sourceLines), hunk.NewStart+max(hunk.NewCount, 1)+contextLines-1)
			surrounding := ""
			if start <= end && len(sourceLines) > 0 {
				surrounding = strings.Join(sourceLines[start-1:end], "\n")
			}
			diffText := strings.Join(hunk.Lines, "\n")
			language := semantic.Language(path)
			first, last := changedRange(hunk)
			if logical, ok := logicalContext(language, string(source), first, last); ok {
				surrounding = logical
			}
			units = append(units, semantic.Unit{
				FilePath: path, Language: language,
				OldContent: strings.Join(oldLines, "\n"), NewContent: strings.Join(newLines, "\n"),
				Diff: diffText, SurroundingCode: surrounding,
				StartLine: hunk.NewStart, EndLine: max(hunk.NewStart, hunk.NewStart+max(hunk.NewCount, 1)-1),
				IsTest: semantic.IsTestFile(path), ContainsComments: semantic.ContainsComment(language, addedContent(hunk)),
				ExistingModified: patch.OldPath != "/dev/null" && (hasPrefix(hunk.Lines, "+") || hasPrefix(hunk.Lines, "-")),
			})
		}
	}
	return units, nil
}

func hunkContent(h Hunk) ([]string, []string) {
	var oldLines, newLines []string
	for _, line := range h.Lines {
		if line == "" {
			continue
		}
		switch line[0] {
		case ' ':
			oldLines = append(oldLines, line[1:])
			newLines = append(newLines, line[1:])
		case '-':
			oldLines = append(oldLines, line[1:])
		case '+':
			newLines = append(newLines, line[1:])
		}
	}
	return oldLines, newLines
}

func addedContent(h Hunk) string {
	var b strings.Builder
	for _, line := range h.Lines {
		if strings.HasPrefix(line, "+") {
			b.WriteString(line[1:])
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func splitLines(value string) []string {
	if value == "" {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func trimPrefix(path string) string {
	// Git appends a tab separator to unquoted file headers containing spaces.
	// Literal tabs in filenames are escaped inside a quoted path.
	path, _, _ = strings.Cut(path, "\t")
	if strings.HasPrefix(path, "\"") {
		if unquoted, err := strconv.Unquote(path); err == nil {
			path = unquoted
		}
	}
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")
	return path
}

func parseInt(value string) int {
	n, _ := strconv.Atoi(value)
	return n
}

func parseCount(value string) int {
	if value == "" {
		return 1
	}
	return parseInt(value)
}

func hasPrefix(lines []string, prefix string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) && !strings.HasPrefix(line, prefix+prefix+prefix) {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
