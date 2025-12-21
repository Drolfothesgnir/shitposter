package markdown

// actBoldOrItalic parses [SymbolItalic] case and returns token with type based on if the [SymbolItalic] is single, in
// which case the [TypeItalic] will be returned, or double, in which case [TypeBold] will be returned.
//
// Designed happy path first.
//
// WARNING: actBoldOrItalic assumes that [SymbolItalic] is 1-byte long.
// TODO: add behaviour docs for every action.
func actBoldOrItalic(input string, i int, _ *[]Warning) (token Token, stride int) {

	// if the rune is last, or the next rune is not [SymbolItalic], the token is considered [TypeItalic]
	t := TypeItalic

	// finalWidth defines total width of the deduced tag in bytes, that is
	// just 1 if the [SymbolItalic] is single and 2 in case the symbol is doubled and considered a bold tag
	finalWidth := 1

	// if the char is not a last in the string and the nect char is also [SymbolItalic]
	if i+1 < len(input) && Symbol(input[i+1]) == SymbolItalic {
		finalWidth = 2
		t = TypeBold
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
