package scum

import (
	"unicode"
	"unicode/utf8"
)

// StepInfraWordCheck checks if the current char is a real [Tag], according to the [RuleInfraWord].
// If the char is a real Tag, it returns false and allows the next steps to handle the Tag properly.
// If the char is considered a plain text, it returns true and sets Skip to true.
func StepInfraWordCheck(ctx *ActionContext) bool {
	leftIsWordPart := false

	i := ctx.Idx

	// if the current index is not at the beginning of the string
	if i > 0 {
		// extracting previous byte
		b := ctx.Input[i-1]

		// if the previous byte is ASCII char, check the byte directly
		if b < 128 {
			leftIsWordPart = isASCIIAlphanum(b) || isASCIIPunct(b) || b == ctx.Tag.ID
		} else {
			// else, decode the previous UTF-8 code point
			prev, _ := utf8.DecodeLastRuneInString(ctx.Input[:i])

			leftIsWordPart = unicode.IsLetter(prev) || // is a letter or
				unicode.IsNumber(prev) || // a number or
				unicode.IsPunct(prev) // a punctuation
		}
	}

	rightIsWordPart := false

	// if the char is not the last in the input
	if i+1 < len(ctx.Input) {
		// extracting the next byte
		b := ctx.Input[i+1]

		// if the next byte is ASCII char, check the byte directly
		if b < 128 {
			rightIsWordPart = isASCIIAlphanum(b) || isASCIIPunct(b) || b == ctx.Tag.ID
		} else {
			// else, decode the next UTF-8 code point
			next, _ := utf8.DecodeRuneInString(ctx.Input[i+1:])

			rightIsWordPart = unicode.IsLetter(next) || // is a letter or
				unicode.IsNumber(next) || // a number or
				unicode.IsPunct(next) // a punctuation
		}
	}

	// if the char is actually a Tag, return false, delegating the Token creation to the next steps
	if !leftIsWordPart || !rightIsWordPart {
		return false
	}

	// else return true, set Skip to true and Stride to 1
	ctx.Stride = 1
	ctx.Skip = true
	return true
}
