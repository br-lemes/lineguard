package lineguard

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const maxLineWidth = 80

var Analyzer = &analysis.Analyzer{
	Name: "lineguard",
	Doc:  "enforces this project's line-break and column-width style rules",
	Run:  run,
}

type checker struct {
	pass             *analysis.Pass
	skipChainCalls   map[*ast.CallExpr]bool
	fileLines        map[string][]string
	fileSources      map[string][]byte
	compositeParents map[*ast.CompositeLit]*ast.CompositeLit
	namedStructs     map[*ast.StructType]bool
	literalStructs   map[*ast.StructType]bool
}

func run(pass *analysis.Pass) (interface{}, error) {
	c := &checker{
		pass:             pass,
		skipChainCalls:   map[*ast.CallExpr]bool{},
		fileLines:        map[string][]string{},
		fileSources:      map[string][]byte{},
		compositeParents: map[*ast.CompositeLit]*ast.CompositeLit{},
		namedStructs:     map[*ast.StructType]bool{},
		literalStructs:   map[*ast.StructType]bool{},
	}
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}
		c.checkCommentWidths(file)
		ast.Inspect(file, func(n ast.Node) bool {
			spec, isTypeSpec := n.(*ast.TypeSpec)
			if isTypeSpec {
				typ, isStructType := spec.Type.(*ast.StructType)
				if isStructType {
					c.namedStructs[typ] = true
				}
			}
			lit, isCompositeLit := n.(*ast.CompositeLit)
			if isCompositeLit {
				typ := inlineStructType(lit.Type)
				if typ != nil {
					c.literalStructs[typ] = true
				}
			}
			parent, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, elt := range parent.Elts {
				child, ok := elt.(*ast.CompositeLit)
				if ok {
					c.compositeParents[child] = parent
				}
			}
			return true
		})
		ast.Inspect(file, func(n ast.Node) bool {
			c.visit(n)
			return true
		})
	}
	return nil, nil
}

func (c *checker) visit(n ast.Node) {
	switch node := n.(type) {
	case *ast.AssignStmt:
		c.checkAssignStmt(node)
	case *ast.ValueSpec:
		c.checkValueSpec(node)
	case *ast.BinaryExpr:
		c.checkBinaryExpr(node)
	case *ast.FuncDecl:
		c.checkFuncDecl(node)
	case *ast.ReturnStmt:
		c.checkReturnStmt(node)
	case *ast.CallExpr:
		c.checkCallExpr(node)
	case *ast.SelectorExpr:
		c.checkSelectorExpr(node)
	case *ast.IfStmt:
		c.checkIfStmt(node)
	case *ast.ForStmt:
		c.checkForStmt(node)
	case *ast.SwitchStmt:
		c.checkSwitchStmt(node)
	case *ast.TypeSpec:
		c.checkTypeSpec(node)
	case *ast.StructType:
		c.checkStructType(node)
	case *ast.CompositeLit:
		c.checkCompositeLit(node)
	}
}

func (c *checker) reportf(pos token.Pos, format string, args ...interface{}) {
	c.pass.Reportf(pos, format, args...)
}

func (c *checker) reportDiagnostic(pos token.Pos, message string) {
	c.reportf(pos, "%s", message)
}

func (c *checker) reportEdit(reportPos token.Pos, edit analysis.TextEdit, message string) {
	c.reportEdits(reportPos, []analysis.TextEdit{edit}, message)
}

// A diagnostic may contain multiple independent edits. If any edit intersects
// a comment, no fix is suggested for the whole diagnostic.
func (c *checker) reportEdits(reportPos token.Pos, edits []analysis.TextEdit, message string) {
	for _, edit := range edits {
		if c.rangeHasComment(edit.Pos, edit.End) {
			c.reportDiagnostic(reportPos, message)
			return
		}
	}
	c.pass.Report(analysis.Diagnostic{
		Pos:     reportPos,
		Message: message,
		SuggestedFixes: []analysis.SuggestedFix{{
			Message:   "apply fix",
			TextEdits: edits,
		}},
	})
}

