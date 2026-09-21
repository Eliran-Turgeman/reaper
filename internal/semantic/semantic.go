package semantic

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

const SchemaVersion = 2

var sourceLanguagesByExtension = map[string]string{
	".c":     "c",
	".cc":    "cpp",
	".cpp":   "cpp",
	".cjs":   "javascript",
	".cs":    "csharp",
	".cts":   "typescript",
	".cxx":   "cpp",
	".dart":  "dart",
	".ex":    "elixir",
	".exs":   "elixir",
	".go":    "go",
	".h":     "c",
	".hh":    "cpp",
	".hpp":   "cpp",
	".hxx":   "cpp",
	".java":  "java",
	".js":    "javascript",
	".jsx":   "javascript",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".lua":   "lua",
	".m":     "objective-c",
	".mjs":   "javascript",
	".mm":    "objective-c",
	".mts":   "typescript",
	".php":   "php",
	".phtml": "php",
	".py":    "python",
	".rb":    "ruby",
	".rs":    "rust",
	".scala": "scala",
	".sc":    "scala",
	".swift": "swift",
	".ts":    "typescript",
	".tsx":   "typescript",
}

type Unit struct {
	RelatedEvidence  json.RawMessage
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
}

func Language(path string) string {
	if language, ok := sourceLanguagesByExtension[strings.ToLower(filepath.Ext(path))]; ok {
		return language
	}
	return "text"
}

func IsSupportedSource(path string) bool {
	_, ok := sourceLanguagesByExtension[strings.ToLower(filepath.Ext(path))]
	return ok
}

func IsMinifiedSource(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return IsSupportedSource(path) &&
		(strings.Contains(base, ".min.") || strings.Contains(base, "-min."))
}

func IsTestFile(path string) bool {
	p := "/" + strings.TrimLeft(strings.ToLower(filepath.ToSlash(path)), "/")
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(p, "/test/") ||
		strings.Contains(p, "/tests/") ||
		strings.Contains(p, "/__tests__/") ||
		strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_test.py") ||
		strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") ||
		strings.HasPrefix(base, "test_") ||
		hasAnySuffix(base,
			"_test.c", "_test.cc", "_test.cpp", "_test.cxx",
			"_test.dart", "_test.exs",
			"test.kt", "test.kts", "test.php", "test.scala",
			"test.swift", "tests.swift", "tests.m", "tests.mm",
		) ||
		strings.HasSuffix(base, "tests.cs") ||
		strings.HasSuffix(base, "test.java")
}

func ContainsComment(language, code string) bool {
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		switch language {
		case "elixir", "python", "ruby":
			if strings.HasPrefix(trimmed, "#") {
				return true
			}
		case "lua":
			if strings.HasPrefix(trimmed, "--") {
				return true
			}
		case "php":
			if strings.HasPrefix(trimmed, "#") || hasSlashComment(trimmed) {
				return true
			}
		default:
			if hasSlashComment(trimmed) {
				return true
			}
		}
	}
	return false
}

func hasAnySuffix(value string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}

func hasSlashComment(line string) bool {
	return strings.HasPrefix(line, "//") ||
		strings.HasPrefix(line, "/*") ||
		strings.HasPrefix(line, "*") ||
		strings.Contains(line, " //")
}
