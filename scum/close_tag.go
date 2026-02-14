package scum

// closeTag finalizes an open tag when its closing token is encountered.
//
// It pops all three parallel stacks (breadcrumbs, cumWidth, stack), computes
// the tag's Span.End as Start + cumulative width (opening + children + closing),
// clears the tag's openedTags flag, and sets lastNodeIdx so that any trailing
// attributes are attached to this tag.
func closeTag(state *parserState, tok Token) {
	idx := state.popCrumb()
	state.lastNodeIdx = idx

	// Span.End = Start + (opening width + children width + closing width).
	// popCumWidth adds tok.Width (closer) and folds the total into the parent.
	state.ast.Nodes[idx].Span.End = state.ast.Nodes[idx].Span.Start + state.popCumWidth(tok.Width)

	openTagID := state.popStack()
	state.openedTags[openTagID] = false
}
