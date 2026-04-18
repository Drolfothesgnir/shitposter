package scum

// processText creates a [NodeText] node from tok and appends it as a child of
// the current open tag (top of breadcrumbs). It also adds the token's byte
// width to the current depth's cumulative width.
func processText(state *parserState, tok Token, warns *Warnings) {
	node := NewNode()
	node.Type = NodeText
	node.Span = tok.Payload
	if textIdx, ok := appendStateNode(state, state.peekCrumb(), node, tok.Pos, warns); ok {
		state.lastNodeIdx = textIdx
	}
	state.incrementCumWidth(tok.Width)
}
