package scum

func processAttribute(state *parserState, tok Token) {
	attr := Attribute{
		Name:    tok.AttrKey,
		Payload: tok.Payload,
		IsFlag:  tok.Type == TokenAttributeFlag,
	}

	appendAttribute(&state.ast, state.lastNodeIdx, attr)
}
