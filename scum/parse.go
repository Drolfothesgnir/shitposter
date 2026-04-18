package scum

import "sync"

// statePool keeps parser state memory available between parses and reduces GC load.
var statePool = sync.Pool{
	New: func() any {
		return newParserState()
	},
}

// Parse tokenizes the input and builds an [AST] in a single pass.
//
// Parse allocates an AST for the result. Use [ParseInto] when the caller owns
// an AST and wants to reuse its backing storage across parses.
// d and warns must be non-nil.
func Parse(input string, d *Dictionary, warns *Warnings) AST {
	var ast AST
	ParseInto(&ast, input, d, warns)
	return ast
}

// ParseInto tokenizes the input and builds an [AST] in dst.
//
// dst owns its node and attribute backing arrays. ParseInto reuses those arrays
// when their capacity is suitable for the current input and configured
// [Limits]. The AST stored in dst remains valid until the caller resets or
// reuses dst.
//
// dst, d and warns must be non-nil.
//
// Processing happens in three phases:
//  1. Tokenize: the input string is split into text, tag, and attribute tokens.
//  2. Tree construction: tokens are consumed left-to-right. Text tokens become
//     [NodeText] children of the current parent. Tag tokens open or close
//     nesting levels according to the registered Tag definitions. Attribute
//     tokens are attached to the most recently created or closed node.
//  3. Finalization: any tags still open at end-of-input are force-closed from
//     innermost to outermost (with [IssueUnclosedTag] warnings), the root
//     node's span is set, and AST-level statistics are populated.
func ParseInto(dst *AST, input string, d *Dictionary, warns *Warnings) {
	// Phase 1: tokenize
	out := Tokenize(d, input, warns)

	resetAST(dst, input, out, d.Limits)

	state := statePool.Get().(*parserState)

	state.ast = *dst
	state.limits = d.Limits

	// Phase 2: build the tree by dispatching each token
	for _, t := range out.Tokens {
		switch t.Type {
		case TokenText:
			processText(state, t, warns)

		case TokenAttributeFlag, TokenAttributeKV:
			processAttribute(state, t, warns)

		case TokenTag:
			processTag(state, d, warns, t)
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
	// TokenizerOutput.TextByteLen is a byte count, so AST.TextByteLen remains byte-based.
	state.ast.TextByteLen += out.TextByteLen
	state.ast.TotalTextNodes = state.textNodes + out.TextTokens
	state.ast.TotalTagNodes = state.totalTagNodes

	*dst = state.ast

	state.Reset()

	statePool.Put(state)
}

func resetAST(dst *AST, input string, out TokenizerOutput, limits Limits) {
	nodeCap := max(out.TagsTotal+out.TextTokens, 1)
	if limits.MaxNodes > 0 {
		nodeCap = min(nodeCap, limits.MaxNodes)
		nodeCap = max(nodeCap, 1)
	}

	nodes := dst.Nodes
	// A full-slice expression could reduce the visible capacity, but it would
	// still retain the old backing array. When an explicit limit is lower than
	// the reused AST's capacity, allocate so the oversized buffer can be freed.
	if cap(nodes) < nodeCap || (limits.MaxNodes > 0 && cap(nodes) > limits.MaxNodes) {
		nodes = make([]Node, 1, nodeCap)
	} else {
		nodes = nodes[:1]
	}
	nodes[0] = NewNode()

	attrCap := out.Attributes
	if limits.MaxAttributes > 0 {
		attrCap = min(attrCap, limits.MaxAttributes)
	}

	attrs := dst.Attributes
	// Same as nodes: if the caller lowered the explicit attribute limit, do not
	// keep an older backing array that is larger than the configured cap.
	if cap(attrs) < attrCap || (limits.MaxAttributes > 0 && cap(attrs) > limits.MaxAttributes) {
		attrs = make([]Attribute, 0, attrCap)
	} else {
		attrs = attrs[:0]
	}

	*dst = AST{
		Input:      input,
		Nodes:      nodes,
		Attributes: attrs,
	}
}
