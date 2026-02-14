package scum

// processUniversalTag handles a tag whose open and close bytes are the same
// (e.g. $$ for bold, * for italic).
//
// If the tag is already open AND is the innermost (top of stack), this
// occurrence is treated as the closer and delegates to [closeTag].
// Otherwise it is treated as an opener and delegates to [processOpeningTag]
// (which will reject it with a duplicate-nesting warning if the tag is open
// but not at the top of the stack).
func processUniversalTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	if state.openedTags[tok.Trigger] && state.peekStack() == tok.Trigger {
		closeTag(state, tok)
		return
	}

	processOpeningTag(state, d, warns, tok)
}
