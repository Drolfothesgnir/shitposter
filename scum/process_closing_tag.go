package scum

func processClosingTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	stacked := state.peekStack()

	tag := d.tags[tok.Trigger]

	// 1. If stack is empty add a Warning and return
	if stacked == 0 {
		warns.Add(Warning{
			Issue:    IssueMisplacedClosingTag,
			Pos:      tok.Pos,
			TagID:    tok.Trigger,
			Expected: tag.OpenID,
		})
		return
	}

	openTag := d.tags[stacked]

	// 2. If the opening and closing Tags mismatched add a Warning, treat the closing Tag as a text and return
	if (openTag.CloseID != tok.Trigger) && (tag.OpenID != stacked) {
		warns.Add(Warning{
			Issue:    IssueOpenCloseTagMismatch,
			Pos:      tok.Pos,
			TagID:    tok.Trigger,
			Expected: stacked,
		})

		state.ast.TextLength += tok.Width
		state.textNodes++
		processText(state, tok)
		return
	}

	// 3. Otherwise close the Tag
	closeTag(state, tok)
}
