package scum

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
