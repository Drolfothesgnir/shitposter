package scum

func processText(state *parserState, tok Token) {
	node := NewNode()
	node.Type = NodeText
	node.Span = tok.Payload
	textIdx := appendNode(&state.ast, state.peekCrumb(), node)
	state.lastNodeIdx = textIdx
	state.incrementCumWidth(tok.Width)
}
