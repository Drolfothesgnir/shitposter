package scum

// StepSkipOpenTag used when the opening Tag is malformed or unclosed, for the Tokenizer
// to treat it as a plain text.
func StepSkipOpenTag(ctx *ActionContext) bool {
	// we explicitely tell that no Token will be returned
	ctx.Token = Token{}

	// we also tell that we only processed the available opening sequence
	ctx.Stride = ctx.Bounds.OpenWidth

	ctx.Skip = true
	return true
}
