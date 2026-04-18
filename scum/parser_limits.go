package scum

func (s *parserState) warnLimitOnce(warns *Warnings, issue Issue, pos int) {
	switch issue {
	case IssueMaxNodesExceeded:
		if s.warnedMaxNodes {
			return
		}
		s.warnedMaxNodes = true

	case IssueMaxAttributesExceeded:
		if s.warnedMaxAttributes {
			return
		}
		s.warnedMaxAttributes = true

	case IssueMaxParseDepthExceeded:
		if s.warnedMaxParseDepth {
			return
		}
		s.warnedMaxParseDepth = true
	}

	warns.Add(Warning{
		Issue: issue,
		Pos:   pos,
	})
}

func appendStateNode(state *parserState, parentIdx int, node Node, pos int, warns *Warnings) (int, bool) {
	if state.limits.MaxNodes > 0 && len(state.ast.Nodes) >= state.limits.MaxNodes {
		state.warnLimitOnce(warns, IssueMaxNodesExceeded, pos)
		return -1, false
	}

	return appendNode(&state.ast, parentIdx, node), true
}

func canAppendAttribute(state *parserState, pos int, warns *Warnings) bool {
	if state.limits.MaxAttributes > 0 && len(state.ast.Attributes) >= state.limits.MaxAttributes {
		state.warnLimitOnce(warns, IssueMaxAttributesExceeded, pos)
		return false
	}

	return true
}

func canPushParseDepth(state *parserState, pos int, warns *Warnings) bool {
	if state.limits.MaxParseDepth > 0 && len(state.stack) >= state.limits.MaxParseDepth {
		state.warnLimitOnce(warns, IssueMaxParseDepthExceeded, pos)
		return false
	}

	return true
}
