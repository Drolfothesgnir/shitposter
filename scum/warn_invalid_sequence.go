package scum

import "strconv"

func WarnInvalidSequence(ctx *ActionContext) {
	w := ctx.Bounds.Width
	i := ctx.Idx

	if ctx.Bounds.SeqValid || w >= MaxTagLen {
		return
	}

	expected := ctx.Tag.Seq.Bytes[w]
	got := ctx.Input[i+w]

	desc := "unexpected symbol while processing the Tag with name " +
		strconv.Quote(ctx.Tag.Name) +
		": expected to get " + strconv.QuoteRune(rune(expected)) + ", bot got " +
		strconv.QuoteRune(rune(got)) + "."

	ctx.Warns.Add(Warning{
		Issue:       IssueUnexpectedSymbol,
		Pos:         i + w,
		Description: desc,
	})
}
