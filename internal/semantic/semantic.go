package semantic

import (
	"path/filepath"
	"strings"
)

const SchemaVersion = 1

type Unit struct {
	FilePath         string
	Language         string
	OldContent       string
	NewContent       string
	Diff             string
	SurroundingCode  string
	StartLine        int
	EndLine          int
	IsTest           bool
	ContainsComments bool
	ExistingModified bool
	Additions        int
}

func Language(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".cs":
		return "csharp"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	default:
		return "text"
	}
}

func IsTestFile(path string) bool {
	p := strings.ToLower(filepath.ToSlash(path))
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(p, "/test/") ||
		strings.Contains(p, "/tests/") ||
		strings.Contains(p, "/__tests__/") ||
		strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_test.py") ||
		strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") ||
		strings.HasSuffix(base, "tests.cs") ||
		strings.HasSuffix(base, "test.java")
}

func ContainsComment(language, code string) bool {
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		switch language {
		case "python", "ruby":
			if strings.HasPrefix(trimmed, "#") {
				return true
			}
		default:
			if strings.HasPrefix(trimmed, "//") ||
				strings.HasPrefix(trimmed, "/*") ||
				strings.HasPrefix(trimmed, "*") ||
				strings.Contains(trimmed, " //") {
				return true
			}
		}
	}
	return false
}
