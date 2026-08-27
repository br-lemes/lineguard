package lineguard

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

func (c *checker) checkCompositeLit(lit *ast.CompositeLit) {
	// Composite literal diagnostics do not provide SuggestedFixes because
	// expanding them safely requires reconstructing indentation, nested
	// literals, function literals, and comments. gofmt is the appropriate
	// formatter for these structural transformations.
	if len(lit.Elts) == 0 {
		return
	}
	if c.isSingleElementArrayOrSlice(lit) && c.singleElementNeedsInlineParent(lit) {
		if !c.sameLine(lit.Lbrace, lit.Elts[0].Pos()) {
			c.reportf(lit.Lbrace, "single-element array or slice must keep its element on the opening line")
		}
		return
	}

	width := c.reconstructedWidth(lit.Pos(), lit)
	multiLine := !c.sameLine(lit.Pos(), lit.End())
	matchSiblings := c.mustMatchMultilineSibling(lit)

	switch {
	case !multiLine && matchSiblings:
		c.reportf(lit.Lbrace, "literal must use multiline sibling layout")
	case !multiLine && width > maxLineWidth:
		c.reportf(lit.Lbrace, "literal exceeds 80 columns and must be split across multiple lines")
	case multiLine && width <= maxLineWidth:
		if !matchSiblings {
			c.reportf(lit.Lbrace, "literal fits within 80 columns and must be on a single line")
		}
	case multiLine && width > maxLineWidth:
		if !c.validMultiLineLiteral(lit) {
			c.reportf(lit.Lbrace, "invalid multi-line literal spacing")
		}
	}
}

func (c *checker) isSingleElementArrayOrSlice(lit *ast.CompositeLit) bool {
	if len(lit.Elts) != 1 {
		return false
	}
	_, ok := lit.Type.(*ast.ArrayType)
	return ok
}

func (c *checker) singleElementNeedsInlineParent(lit *ast.CompositeLit) bool {
	return c.reconstructedWidth(lit.Elts[0].Pos(), lit.Elts[0]) > maxLineWidth
}

// Sibling literals share multiline layout when one of them requires it.
func (c *checker) mustMatchMultilineSibling(lit *ast.CompositeLit) bool {
	parent := c.compositeParents[lit]
	if parent == nil || c.sameLine(parent.Pos(), parent.End()) {
		return false
	}
	for _, elt := range parent.Elts {
		sibling, ok := elt.(*ast.CompositeLit)
		if !ok || sibling == lit {
			continue
		}
		if c.reconstructedWidth(sibling.Pos(), sibling) > maxLineWidth {
			return true
		}
	}
	return false
}

func (c *checker) validMultiLineLiteral(lit *ast.CompositeLit) bool {
	if c.sameLine(lit.Lbrace, lit.Elts[0].Pos()) {
		return false
	}

	if c.sameLine(lit.Elts[len(lit.Elts)-1].End(), lit.Rbrace) {
		return false
	}

	for i := 1; i < len(lit.Elts); i++ {
		if c.sameLine(lit.Elts[i-1].End(), lit.Elts[i].Pos()) {
			return false
		}
	}

	return true
}

func isChainCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	inner, ok := sel.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	_, ok = inner.Fun.(*ast.SelectorExpr)
	return ok
}

func collectChain(call *ast.CallExpr) (base ast.Expr, calls []*ast.CallExpr) {
	cur := call
	for {
		sel, ok := cur.Fun.(*ast.SelectorExpr)
		if !ok {
			//+gocover:ignore:block rare call without selector
			base = cur
			break
		}

		calls = append(calls, cur)

		inner, ok := sel.X.(*ast.CallExpr)
		if ok {
			_, ok := inner.Fun.(*ast.SelectorExpr)
			if ok {
				cur = inner
				continue
			}
		}

		base = sel.X
		break
	}

	for i, j := 0, len(calls)-1; i < j; i, j = i+1, j-1 {
		calls[i], calls[j] = calls[j], calls[i]
	}

	return base, calls
}

