package scum

import (
	"strconv"
)

// TODO: document steps for Myself
type parserState struct {
	ast         AST
	breadcrumbs []int
	cumWidth    []int
	skip        [256]int
	openedTags  [256]bool
	stack       []byte
	maxDepth    int
	lastNodeIdx int
}

func (s parserState) peekCrumb() int {
	// since there will always be a root node, 0 len should not be an issue
	return s.breadcrumbs[len(s.breadcrumbs)-1]
}

func (s *parserState) popCrumb() int {
	lastItemIdx := len(s.breadcrumbs) - 1
	lastItem := s.breadcrumbs[lastItemIdx]
	s.breadcrumbs = s.breadcrumbs[:lastItemIdx]
	return lastItem
}

func (s *parserState) pushCrumb(idx int) {
	s.breadcrumbs = append(s.breadcrumbs, idx)
}

func (s parserState) peekStack() byte {
	l := len(s.stack)
	if l > 0 {
		return s.stack[l-1]
	}

	return 0
}

func (s *parserState) popStack() byte {
	lastItemIdx := len(s.stack) - 1
	lastItem := s.stack[lastItemIdx]
	s.stack = s.stack[:lastItemIdx]
	return lastItem
}

func (s *parserState) pushStack(b byte) {
	s.stack = append(s.stack, b)
	s.maxDepth = max(s.maxDepth, len(s.stack))
}

func (s *parserState) incrementCumWidth(w int) {
	s.cumWidth[len(s.cumWidth)-1] += w
}

func (s *parserState) pushCumWidth(w int) {
	s.cumWidth = append(s.cumWidth, w)
}

func (s *parserState) popCumWidth(w int) int {
	lastItemIdx := len(s.cumWidth) - 1
	lastItem := s.cumWidth[lastItemIdx]
	delta := lastItem + w
	s.cumWidth[lastItemIdx-1] += delta
	s.cumWidth = s.cumWidth[:lastItemIdx]
	return delta
}

func (s *parserState) peekCumWidth() int {
	return s.cumWidth[len(s.cumWidth)-1]
}

func newParserState(input string, out TokenizerOutput) parserState {
	root := NewNode()

	// we expect to have at most as many nodes as there are tags and text tokens
	totalExpectedNodes := max(out.TagsTotal+out.TextTokens, 1)
	nodes := make([]Node, 1, totalExpectedNodes)
	nodes[0] = root

	ast := AST{
		Input:      input,
		Nodes:      nodes,
		Attributes: make([]Attribute, 0, out.Attributes),
	}

	return parserState{
		ast: ast,
		// root node should always be present
		breadcrumbs: []int{0},
		// cumulative width of the root should always be present
		cumWidth: []int{0},
		maxDepth: 1,
	}
}

func Parse(input string, d *Dictionary, warns *Warnings) AST {
	out := Tokenize(d, input, warns)

	state := newParserState(input, out)

	for _, t := range out.Tokens {
		switch t.Type {
		case TokenText:
			processText(&state, t)

		case TokenAttributeFlag, TokenAttributeKV:
			processAttribute(&state, t)

		case TokenTag:
			processTag(&state, d, warns, t)
		}
	}

	for len(state.breadcrumbs) > 1 {
		idx := state.popCrumb()
		childWidth := state.popCumWidth(0) // no closing tag
		// Use assignment, not +=, because childWidth already includes opening tag width
		state.ast.Nodes[idx].Span.End = state.ast.Nodes[idx].Span.Start + childWidth

		openTagID := state.popStack()
		state.openedTags[openTagID] = false

		desc := "missing closing Tag for the Tag with ID " +
			strconv.QuoteRune(rune(openTagID)) +
			" at position " +
			strconv.Itoa(state.ast.Nodes[idx].Span.Start) + "."

		warns.Add(Warning{
			Issue:       IssueUnclosedTag,
			Pos:         state.ast.Nodes[idx].Span.Start,
			Description: desc,
		})
	}

	// then finalize root
	state.ast.Nodes[0].Span.End = state.peekCumWidth()

	state.ast.MaxDepth = state.maxDepth

	return state.ast
}

func appendNode(ast *AST, parentIdx int, node Node) int {
	nodeIdx := len(ast.Nodes)
	ast.Nodes = append(ast.Nodes, node)

	parent := &ast.Nodes[parentIdx]
	parent.ChildCount++

	if parent.FirstChild == -1 {
		parent.FirstChild = nodeIdx
		parent.LastChild = nodeIdx
		return nodeIdx
	}

	lastChild := &ast.Nodes[parent.LastChild]
	lastChild.NextSibling = nodeIdx
	parent.LastChild = nodeIdx

	return nodeIdx
}

func appendAttribute(ast *AST, parentIdx int, attr Attribute) {
	attrIdx := len(ast.Attributes)
	ast.Attributes = append(ast.Attributes, attr)

	parent := &ast.Nodes[parentIdx]

	if parent.Attributes.Len == 0 {
		parent.Attributes.Start = attrIdx
	}

	parent.Attributes.Len++
}

func processTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	if state.skip[tok.Trigger] > 0 {
		state.skip[tok.Trigger]--
		return
	}

	tag := d.tags[tok.Trigger]

	// if the tag is greedy - just append it to the last crumb, and its payload to it
	if tag.Greed > NonGreedy {
		appendGreedyNode(state, tok)
		return
	}

	switch {
	case tag.IsUniversal():
		processUniversalTag(state, d, warns, tok)
		return

	case tag.IsOpening():
		processOpeningTag(state, d, warns, tok)
		return

	case tag.IsClosing():
		processClosingTag(state, d, warns, tok)
	}
}

// TODO: doc comment the behaviour of adding two nodes, one for the tag itself and other
// for the inner content
func appendGreedyNode(state *parserState, tok Token) {
	node := NewNode()
	node.Type = NodeTag
	node.TagID = tok.Trigger
	node.Span = NewSpan(tok.Pos, tok.Width)
	parentIdx := appendNode(&state.ast, state.peekCrumb(), node)
	payload := NewNode()
	payload.Type = NodeText
	payload.Span = tok.Payload
	nodeIdx := appendNode(&state.ast, parentIdx, payload)
	state.lastNodeIdx = nodeIdx
	state.incrementCumWidth(tok.Width)
}

func processUniversalTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	// 1. Check if the tag is a closing one and close if true
	if state.openedTags[tok.Trigger] && state.peekStack() == tok.Trigger {
		closeTag(state, tok)
		return
	}

	// 2. Process the tag as an opening one.
	processOpeningTag(state, d, warns, tok)
}

func processOpeningTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	// 1. Check if the tag is opened already, skip if true
	if state.openedTags[tok.Trigger] {
		desc := "tag with ID " +
			strconv.QuoteRune(rune(tok.Trigger)) +
			" is a descendant of the tag with the same ID."

		warns.Add(Warning{
			Issue:       IssueDuplicateNestedTag,
			Pos:         tok.Pos,
			Description: desc,
		})

		state.skip[d.tags[tok.Trigger].CloseID]++
		return
	}

	// 2. Else, open new Tag
	node := NewNode()
	node.Type = NodeTag
	node.TagID = tok.Trigger
	node.Span = NewSpan(tok.Pos, tok.Width)
	idx := appendNode(&state.ast, state.peekCrumb(), node)
	state.pushCrumb(idx)
	state.pushStack(tok.Trigger)
	state.pushCumWidth(tok.Width)
	state.lastNodeIdx = idx
	state.openedTags[tok.Trigger] = true
}

func processClosingTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	stacked := state.peekStack()

	tag := d.tags[tok.Trigger]

	// TODO: doc comment this behaviour
	// 1. If stack is empty add a Warning and return
	if stacked == 0 {
		// FIXME: refactor message
		desc := "closing tag with ID " +
			strconv.QuoteRune(rune(tok.Trigger)) +
			" expected to have an opening counterpart with ID " +
			strconv.QuoteRune(rune(tag.OpenID)) + " which is missing in the input."

		warns.Add(Warning{
			Issue:       IssueMisplacedClosingTag,
			Pos:         tok.Pos,
			Description: desc,
		})
		return
	}

	openTag := d.tags[stacked]

	// TODO: doc comment this behaviour
	// 2. If the opening and closing Tags mismatched add a Warning and return
	if (openTag.CloseID != tok.Trigger) && (tag.OpenID != stacked) {
		// FIXME: refactor message
		desc := "closing tag with ID " +
			strconv.QuoteRune(rune(tok.Trigger)) +
			" cannot match with opening tag with ID " +
			strconv.QuoteRune(rune(stacked))

		warns.Add(Warning{
			Issue:       IssueOpenCloseTagMismatch,
			Pos:         tok.Pos,
			Description: desc,
		})
		return
	}

	// 3. Otherwise close the Tag
	closeTag(state, tok)
}

func closeTag(state *parserState, tok Token) {
	idx := state.popCrumb()
	// making the closed tag the target for appending attributes
	state.lastNodeIdx = idx

	// update the span of the closed tag
	// Use assignment: Span.End = Start + cumWidth (includes opening + children) + closingWidth
	// This correctly handles tags with different opening/closing widths
	state.ast.Nodes[idx].Span.End = state.ast.Nodes[idx].Span.Start + state.popCumWidth(tok.Width)

	openTagID := state.popStack()
	state.openedTags[openTagID] = false
}

func processText(state *parserState, tok Token) {
	node := NewNode()
	node.Type = NodeText
	node.Span = tok.Payload
	textIdx := appendNode(&state.ast, state.peekCrumb(), node)
	state.lastNodeIdx = textIdx
	state.incrementCumWidth(tok.Width)
}

func processAttribute(state *parserState, tok Token) {
	attr := Attribute{
		Name:    tok.AttrKey,
		Payload: tok.Payload,
		IsFlag:  tok.Type == TokenAttributeFlag,
	}

	appendAttribute(&state.ast, state.lastNodeIdx, attr)
}