func (c *checker) rangeHasComment(start, end token.Pos) bool {
	startPos := c.pass.Fset.Position(start)
	endPos := c.pass.Fset.Position(end)
	if startPos.Filename != endPos.Filename {
		//+gocover:ignore:block positions from different files
		return true
	}

	for _, file := range c.pass.Files {
		filePos := c.pass.Fset.Position(file.Pos())
		if filePos.Filename == startPos.Filename {
			for _, group := range file.Comments {
				if group.End() <= start || group.Pos() >= end {
					continue
				}
				return true
			}
			return false
		}
	}
	//+gocover:ignore:block missing parsed file
	return true
}

func (c *checker) sourceBytes(filename string) ([]byte, error) {
	source, ok := c.fileSources[filename]
	if ok {
		return source, nil
	}
	source, err := c.pass.ReadFile(filename)
	if err != nil {
		//+gocover:ignore:block source read failure
		return nil, err
	}
	c.fileSources[filename] = source
	return source, nil
}

func (c *checker) line(pos token.Pos) int {
	return c.pass.Fset.Position(pos).Line
}

func (c *checker) sameLine(a, b token.Pos) bool {
	return c.line(a) == c.line(b)
}

func isLiteralExpr(e ast.Expr) bool {
	switch expr := e.(type) {
	case *ast.FuncLit, *ast.CompositeLit:
		return true
	case *ast.BasicLit:
		return expr.Kind == token.STRING && strings.HasPrefix(expr.Value, "`") && strings.Contains(expr.Value, "\n")
	}
	return false
}

func (c *checker) checkAssignStmt(a *ast.AssignStmt) {
	if len(a.Rhs) == 0 {
		//+gocover:ignore:block impossible empty RHS
		return
	}
	if !c.sameLine(a.TokPos, a.Rhs[0].Pos()) {
		edit := analysis.TextEdit{
			Pos:     a.TokPos + token.Pos(len(a.Tok.String())),
			End:     a.Rhs[0].Pos(),
			NewText: []byte(" "),
		}
		if a.Tok == token.DEFINE {
			c.reportEdit(a.Rhs[0].Pos(), edit, "invalid line break after operator :=")
		} else {
			c.reportEdit(a.Rhs[0].Pos(), edit, "invalid line break after operator =")
		}
	}
	for i := 1; i < len(a.Rhs); i++ {
		prev, next := a.Rhs[i-1], a.Rhs[i]
		if !c.sameLine(prev.End(), next.Pos()) {
			edit := analysis.TextEdit{
				Pos:     prev.End(),
				End:     next.Pos(),
				NewText: []byte(", "),
			}
			c.reportEdit(next.Pos(), edit, "invalid line break after assignment comma")
		}
	}
}

func (c *checker) checkValueSpec(v *ast.ValueSpec) {
	if len(v.Values) == 0 {
		return
	}
	for i := 1; i < len(v.Values); i++ {
		prev, next := v.Values[i-1], v.Values[i]
		if !c.sameLine(prev.End(), next.Pos()) {
			edit := analysis.TextEdit{
				Pos:     prev.End(),
				End:     next.Pos(),
				NewText: []byte(", "),
			}
			c.reportEdit(next.Pos(), edit, "invalid line break after assignment comma")
		}
	}
}

func (c *checker) checkBinaryExpr(b *ast.BinaryExpr) {
	if !c.sameLine(b.X.End(), b.Y.Pos()) {
		edit := analysis.TextEdit{
			Pos:     b.OpPos + token.Pos(len(b.Op.String())),
			End:     b.Y.Pos(),
			NewText: []byte(" "),
		}
		c.reportEdit(b.Y.Pos(), edit, "invalid line break after binary operator")
	}
}

func (c *checker) checkFuncDecl(f *ast.FuncDecl) {
	end := f.Type.Params.Closing
	if f.Type.Results != nil {
		end = f.Type.Results.End()
	}
	if !c.sameLine(f.Pos(), end) {
		c.reportf(f.Pos(), "function signature must be on a single line")
	}
}

func (c *checker) checkReturnStmt(r *ast.ReturnStmt) {
	if len(r.Results) == 0 {
		return
	}
	for i := 1; i < len(r.Results); i++ {
		prev, next := r.Results[i-1], r.Results[i]
		if !c.sameLine(prev.End(), next.Pos()) {
			edit := analysis.TextEdit{
				Pos:     prev.End(),
				End:     next.Pos(),
				NewText: []byte(", "),
			}
			if isLiteralExpr(prev) || isLiteralExpr(next) {
				c.reportEdit(next.Pos(), edit, "invalid multi-line return spacing")
			} else {
				c.reportEdit(next.Pos(), edit, "return statement must be on a single line")
			}
		}
	}
}

