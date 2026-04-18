package scum

// appendGreedyNode handles a self-contained (greedy) tag token such as
// backtick code spans. It creates a [NodeTag] parent and a [NodeText] child
// for the payload in one shot, without pushing onto the breadcrumb/stack
// (since greedy tags are already fully closed by the tokenizer).
// The token's full width is added to the current depth's cumulative width.
func appendGreedyNode(state *parserState, tok Token, warns *Warnings) {
	node := NewNode()
	node.Type = NodeTag
	node.TagID = tok.Trigger
	node.Span = NewSpan(tok.Pos, tok.Width)
	parentIdx, ok := appendStateNode(state, state.peekCrumb(), node, tok.Pos, warns)
	if !ok {
		state.incrementCumWidth(tok.Width)
		return
	}
	payload := NewNode()
	payload.Type = NodeText
	payload.Span = tok.Payload
	if nodeIdx, ok := appendStateNode(state, parentIdx, payload, tok.Payload.Start, warns); ok {
		state.lastNodeIdx = nodeIdx
	} else {
		state.lastNodeIdx = parentIdx
	}
	state.incrementCumWidth(tok.Width)
}
