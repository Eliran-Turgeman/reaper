package semantic

import (
	"go/ast"
	"go/parser"
	"go/token"
)

func GoCandidateFacts(source string, first, last int) (bool, bool, bool) {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "", source, parser.SkipObjectResolution)
	if err != nil {
		return false, false, false
	}
	booleanInput, forwarder := false, false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || set.Position(fn.End()).Line < first || set.Position(fn.Pos()).Line > last {
			continue
		}
		for _, field := range fn.Type.Params.List {
			id, ok := field.Type.(*ast.Ident)
			if ok && id.Name == "bool" {
				line := set.Position(field.Pos()).Line
				if line >= first && line <= last {
					booleanInput = true
				}
			}
		}
		if fn.Body != nil && len(fn.Body.List) == 1 {
			candidate := false
			switch statement := fn.Body.List[0].(type) {
			case *ast.ReturnStmt:
				candidate = len(statement.Results) == 1 && isCall(statement.Results[0])
			case *ast.ExprStmt:
				candidate = isCall(statement.X)
			}
			if candidate {
				forwarder = true
			}
		}
	}
	return true, booleanInput, forwarder
}

func isCall(expression ast.Expr) bool {
	_, ok := expression.(*ast.CallExpr)
	return ok
}
