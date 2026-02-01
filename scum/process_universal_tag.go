package scum

func processUniversalTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	// 1. Check if the tag is a closing one and close if true
	if state.openedTags[tok.Trigger] && state.peekStack() == tok.Trigger {
		closeTag(state, tok)
		return
	}

	// 2. Process the tag as an opening one.
	processOpeningTag(state, d, warns, tok)
}
