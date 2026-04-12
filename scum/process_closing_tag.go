package scum

// processClosingTag handles a token that can only close a tag (e.g. ]).
//
// Three outcomes are possible:
//  1. No tag is open (stack empty): emit [IssueMisplacedClosingTag] and discard
//     the token entirely.
//  2. The top-of-stack tag doesn't match: emit [IssueOpenCloseTagMismatch] and
//     demote the closing token to a text node (incrementing textNodes so it is
//     reflected in [AST.TotalTextNodes]).
//  3. Match: delegate to [closeTag] to pop the stacks and finalize the span.
func processClosingTag(state *parserState, d *Dictionary, warns *Warnings, tok Token) {
	stacked := state.peekStack()

	tag := d.tags[tok.Trigger]

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

	if (openTag.CloseID != tok.Trigger) && (tag.OpenID != stacked) {
		warns.Add(Warning{
			Issue:    IssueOpenCloseTagMismatch,
			Pos:      tok.Pos,
			TagID:    tok.Trigger,
			Expected: stacked,
		})

		// Count the demoted closing tag as text bytes; Token.Width is byte-based.
		state.ast.TextByteLen += tok.Width
		state.textNodes++
		// processText uses Payload as the text node span. For a demoted tag, the
		// text is the raw tag bytes, not the tag payload.
		tok.Type = TokenText
		tok.Payload = NewSpan(tok.Pos, tok.Width)
		processText(state, tok)
		return
	}

	closeTag(state, tok)
}
