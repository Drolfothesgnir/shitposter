package scum

// WarnCloseTagAfterStart adds a [Warning] of misplaced closing [Tag] if the
// Tag is closing and found at the index 0 of the input.
func WarnCloseTagAfterStart(ctx *ActionContext) {
	if ctx.Idx != 0 || !ctx.Tag.IsClosing() {
		return
	}

	ctx.Warns.Add(Warning{
		Issue: IssueMisplacedClosingTag,
		Pos:   ctx.Idx,
		TagID: ctx.Tag.ID(),
	})
}
