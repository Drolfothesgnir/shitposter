package markdown

import "unicode/utf8"

// actBoldOrItalic parses SymbolItalic case and returns token with type based on if the SymbolItalic is single, in
// which case the TypeItalic will be returned, or double, in which case TypeBold will be returned.
func actBoldOrItalic(input string, cur rune, width, i int) (token Token, warnings []Warning, stride int, ok bool) {

	// actBoldOrItalic will return token in any case
	ok = true

	// if the rune is last, or the next rune is not SymbolItalic, the token is considered TypeItalic
	t := TypeItalic

	// finalWidth defines total width of the deduced tag in bytes, that is
	// just 'width' if the SymbolItalic is single and 'width' + width of the next
	// symbol in case the symbol is doubled and considered a bold tag
	finalWidth := width

	isLastRune := i+width == len(input)

	if !isLastRune {
		next, nextWidth := utf8.DecodeRuneInString(input[i+width:])

		// case when the next symbol is also SymbolItalic
		if Symbol(next) == SymbolItalic {
			finalWidth = width + nextWidth
			t = TypeBold
		}
	}

	token = Token{
		Type: t,
		Pos:  i,
		Len:  finalWidth,
		Val:  input[i : i+finalWidth],
	}

	// explicitely telling the number of proccessed bytes
	stride = finalWidth

	return
}
