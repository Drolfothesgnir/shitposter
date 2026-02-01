package scum

func processOpeningTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	// 1. Check if the tag is opened already, skip if true
	if state.openedTags[tok.Trigger] {
		warns.Add(Warning{
			Issue: IssueDuplicateNestedTag,
			Pos:   tok.Pos,
			TagID: tok.Trigger,
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
	state.totalTagNodes++
}
