package scum

// StepCloseTagAfterStart skips the Tag as plain text if it's
// closed and placed at the index 0 of the input.
func StepCloseTagAfterStart(ctx *ActionContext) bool {
	if ctx.Idx != 0 || !ctx.Tag.IsClosing() {
		return false
	}

	ctx.Token = Token{}
	ctx.Stride = ctx.Bounds.CloseWidth
	ctx.Skip = true
	return true
}
