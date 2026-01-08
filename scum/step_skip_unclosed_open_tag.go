package scum

// StepSkipUnclosedOpenTag will do nothing if the [Tag] is closed.
func StepSkipUnclosedOpenTag(ctx *ActionContext) bool {
	if ctx.Bounds.Closed {
		return false
	}

	// we explicitely tell that no Token will be returned
	ctx.Token = Token{}

	// we also tell that we only processed the available opening sequence
	ctx.Stride = ctx.Bounds.OpenWidth

	ctx.Skip = true
	return true
}
