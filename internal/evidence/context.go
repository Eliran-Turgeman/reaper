package evidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strings"

	"github.com/Eliran-Turgeman/reaper/internal/diff"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
)

const targetedEvidenceLimit = 16384

type codeEvidence struct {
	Side  string `json:"side"`
	File  string `json:"file"`
	Start int    `json:"start_line"`
	End   int    `json:"end_line"`
	Kind  string `json:"kind"`
	Code  string `json:"code"`
}

type Context struct {
	Protocol    string         `json:"protocol"`
	Scope       string         `json:"scope"`
	Snippets    []codeEvidence `json:"snippets"`
	Limitations []string       `json:"limitations"`
}

type Coverage struct {
	Complete bool
	Reasons  []string
}

func Assess(raw json.RawMessage) (Coverage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var report Context
	if err := decoder.Decode(&report); err != nil {
		return Coverage{}, fmt.Errorf("parse targeted evidence coverage: %w", err)
	}
	sides := map[string]bool{}
	for _, snippet := range report.Snippets {
		if snippet.Kind == "changed-function" {
			sides[snippet.Side] = true
		}
	}
	var reasons []string
	for _, side := range []string{"before", "after"} {
		if !sides[side] {
			reasons = append(reasons, side+": changed function evidence is missing")
		}
	}
	for _, limitation := range report.Limitations {
		lower := strings.ToLower(limitation)
		unresolved := strings.Contains(lower, "unresolved direct calls") &&
			!strings.Contains(lower, "unchanged unresolved direct calls")
		if strings.Contains(lower, "unsupported language") ||
			strings.Contains(lower, "source missing") ||
			strings.Contains(lower, "source unavailable") ||
			strings.Contains(lower, "did not parse") ||
			strings.Contains(lower, "could be located") ||
			unresolved ||
			strings.Contains(lower, "unresolved changed direct calls") ||
			strings.Contains(lower, "limit reached") ||
			strings.Contains(lower, "exceeds remaining") ||
			strings.Contains(lower, "was omitted") {
			reasons = append(reasons, limitation)
		}
	}
	return Coverage{Complete: len(reasons) == 0, Reasons: reasons}, nil
}

// This experiment is deliberately limited to Go syntax and a single call hop.
// It does not execute fixtures or claim type-checked symbol resolution.
func Add(beforeFiles, afterFiles map[string]string, patchText string, units []semantic.Unit, helpers bool) error {
	patches, err := diff.Parse(patchText)
	if err != nil {
		return err
	}
	for i := range units {
		u := &units[i]
		var selected *diff.FilePatch
		var hunk *diff.Hunk
		for p := range patches {
			name := patches[p].NewPath
			if name == "/dev/null" {
				name = patches[p].OldPath
			}
			if name != u.FilePath {
				continue
			}
			for h := range patches[p].Hunks {
				if patches[p].Hunks[h].NewStart == u.StartLine {
					selected, hunk = &patches[p], &patches[p].Hunks[h]
					break
				}
			}
		}
		if hunk == nil {
			return fmt.Errorf("no matching hunk for context: %s:%d", u.FilePath, u.StartLine)
		}
		evidence := Context{Protocol: "go-functions-v2", Scope: "Changed functions on each snapshot; optional one-hop same-package direct function candidates. Changed unresolved calls are incomplete evidence; identical unresolved calls on both sides are recorded without forcing abstention. Lexical evidence, not exhaustive symbol resolution."}
		if u.Language != "go" {
			evidence.Limitations = append(evidence.Limitations, "Unsupported language: retained original local context.")
		} else {
			before := collectSide(&evidence, beforeFiles, selected.OldPath, *hunk, true, helpers)
			after := collectSide(&evidence, afterFiles, selected.NewPath, *hunk, false, helpers)
			appendUnresolvedLimitations(&evidence, before.unresolved, after.unresolved)
		}
		data, err := json.Marshal(evidence)
		if err != nil {
			return err
		}
		if len(data) > targetedEvidenceLimit {
			return fmt.Errorf("targeted context metadata exceeds 16 KiB")
		}
		u.RelatedEvidence = data
	}
	return nil
}

func changedLines(h diff.Hunk, before bool) (int, int) {
	line := h.NewStart
	if before {
		line = h.OldStart
	}
	first, last, adjacent := 0, 0, 0
	for _, text := range h.Lines {
		if text == "" {
			continue
		}
		own := text[0] == '+' && !before || text[0] == '-' && before
		if !own && text[0] != ' ' && adjacent == 0 {
			adjacent = line
		}
		if own {
			if first == 0 {
				first = line
			}
			last = line
		}
		if text[0] == ' ' || own {
			line++
		}
	}
	// Insertions/deletions have no changed lines on one side; the adjacent
	// declaration is a candidate, not proof of a matching function identity.
	if first == 0 {
		first = max(1, adjacent)
		last = first
	}
	return first, last
}

type sideCoverage struct {
	unresolved []string
}

