package scum

// StepEmitSingleCharUniversalToken simply returns single-char [Token] from the current char in the input.
func StepEmitSingleCharUniversalToken(ctx *ActionContext) bool {
	ctx.Token = Token{
		Type:  TokenTag,
		TagID: ctx.Tag.ID,
		Pos:   ctx.Idx,
		Width: 1,
		Raw:   NewSpan(ctx.Idx, 1),
		Inner: NewSpan(ctx.Idx+1, 0),
	}

	ctx.Stride = 1
	ctx.Skip = false

	return true
}
