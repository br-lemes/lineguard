package lineguard

import (
	"go/ast"
	"go/token"
	"strings"
)

const tabWidth = 4

func (c *checker) checkCommentWidths(file *ast.File) {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if !strings.HasPrefix(comment.Text, "//") {
				continue
			}
			_, isDirective := ast.ParseDirective(comment.Slash, comment.Text)
			if isDirective {
				continue
			}
			position := c.pass.Fset.Position(comment.Pos())
			lines := c.sourceLines(position.Filename)
			if position.Line < 1 || position.Line > len(lines) {
				//+gocover:ignore:block inconsistent comment source position
				continue
			}
			line := lines[position.Line-1]
			if strings.TrimSpace(line[:position.Column-1]) != "" {
				continue
			}
			if visualWidth(line) > maxLineWidth {
				if hasUnbreakableToken(line) {
					continue
				}
				c.reportf(comment.Pos(), "comment exceeds 80 columns")
			}
		}
	}
}

func visualWidth(text string) int {
	width := 0
	for _, r := range text {
		if r == '\t' {
			width += tabWidth
		} else {
			width++
		}
	}
	return width
}

func hasUnbreakableToken(line string) bool {
	for _, token := range strings.Fields(line) {
		if visualWidth(token) > maxLineWidth {
			return true
		}
	}
	return false
}

func (c *checker) sourceLines(filename string) []string {
	lines, ok := c.fileLines[filename]
	if ok {
		return lines
	}
	data, err := c.pass.ReadFile(filename)
	if err == nil {
		lines = strings.Split(string(data), "\n")
	}
	c.fileLines[filename] = lines
	return lines
}

func (c *checker) indentWidth(pos token.Pos) int {
	p := c.pass.Fset.Position(pos)
	lines := c.sourceLines(p.Filename)
	if lines == nil || p.Line-1 >= len(lines) {
		//+gocover:ignore:block impossible missing source
		return 0
	}
	lineText := lines[p.Line-1]
	col := p.Column - 1
	if col > len(lineText) {
		//+gocover:ignore:block rare column beyond line
		col = len(lineText)
	}
	return visualWidth(lineText[:col])
}

func (c *checker) compactPrefixWidth(pos token.Pos) int {
	p := c.pass.Fset.Position(pos)
	lines := c.sourceLines(p.Filename)
	if lines == nil || p.Line-1 >= len(lines) {
		//+gocover:ignore:block unavailable source lines
		return 0
	}
	prefix := lines[p.Line-1][:p.Column-1]
	trimmed := strings.TrimRight(prefix, " \t")
	if len(trimmed) < len(prefix) && strings.HasSuffix(trimmed, ":") {
		return visualWidth(trimmed) + 1
	}
	return visualWidth(prefix)
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
