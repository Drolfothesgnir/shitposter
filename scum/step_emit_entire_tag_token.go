package scum

func StepEmitEntireTagToken(ctx *ActionContext) bool {
	w := ctx.Bounds.Raw.End - ctx.Bounds.Raw.Start

	ctx.Token = Token{
		Type:  TokenTag,
		TagID: ctx.Tag.ID(),
		Pos:   ctx.Idx,
		Width: w,
		Raw:   ctx.Bounds.Raw,
		Inner: ctx.Bounds.Inner,
	}

	ctx.Stride = w
	ctx.Skip = false

	return true
}
