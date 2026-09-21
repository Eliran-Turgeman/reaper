package diff

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

type boundary struct{ start, end int }

var pythonDeclaration = regexp.MustCompile(`^\s*(?:async\s+)?(?:def|class)\s+\w+`)
var braceDeclaration = regexp.MustCompile(`(?:\b(?:function|class|interface|record)\s+\w+|\)|=>)\s*(?::[^{}]+)?$`)
var controlBlock = regexp.MustCompile(`^\s*(?:if|else|for|while|switch|catch|try|with|using|foreach|synchronized)\b`)

// logicalContext keeps the changed hunk intact and enriches its surrounding context.
func logicalContext(language, source string, first, last int) (string, bool) {
	lines := splitLines(source)
	var spans []boundary
	switch language {
	case "go":
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, "", source, parser.ParseComments)
		if err != nil {
			return "", false
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.FuncDecl, *ast.TypeSpec:
				spans = append(spans, boundary{set.Position(node.Pos()).Line, set.Position(node.End()).Line})
			}
			return true
		})
	case "python":
		masked, ok := maskCode(source, true)
		if !ok {
			return "", false
		}
		codeLines := splitLines(masked)
		for i, line := range codeLines {
			if !pythonDeclaration.MatchString(line) {
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			end := len(lines)
			for j := i + 1; j < len(codeLines); j++ {
				if strings.TrimSpace(codeLines[j]) != "" && len(codeLines[j])-len(strings.TrimLeft(codeLines[j], " \t")) <= indent {
					end = j
					break
				}
			}
			start := i + 1
			for start > 1 && strings.HasPrefix(strings.TrimSpace(lines[start-2]), "@") {
				start--
			}
			spans = append(spans, boundary{start, end})
		}
	case "typescript", "javascript", "csharp", "java":
		masked, ok := maskCode(source, false)
		if !ok {
			return "", false
		}
		type opening struct{ offset, line int }
		var stack []opening
		line := 1
		for i, ch := range masked {
			if ch == '\n' {
				line++
			}
			if ch == '{' {
				stack = append(stack, opening{i, line})
			}
			if ch != '}' {
				continue
			}
			if len(stack) == 0 {
				return "", false
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			start := strings.LastIndex(masked[:open.offset], "\n") + 1
			header := strings.TrimSpace(masked[start:open.offset])
			if braceDeclaration.MatchString(header) && !controlBlock.MatchString(header) {
				spans = append(spans, boundary{open.line, line})
			}
		}
		if len(stack) != 0 {
			return "", false
		}
	default:
		return "", false
	}
	best := boundary{0, len(lines) + 1}
	for _, span := range spans {
		if span.start <= first && span.end >= last && span.end-span.start < best.end-best.start {
			best = span
		}
	}
	if best.start < 1 || best.end > len(lines) {
		return "", false
	}
	return strings.Join(lines[best.start-1:best.end], "\n"), true
}

// Mask comments and literals before conservative non-Go boundary scanning.
func maskCode(source string, python bool) (string, bool) {
	b := []byte(source)
	for i := 0; i < len(b); {
		if b[i] == '#' && python || !python && i+1 < len(b) && string(b[i:i+2]) == "//" {
			for i < len(b) && b[i] != '\n' {
				b[i] = ' '
				i++
			}
			continue
		}
		if !python && i+1 < len(b) && string(b[i:i+2]) == "/*" {
			end := strings.Index(source[i+2:], "*/")
			if end < 0 {
				return "", false
			}
			end += i + 4
			for ; i < end; i++ {
				if b[i] != '\n' {
					b[i] = ' '
				}
			}
			continue
		}
		if b[i] != '\'' && b[i] != '"' && b[i] != '`' {
			i++
			continue
		}
		quote := b[i]
		width := 1
		if python && i+2 < len(b) && b[i+1] == quote && b[i+2] == quote {
			width = 3
		}
		j := i + width
		closed := false
		for j < len(b) {
			if b[j] == '\\' {
				j += 2
				continue
			}
			if j+width <= len(b) && source[j:j+width] == strings.Repeat(string(quote), width) {
				j += width
				closed = true
				break
			}
			j++
		}
		if !closed {
			return "", false
		}
		for ; i < j; i++ {
			if b[i] != '\n' {
				b[i] = ' '
			}
		}
	}
	return string(b), true
}

func changedRange(h Hunk) (int, int) {
	line := max(1, h.NewStart)
	first, last := 0, 0
	for _, text := range h.Lines {
		if strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-") {
			if first == 0 {
				first = line
			}
			last = line
		}
		if !strings.HasPrefix(text, "-") {
			line++
		}
	}
	if first == 0 {
		return h.NewStart, max(h.NewStart, h.NewStart+h.NewCount-1)
	}
	return first, last
}
