package scum

// StepSkipOpenTagBeforeEOL checks if the opening Tag is the last Tag in the
// input. If the Tag is the last one, the step handle the
// opening Tag, by skipping it.
func StepSkipOpenTagBeforeEOL(ctx *ActionContext) bool {
	n := len(ctx.Input)
	w := ctx.Bounds.Width
	// if the opening Tag is not the last sequence in the input
	// don't handle and continue to the next handler
	if ctx.Idx+w < n {
		return false
	}

	// else skip the Tag as a plain text

	ctx.Token = Token{}
	ctx.Stride = w
	ctx.Skip = true
	return true
}
