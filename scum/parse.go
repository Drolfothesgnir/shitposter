package scum

// Parse tokenizes the input and builds an [AST] in a single pass.
//
// Processing happens in three phases:
//  1. Tokenize: the input string is split into text, tag, and attribute tokens.
//  2. Tree construction: tokens are consumed left-to-right. Text tokens become
//     [NodeText] children of the current parent. Tag tokens open or close
//     nesting levels (see [processTag]). Attribute tokens are attached to the
//     most recently created or closed node.
//  3. Finalization: any tags still open at end-of-input are force-closed from
//     innermost to outermost (with [IssueUnclosedTag] warnings), the root
//     node's span is set, and AST-level statistics are populated.
func Parse(input string, d *Dictionary, warns *Warnings) AST {
	// Phase 1: tokenize
	out := Tokenize(d, input, warns)

	state := newParserState(input, out)

	// Phase 2: build the tree by dispatching each token
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

	// Phase 3: force-close any remaining open tags (innermost first)
	for len(state.breadcrumbs) > 1 {
		idx := state.popCrumb()
		childWidth := state.popCumWidth(0) // 0 because there is no closing tag
		// childWidth already includes the opening tag width, so assign directly
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

	// Finalize root span and collect statistics
	state.ast.Nodes[0].Span.End = state.peekCumWidth()
	state.ast.MaxDepth = state.maxDepth
	state.ast.TextLength += out.TextLen
	state.ast.TotalTextNodes = state.textNodes + out.TextTokens
	state.ast.TotalTagNodes = state.totalTagNodes

	return state.ast
}
