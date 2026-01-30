package scum

// WarnTagKeyTooLong adds a Warnig if the Tag's opening/closing sequence is too long.
func WarnTagKeyTooLong(ctx *ActionContext) {
	if !ctx.Bounds.KeyLenLimitReached {
		return
	}

	ctx.Warns.Add(Warning{
		Issue: IssueTagKeyTooLong,
		Pos:   ctx.Idx,
	})
}
