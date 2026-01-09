package scum

// StepSkipInvalidTriggerTag skips the Tag only if its symbol sequence is invalid, that is missing some symbols.
func StepSkipInvalidTriggerTag(ctx *ActionContext) bool {
	w := ctx.Bounds.Width

	if ctx.Bounds.SeqValid || w >= MaxTagLen {
		return false
	}

	ctx.Token = Token{}
	ctx.Stride = w
	ctx.Skip = true
	return true
}
