package scum

import "strconv"

type parserState struct {
	ast         AST
	breadcrumbs []int
	skip        [256]int
	openedTags  [256]bool
	stack       []byte
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
}

func newParserState(input string) parserState {
	ast := AST{
		Input: input,
		Nodes: []Node{{
			Type: NodeRoot,
			Span: NewSpan(0, len(input)),
		}},
	}

	return parserState{
		ast: ast,
		// root node should always be present
		breadcrumbs: []int{0},
	}
}

func Parse(input string, d *Dictionary, warns *Warnings) AST {
	tokens := Tokenize(d, input, warns)

	state := newParserState(input)

	for _, t := range tokens {
		switch t.Type {
		case TokenText:
			processText(&state, t)

		case TokenAttributeFlag, TokenAttributeKV:
			processAttribute(&state, t)

		case TokenTag:
			processTag(&state, d, warns, t)
		}
	}

	return state.ast
}

func appendNode(ast *AST, parentIdx int, node Node) int {
	nodeIdx := len(ast.Nodes)
	ast.Nodes = append(ast.Nodes, node)

	parent := &ast.Nodes[parentIdx]

	lastChildIdx := parent.LastChild
	lastChild := &ast.Nodes[lastChildIdx]
	lastChild.NextSibling = nodeIdx

	parent.LastChild = nodeIdx

	if parent.FirstChild == 0 {
		parent.FirstChild = nodeIdx
	}

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
	node := Node{
		Type:  NodeTag,
		TagID: tok.Trigger,
		Span:  NewSpan(tok.Pos, tok.Width),
	}
	parentIdx := appendNode(&state.ast, state.peekCrumb(), node)

	payload := Node{
		Type: NodeText,
		Span: tok.Payload,
	}
	nodeIdx := appendNode(&state.ast, parentIdx, payload)
	state.lastNodeIdx = nodeIdx
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
	node := Node{
		Type:  NodeTag,
		TagID: tok.Trigger,
		Span:  NewSpan(tok.Pos, tok.Width),
	}

	idx := appendNode(&state.ast, state.peekCrumb(), node)
	state.pushCrumb(idx)
	state.pushStack(tok.Trigger)
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
	openTagID := state.popStack()
	state.openedTags[openTagID] = false
}

func processText(state *parserState, tok Token) {
	node := Node{
		Type: NodeText,
		Span: tok.Payload,
	}
	textIdx := appendNode(&state.ast, state.peekCrumb(), node)
	state.lastNodeIdx = textIdx
}

func processAttribute(state *parserState, tok Token) {
	attr := Attribute{
		Name:    tok.AttrKey,
		Payload: tok.Payload,
		IsFlag:  tok.Type == TokenAttributeFlag,
	}

	appendAttribute(&state.ast, state.lastNodeIdx, attr)
}
