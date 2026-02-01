package scum

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
