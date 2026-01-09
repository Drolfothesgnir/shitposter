package scum

// CheckMultiCharTagSeq tells the ActionContext if the trigger Tag has no missing symbols.
func CheckMultiCharTagSeq(ctx *ActionContext) {
	i := ctx.Idx

	// we have to decrement by 1 since the ith char is already the first one in the seq
	expectedLastCharIdx := i + int(ctx.Tag.Seq.Len)
	seqEnd := min(len(ctx.Input), expectedLastCharIdx)

	contained, _, l := ctx.Tag.Seq.IsContainedIn(ctx.Input[i:seqEnd])
	ctx.Bounds.SeqValid = contained
	ctx.Bounds.Width = l
	ctx.Bounds.Raw = NewSpan(i, l)
	ctx.Bounds.Inner = NewSpan(i+l, 0)
}
