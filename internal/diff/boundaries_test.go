package diff

import (
	"strings"
	"testing"
)

func TestLogicalBoundaries(t *testing.T) {
	for language, source := range map[string]string{
		"go":         "package p\nfunc changed() {\n println(1)\n}\nfunc other() {}",
		"python":     "def changed():\n    text = '}'\n    return text\ndef other():\n    pass",
		"javascript": "function changed() {\n const s = '}';\n return s;\n}\nfunction other() {}",
		"typescript": "function changed(): string {\n const s = '}';\n return s;\n}\nfunction other() {}",
		"java":       "class Example {\n void changed() {\n run();\n }\n void other() {}\n}",
		"csharp":     "class Example {\n void Changed() {\n Run();\n }\n void Other() {}\n}",
	} {
		context, ok := logicalContext(language, source, 3, 3)
		if !ok || strings.Contains(context, "other") || strings.Contains(context, "Other") {
			t.Fatalf("%s: %q %v", language, context, ok)
		}
	}
}

func TestMalformedLogicalContextFallsBack(t *testing.T) {
	for _, language := range []string{"go", "javascript", "java", "csharp", "typescript"} {
		if _, ok := logicalContext(language, "function changed() {", 1, 1); ok {
			t.Fatal(language)
		}
	}
}
