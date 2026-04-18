package scum

// StepEmitEntireTagToken emits the current Tag bounds as a [TokenTag].
func StepEmitEntireTagToken(ctx *ActionContext) bool {
	w := ctx.Bounds.Raw.End - ctx.Bounds.Raw.Start

	ctx.Token = Token{
		Type:    TokenTag,
		Trigger: ctx.Tag.ID(),
		Pos:     ctx.Idx,
		Width:   w,
		Payload: ctx.Bounds.Inner,
	}

	ctx.Stride = w
	ctx.Skip = false

	return true
}
