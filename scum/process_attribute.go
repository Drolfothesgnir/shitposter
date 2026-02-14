package scum

// processAttribute converts an attribute token (key-value or flag) into an
// [Attribute] and appends it to the most recently created or closed node
// (state.lastNodeIdx).
func processAttribute(state *parserState, tok Token) {
	attr := Attribute{
		Name:    tok.AttrKey,
		Payload: tok.Payload,
		IsFlag:  tok.Type == TokenAttributeFlag,
	}

	appendAttribute(&state.ast, state.lastNodeIdx, attr)
}
