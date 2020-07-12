package parser

import (
	ast "../ast"
	text "../text"
	util "../util"
)

type codeSpanParser struct {
}

var defaultCodeSpanParser = &codeSpanParser{}

func NewCodeSpanParser() InlineParser {
	return defaultCodeSpanParser
}

func (self *codeSpanParser) Trigger() []byte {
	return []byte{'`'}
}

func (s *codeSpanParser) Parse(parent ast.Node, block text.Reader, pc Context) ast.Node {
	line, startSegment := block.PeekLine()
	opener := 0
	for ; opener < len(line) && line[opener] == '`'; opener++ {
	}
	block.Advance(opener)
	l, pos := block.Position()
	node := ast.NewCodeSpan()
	for {
		line, segment := block.PeekLine()
		if line == nil {
			block.SetPosition(l, pos)
			return ast.NewTextSegment(startSegment.WithStop(startSegment.Start + opener))
		}
		for i := 0; i < len(line); i++ {
			c := line[i]
			if c == '`' {
				oldi := i
				for ; i < len(line) && line[i] == '`'; i++ {
				}
				closure := i - oldi
				if closure == opener && (i >= len(line) || line[i] != '`') {
					segment = segment.WithStop(segment.Start + i - closure)
					if !segment.IsEmpty() {
						node.AppendChild(node, ast.NewRawTextSegment(segment))
					}
					block.Advance(i)
					goto end
				}
			}
		}
		if !util.IsBlank(line) {
			node.AppendChild(node, ast.NewRawTextSegment(segment))
		}
		block.AdvanceLine()
	}
end:
	if !node.IsBlank(block.Source()) {
		// trim first halfspace and last halfspace
		segment := node.FirstChild().(*ast.Text).Segment
		shouldTrimmed := true
		if !(!segment.IsEmpty() && block.Source()[segment.Start] == ' ') {
			shouldTrimmed = false
		}
		segment = node.LastChild().(*ast.Text).Segment
		if !(!segment.IsEmpty() && block.Source()[segment.Stop-1] == ' ') {
			shouldTrimmed = false
		}
		if shouldTrimmed {
			t := node.FirstChild().(*ast.Text)
			segment := t.Segment
			t.Segment = segment.WithStart(segment.Start + 1)
			t = node.LastChild().(*ast.Text)
			segment = node.LastChild().(*ast.Text).Segment
			t.Segment = segment.WithStop(segment.Stop - 1)
		}

	}
	return node
}
