package scum

// WarnInvalidSequence adds [IssueUnexpectedSymbol] when a multi-char Tag
// sequence is interrupted before it is complete.
func WarnInvalidSequence(ctx *ActionContext) {
	w := ctx.Bounds.Width
	i := ctx.Idx

	if ctx.Bounds.SeqValid || w >= MaxTagLen {
		return
	}

	expected := ctx.Tag.Seq.Bytes[w]
	// have to decrement w by 1 to account for the first symbol already being counted
	got := ctx.Input[i+w-1]

	ctx.Warns.Add(Warning{
		Issue:    IssueUnexpectedSymbol,
		Pos:      i + w,
		TagID:    ctx.Tag.ID(),
		Expected: expected,
		Got:      got,
	})
}
