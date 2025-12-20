package markdown

import (
	"unicode/utf8"
)

// actImageMarker checks if the next character after the SymbolImageMarker is equal to the SymbolLinkTextStart and
// returns token TypeImageTextStart both symbols included, if true, or a TypeText and a Warning, based
// on the next symbol.
//
// Designed happy path first.
func actImageMarker(input string, i int, warns *[]Warning) (token Token, stride int) {

	n := len(input)

	// happy path
	if i+1 < n && Symbol(input[i+1]) == SymbolLinkTextStart {
		token = Token{
			Type: TypeImageTextStart,
			Pos:  i,
			Len:  2,
			Val:  input[i : i+2],
		}

		stride = 2
		return
	}

	// if we are here then either the symbol is the last in the string or
	// the next symbol is not a SymbolLinkTextStart

	// in any case we return token TypeText with current symbol
	token = Token{
		Type: TypeText,
		Pos:  i,
		Len:  1,
		Val:  input[i : i+1],
	}

	stride = 1

	// forming the warning
	var (
		issue Issue
		desc  string
		near  string
	)

	// the last symbol case
	if i+1 == n {
		issue = IssueUnexpectedEOL

		desc = "Unexpected end of the line: expected to get '" + TagLinkTextStart + "', got EOL instead."
	} else {
		// the next symbol case

		next := input[i+1]

		// r used in the warning to describe what char the user typed instead of the SymbolLinkTextStart
		var r rune

		// if the next symbol is an ASCII char, then we assign r to the next byte
		if next < 128 {
			r = rune(next)
		} else {
			r, _ = utf8.DecodeRuneInString(input[i+1:])
		}

		issue = IssueUnexpectedSymbol
		desc = "Unexpected symbol after '" + string(SymbolImageMarker) +
			"': expected to get '" + string(SymbolLinkTextStart) +
			"', got '" + string(r) + "'."

		near = input[i : i+1]
	}

	*warns = append(*warns, Warning{
		Node:        NodeText,
		Index:       i,
		Issue:       issue,
		Description: desc,
		Near:        near,
	})

	return
}
