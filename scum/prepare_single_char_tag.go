package scum

// PrepareSingleCharTag sets bounds for the single byte Tag.
func PrepareSingleCharTag(ctx *ActionContext) {
	ctx.Bounds.Width = 1
	ctx.Bounds.Raw = NewSpan(ctx.Idx, 1)
	ctx.Bounds.Inner = NewSpan(ctx.Idx+1, 0)
	// since single char Tags have valid sequence by default
	ctx.Bounds.SeqValid = true
}
