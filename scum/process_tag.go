package scum

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