func (c *checker) checkCallExpr(call *ast.CallExpr) {
	if c.skipChainCalls[call] {
		return
	}
	if isChainCall(call) {
		c.checkChain(call)
		return
	}
	c.checkCallSpacing(call)
}

func (c *checker) checkCallSpacing(call *ast.CallExpr) {
	hasLiteralArg := false
	for _, arg := range call.Args {
		if isLiteralExpr(arg) {
			hasLiteralArg = true
			break
		}
	}
	msg := "function call must be on a single line"
	if hasLiteralArg {
		msg = "invalid multi-line arguments spacing"
	}

	var edits []analysis.TextEdit
	var reportPos token.Pos
	if len(call.Args) == 0 {
		return
	} else {
		if !c.sameLine(call.Lparen, call.Args[0].Pos()) {
			reportPos = call.Args[0].Pos()
			if isLiteralExpr(call.Args[0]) {
				reportPos = call.Lparen + 1
			}
			edits = append(edits, analysis.TextEdit{
				Pos:     call.Lparen + 1,
				End:     call.Args[0].Pos(),
				NewText: []byte(" "),
			})
		}
		for i := 1; i < len(call.Args); i++ {
			prev, next := call.Args[i-1], call.Args[i]
			if !c.sameLine(prev.End(), next.Pos()) {
				reportPos = next.Pos()
				edit := analysis.TextEdit{
					Pos:     prev.End(),
					End:     next.Pos(),
					NewText: []byte(", "),
				}
				edits = append(edits, edit)
			}
		}
		last := call.Args[len(call.Args)-1]
		if !c.sameLine(last.End(), call.Rparen) {
			reportPos = call.Rparen
			edit := analysis.TextEdit{
				Pos:     last.End(),
				End:     call.Rparen,
				NewText: []byte(" "),
			}
			edits = append(edits, edit)
		}
	}
	if len(edits) == 0 {
		return
	}
	for _, edit := range edits {
		if c.rangeHasComment(edit.Pos, edit.End) {
			c.reportf(reportPos, "%s", msg)
			return
		}
	}
	c.reportEdits(reportPos, edits, msg)
}

func (c *checker) checkSelectorExpr(sel *ast.SelectorExpr) {
	inner, ok := sel.X.(*ast.SelectorExpr)
	if !ok {
		return
	}

	base := inner
	for {
		next, ok := base.X.(*ast.SelectorExpr)
		if !ok {
			break
		}
		base = next
	}

	if !c.sameLine(base.Pos(), sel.End()) {
		c.reportSelectorChainFix(base.Pos(), sel, "chained selections must be on a single line")
	}
}

func (c *checker) checkIfStmt(s *ast.IfStmt) {
	if s.Init != nil {
		c.reportf(s.If, "if statements with initialization are not allowed")
	}
}

func (c *checker) checkSwitchStmt(s *ast.SwitchStmt) {
	if s.Init != nil {
		c.reportf(s.Switch, "switch statements with initialization are not allowed")
	}
}

func (c *checker) checkForStmt(f *ast.ForStmt) {
	if !c.sameLine(f.For, f.Body.Lbrace) {
		c.reportf(f.For, "invalid line break in for statement")
	}
}

func (c *checker) checkTypeSpec(t *ast.TypeSpec) {
	switch typ := t.Type.(type) {
	case *ast.InterfaceType:
		if len(typ.Methods.List) == 0 {
			if !c.sameLine(typ.Pos(), typ.End()) {
				c.reportf(typ.Pos(), "empty type declaration must be on a single line")
			}
		}
	}
}

func (c *checker) checkStructType(typ *ast.StructType) {
	if len(typ.Fields.List) == 0 {
		if !c.sameLine(typ.Pos(), typ.End()) {
			c.reportf(typ.Pos(), "empty type declaration must be on a single line")
		}
		return
	}
	if c.literalStructs[typ] && !c.namedStructs[typ] && len(typ.Fields.List) == 1 {
		if !c.sameLine(typ.Pos(), typ.End()) {
			c.reportf(typ.Pos(), "single-field inline struct type must be on a single line")
		}
		return
	}
	if c.sameLine(typ.Pos(), typ.End()) {
		c.reportf(typ.End(), "non-empty struct type must be on multiple lines")
	}
}