func (c *checker) checkChain(outer *ast.CallExpr) {
	base, calls := collectChain(outer)

	// Avoid reprocessing chain segments during the AST walk.
	for _, call := range calls {
		c.skipChainCalls[call] = true
		c.checkCallSpacing(call)
	}

	const msg = "chained methods must be on a single line"

	var broken []*ast.SelectorExpr
	firstSel := calls[0].Fun.(*ast.SelectorExpr)
	if !c.sameLine(base.End(), firstSel.Sel.Pos()) {
		broken = append(broken, firstSel)
	}
	for i := 1; i < len(calls); i++ {
		prev := calls[i-1]
		nextSel := calls[i].Fun.(*ast.SelectorExpr)
		if !c.sameLine(prev.End(), nextSel.Sel.Pos()) {
			broken = append(broken, nextSel)
		}
	}
	if len(broken) > 0 {
		c.reportChainFixes(broken, msg)
	}
}

// One diagnostic owns all edits for a broken method chain.
func (c *checker) reportChainFixes(selectors []*ast.SelectorExpr, message string) {
	var edits []analysis.TextEdit
	for _, sel := range selectors {
		file := c.pass.Fset.File(sel.Sel.Pos())
		if file == nil {
			//+gocover:ignore:block missing source file
			c.reportf(sel.Sel.Pos(), "%s", message)
			return
		}
		source, err := c.sourceBytes(c.pass.Fset.Position(sel.Sel.Pos()).Filename)
		if err != nil {
			//+gocover:ignore:block source read failure
			c.reportf(sel.Sel.Pos(), "%s", message)
			return
		}
		end := file.Offset(sel.Sel.Pos())
		start := end
		for start > 0 && (source[start-1] == ' ' || source[start-1] == '\t' || source[start-1] == '\r' || source[start-1] == '\n') {
			start--
		}
		if start == 0 || source[start-1] != '.' {
			c.reportf(sel.Sel.Pos(), "%s", message)
			return
		}
		edit := analysis.TextEdit{
			Pos: token.Pos(file.Pos(start)),
			End: sel.Sel.Pos(),
		}
		edits = append(edits, edit)
	}
	c.reportEdits(selectors[len(selectors)-1].Sel.Pos(), edits, message)
}

func (c *checker) reportSelectorChainFix(reportPos token.Pos, outer *ast.SelectorExpr, message string) {
	var selectors []*ast.SelectorExpr
	for current := outer; ; {
		selectors = append(selectors, current)
		inner, ok := current.X.(*ast.SelectorExpr)
		if !ok {
			break
		}
		current = inner
	}

	var edits []analysis.TextEdit
	for _, sel := range selectors {
		file := c.pass.Fset.File(sel.Sel.Pos())
		if file == nil {
			//+gocover:ignore:block missing selector file
			c.reportf(reportPos, "%s", message)
			return
		}
		source, err := c.sourceBytes(c.pass.Fset.Position(sel.Sel.Pos()).Filename)
		if err != nil {
			//+gocover:ignore:block selector source failure
			c.reportf(reportPos, "%s", message)
			return
		}
		end := file.Offset(sel.Sel.Pos())
		start := end
		for start > 0 && (source[start-1] == ' ' || source[start-1] == '\t' || source[start-1] == '\r' || source[start-1] == '\n') {
			start--
		}
		if start == 0 || source[start-1] != '.' {
			continue
		}
		edit := analysis.TextEdit{
			Pos:     file.Pos(start),
			End:     sel.Sel.Pos(),
			NewText: []byte(" "),
		}
		edits = append(edits, edit)
	}
	if len(edits) == 0 {
		//+gocover:ignore:block no selector edits
		c.reportf(reportPos, "%s", message)
		return
	}
	c.reportEdits(reportPos, edits, message)
}

// Width is measured against a reconstructed single-line rendering, including
// the source text preceding the expression.
func singleLineString(expr ast.Expr) string {
	var buf bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces, Tabwidth: tabWidth}
	_ = cfg.Fprint(&buf, token.NewFileSet(), expr)
	return buf.String()
}

func (c *checker) reconstructedWidth(startPos token.Pos, expr ast.Expr) int {
	exprWidth := runeLen(singleLineString(expr))
	compact := c.compactPrefixWidth(startPos) + exprWidth
	return compact
}