func collectSide(e *Context, files map[string]string, name string, h diff.Hunk, before, helpers bool) sideCoverage {
	side := "after"
	if before {
		side = "before"
	}
	if name == "/dev/null" {
		return sideCoverage{}
	}
	source, exists := files[name]
	if !exists {
		e.Limitations = append(e.Limitations, side+": source missing")
		return sideCoverage{}
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, name, source, parser.ParseComments)
	if err != nil {
		e.Limitations = append(e.Limitations, side+": changed file did not parse")
		return sideCoverage{}
	}
	first, last := changedLines(h, before)
	calls := map[string]map[string]bool{}
	focal := map[*ast.FuncDecl]bool{}
	var functions, selectedFunctions []*ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		functions = append(functions, fn)
		if set.Position(fn.End()).Line < first || set.Position(fn.Pos()).Line > last {
			continue
		}
		selectedFunctions = append(selectedFunctions, fn)
	}
	if len(selectedFunctions) == 0 {
		bestDistance := 3
		for _, fn := range functions {
			start, end := set.Position(fn.Pos()).Line, set.Position(fn.End()).Line
			distance := start - last
			if end < first {
				distance = first - end
			}
			if distance >= 0 && distance < bestDistance {
				selectedFunctions = []*ast.FuncDecl{fn}
				bestDistance = distance
			} else if distance == bestDistance {
				selectedFunctions = append(selectedFunctions, fn)
			}
		}
	}
	for _, fn := range selectedFunctions {
		focal[fn] = true
		if !appendEvidence(e, snippet(side, name, source, set, fn, "changed-function")) {
			return sideCoverage{}
		}
		if fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			// Locally bound function variables are not package-level helpers.
			if id.Obj != nil && id.Obj.Kind != ast.Fun {
				return true
			}
			if calls[id.Name] == nil {
				calls[id.Name] = map[string]bool{}
			}
			start, end := set.Position(call.Pos()), set.Position(call.End())
			calls[id.Name][strings.TrimSpace(source[start.Offset:end.Offset])] = true
			return true
		})
	}
	if len(focal) == 0 {
		e.Limitations = append(e.Limitations, side+": no changed function could be located")
		return sideCoverage{}
	}
	if !helpers {
		return sideCoverage{}
	}
	paths := make([]string, 0, len(files))
	for name := range files {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	found := map[string]bool{}
	bytesRead, scanned := 0, 0
	for _, candidate := range paths {
		if path.Dir(candidate) != path.Dir(name) || path.Ext(candidate) != ".go" || semantic.IsTestFile(candidate) {
			continue
		}
		code := files[candidate]
		if len(code) > 262144 || scanned >= 128 || bytesRead+len(code) > 1048576 {
			e.Limitations = append(e.Limitations, side+": helper search hit file/byte limit")
			break
		}
		scanned++
		bytesRead += len(code)
		candidateSet := token.NewFileSet()
		parsed, err := parser.ParseFile(candidateSet, candidate, code, parser.ParseComments)
		if err != nil {
			e.Limitations = append(e.Limitations, side+": helper candidate did not parse: "+candidate)
			continue
		}
		if parsed.Name.Name != file.Name.Name {
			continue
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if _, called := calls[fn.Name.Name]; !called {
				continue
			}
			found[fn.Name.Name] = true
			already := false
			if candidate == name {
				for f := range focal {
					if f.Name.Name == fn.Name.Name {
						already = true
					}
				}
			}
			if already {
				continue
			}
			if !appendEvidence(e, snippet(side, candidate, code, candidateSet, fn, "direct-helper-candidate")) {
				return sideCoverage{}
			}
		}
	}
	var unresolved []string
	for name, references := range calls {
		if found[name] {
			continue
		}
		for reference := range references {
			unresolved = append(unresolved, reference)
		}
	}
	sort.Strings(unresolved)
	e.Limitations = append(e.Limitations, side+": methods, imports, function values, transitive calls and build constraints are not resolved; duplicate definitions are candidates, not proof of enforcement")
	return sideCoverage{unresolved: unresolved}
}

func appendUnresolvedLimitations(e *Context, before, after []string) {
	beforeSet := make(map[string]bool, len(before))
	afterSet := make(map[string]bool, len(after))
	for _, call := range before {
		beforeSet[call] = true
	}
	for _, call := range after {
		afterSet[call] = true
	}
	var unchanged, beforeOnly, afterOnly []string
	for _, call := range before {
		if afterSet[call] {
			unchanged = append(unchanged, call)
		} else {
			beforeOnly = append(beforeOnly, call)
		}
	}
	for _, call := range after {
		if !beforeSet[call] {
			afterOnly = append(afterOnly, call)
		}
	}
	if len(unchanged) > 0 {
		e.Limitations = append(e.Limitations, fmt.Sprintf("unchanged unresolved direct calls (possibly builtins/conversions): %v", unchanged))
	}
	if len(beforeOnly) > 0 {
		e.Limitations = append(e.Limitations, fmt.Sprintf("before: unresolved changed direct calls (possibly builtins/conversions): %v", beforeOnly))
	}
	if len(afterOnly) > 0 {
		e.Limitations = append(e.Limitations, fmt.Sprintf("after: unresolved changed direct calls (possibly builtins/conversions): %v", afterOnly))
	}
}

func snippet(side, name, source string, set *token.FileSet, fn *ast.FuncDecl, kind string) codeEvidence {
	start, end := set.Position(fn.Pos()), set.Position(fn.End())
	return codeEvidence{Side: side, File: name, Start: start.Line, End: end.Line, Kind: kind, Code: source[start.Offset:end.Offset]}
}

func appendEvidence(e *Context, entry codeEvidence) bool {
	e.Snippets = append(e.Snippets, entry)
	data, _ := json.Marshal(e)
	// Reserve room for completeness notes; never cut a function in half.
	if len(data) > targetedEvidenceLimit-2048 {
		e.Snippets = e.Snippets[:len(e.Snippets)-1]
		e.Limitations = append(e.Limitations, "Evidence size limit reached; a complete function was omitted. Absence is not proof of no enforcement.")
		return false
	}
	return true
}
