package scum

// TODO: document steps for Myself
func Parse(input string, d *Dictionary, warns *Warnings) AST {
	out := Tokenize(d, input, warns)

	state := newParserState(input, out)

	for _, t := range out.Tokens {
		switch t.Type {
		case TokenText:
			processText(&state, t)

		case TokenAttributeFlag, TokenAttributeKV:
			processAttribute(&state, t)

		case TokenTag:
			processTag(&state, d, warns, t)
		}
	}

	for len(state.breadcrumbs) > 1 {
		idx := state.popCrumb()
		childWidth := state.popCumWidth(0) // no closing tag
		// Use assignment, not +=, because childWidth already includes opening tag width
		state.ast.Nodes[idx].Span.End = state.ast.Nodes[idx].Span.Start + childWidth

		openTagID := state.popStack()
		state.openedTags[openTagID] = false

		warns.Add(Warning{
			Issue:      IssueUnclosedTag,
			Pos:        state.ast.Nodes[idx].Span.Start,
			TagID:      openTagID,
			CloseTagID: d.tags[openTagID].CloseID,
		})
	}

	// then finalize root
	state.ast.Nodes[0].Span.End = state.peekCumWidth()

	state.ast.MaxDepth = state.maxDepth

	state.ast.TextLength += out.TextLen

	state.ast.TotalTextNodes = state.textNodes + out.TextTokens

	state.ast.TotalTagNodes = state.totalTagNodes

	return state.ast
}
