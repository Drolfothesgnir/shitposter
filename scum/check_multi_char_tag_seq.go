package scum

// CheckMultiCharTagSeq tells the ActionContext if the trigger Tag has no missing symbols.
func CheckMultiCharTagSeq(ctx *ActionContext) {
	i := ctx.Idx
	contained, _, l := ctx.Tag.Seq.IsContainedIn(ctx.Input[i : i+int(ctx.Tag.Seq.Len)])
	ctx.Bounds.SeqValid = contained
	ctx.Bounds.Width = l
}
