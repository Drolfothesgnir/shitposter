package markup

// actLinkTextStart returns token TypeLinkTextStart if the SymbolLinkTextStart is not the last in the string,
// otherwise returns token TypeText and adds a Warning.
//
// Designed happy path first.
func actLinkTextStart(input string, i int, warns *[]Warning) (token Token, stride int) {
	// we've processed single byte anyway
	stride = 1

	// happy path
	// if the SymbolLinkTextStart is not the last in the string, return token TypeLinkTextStart
	if i+1 < len(input) {
		token = Token{
			Type: TypeLinkTextStart,
			Pos:  i,
			Len:  1,
			Val:  input[i : i+1],
		}

		return
	}

	// else return token TypeText and add a Warning
	token = Token{
		Type: TypeText,
		Pos:  i,
		Len:  1,
		Val:  input[i : i+1],
	}

	*warns = append(*warns, Warning{
		Node:        NodeText,
		Index:       i + 1,
		Issue:       IssueUnexpectedEOL,
		Description: "Unexpected end of the line: expected to get '" + input[i:i+1] + "', got EOL instead.",
	})

	return
}
