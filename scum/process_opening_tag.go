package scum

// processOpeningTag handles a tag token that opens a new nesting level.
//
// If the same tag ID is already open in the ancestor chain, the token is
// rejected with an [IssueDuplicateNestedTag] warning, and the corresponding
// close tag's skip counter is incremented so its future closer is also
// discarded.
//
// Otherwise, a new [NodeTag] is created and appended as a child of the current
// parent. The three parallel stacks (breadcrumbs, cumWidth, stack) are all
// pushed to reflect the new depth.
func processOpeningTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	if state.openedTags[tok.Trigger] {
		warns.Add(Warning{
			Issue: IssueDuplicateNestedTag,
			Pos:   tok.Pos,
			TagID: tok.Trigger,
		})

		state.skip[d.tags[tok.Trigger].CloseID]++
		return
	}

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
	state.totalTagNodes++
}
