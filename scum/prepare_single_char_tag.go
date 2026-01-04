package scum

// PrepareSingleCharTag sets bounds for the single byte Tag.
func PrepareSingleCharTag(ctx *ActionContext) {

	// if the Tag is either universal or opening, set its OpenWidth
	if !ctx.Tag.IsClosing() {
		ctx.Bounds.OpenWidth = 1
	} else {
		// otherwise, set the closing Tag position and CloseWidth
		ctx.Bounds.CloseIdx = ctx.Idx
		ctx.Bounds.CloseWidth = 1
	}
	ctx.Bounds.Raw = NewSpan(ctx.Idx, 1)
	ctx.Bounds.Inner = NewSpan(ctx.Idx+1, 0)
}
