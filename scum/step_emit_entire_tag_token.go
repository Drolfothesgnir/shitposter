package scum

func StepEmitEntireTagToken(ctx *ActionContext) bool {
	w := ctx.Bounds.Raw.End - ctx.Bounds.Raw.Start

	ctx.Token = Token{
		Type:    TokenTag,
		Trigger: ctx.Tag.ID(),
		Pos:     ctx.Idx,
		Width:   w,
		Raw:     ctx.Bounds.Raw,
		Payload: ctx.Bounds.Inner,
	}

	ctx.Stride = w
	ctx.Skip = false

	return true
}
