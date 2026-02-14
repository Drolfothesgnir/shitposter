package scum

// processTag is the top-level dispatcher for tag tokens.
//
// It first checks if the token should be silently consumed (skip counter > 0,
// set when a duplicate nested tag's closer needs to be discarded). Otherwise
// it routes to the appropriate handler based on the tag's properties:
//   - Greedy tags (e.g. backtick code): handled atomically by [appendGreedyNode].
//   - Universal tags (same byte opens and closes, e.g. $$ or *): [processUniversalTag].
//   - Opening-only tags (e.g. [): [processOpeningTag].
//   - Closing-only tags (e.g. ]): [processClosingTag].
func processTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	if state.skip[tok.Trigger] > 0 {
		state.skip[tok.Trigger]--
		return
	}

	tag := d.tags[tok.Trigger]

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
