package markdown

import (
	"unicode"
	"unicode/utf8"
)

// isUnderlineTag accepts an input string, an index of the SymbolUnderline in it, length of the
// input and the previous rune to determine if the symbol defines an underline tag or a plain text.
//
// The rule is the following:
//
// if the symbol does not have alphanumeric character on either sides of it, it is considered a tag,
// otherwise, if the symbol is between two alphanums, then it's considered a plain text.
//
// Example: isUnderlineTag(" _hello", 1, 7, " ") will return true, but isUnderlineTag("hello_world", 5, 11, "o") will return false.
func isUnderlineTag(input string, i, n int, prevRune rune) bool {
	// checking the left side
	leftIsAlphanum := false

	// if the symbol is not the first one
	if i > 0 {

		// if the previous symbol is also an Underline, then the current one
		// is considered a plain text
		if Symbol(prevRune) == SymbolUnderline {
			return false
		}

		leftIsAlphanum = unicode.IsLetter(prevRune) || unicode.IsDigit(prevRune)
	}

	// checking the right side
	rightIsAlphanum := false

	// if the symbol is not the last in the string

	var next rune

	if i+1 < n {
		// if the next symbol is 1-byte long and is
		if input[i+1] < 128 {
			next = rune(input[i+1])

			if Symbol(next) == SymbolUnderline {
				return false
			}
		} else {
			next, _ = utf8.DecodeRuneInString(input[i+1:])
		}

		// if the next symbol is also an Underline, then the current one
		// is considered a plain text

		rightIsAlphanum = unicode.IsLetter(next) || unicode.IsDigit(next)
	}

	return !leftIsAlphanum || !rightIsAlphanum
}

// actUnderline returns current symbol in the input string as a Token TypeUnderline.
func actUnderline(input string, i int) (token Token, warnings []Warning, stride int, ok bool) {
	token = Token{
		Type: TypeUnderline,
		Pos:  i,
		Len:  1,
		Val:  input[i : i+1],
	}

	stride = 1

	ok = true
	return
}
