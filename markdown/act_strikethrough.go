package markdown

import (
	"unicode/utf8"
)

// actStrikethrough processes rest of the string starting from the next rune after first occurance of the
// Strikethrough rune, '~'.
//
// Designed happy path first.
//
// WARNING: actStrikeThrough assumes SymbolStrikethrough is 1-byte long ASCII character.
func actStrikethrough(input string, i int, warns *[]Warning) (token Token, stride int) {

	n := len(input)

	// happy path
	// if the current symbol is not the last and the next symbol is also SymbolStrikethrough
	// return token
	if i+1 < n && Symbol(input[i+1]) == SymbolStrikethrough {
		token = Token{
			Type: TypeStrikethrough,
			Pos:  i,
			Len:  2,
			Val:  input[i : i+2],
		}

		stride = 2

		return
	}

	// next the last char case
	// if the first '~' occured at the very end of the string ->
	// create a token with plain text and return a warning
	if i+1 == n {
		token = Token{
			Type: TypeText,
			Pos:  i,
			Len:  1,
			Val:  input[i : i+1],
		}

		desc := "Unexpected end of the line: expected to get '" + input[i:i+1] + "', got EOL instead."

		*warns = append(*warns, Warning{
			Node:        NodeText,
			Index:       i + 1,
			Issue:       IssueUnexpectedEOL,
			Description: desc,
		})

		// explicitely signal the main loop that we have proccessed only the original symbol.
		stride = 1

		return
	}

	next := input[i+1]

	// the last case: unexpected symbol
	// if the next rune is not '~' -> make a first '~' as a plain text token, and
	// add a warning

	// checking next rune
	width := 1

	// check if the next byte is a multi-byte char
	if next >= 128 {
		_, width = utf8.DecodeRuneInString(input[i+1:])
	}

	token = Token{
		Type: TypeText,
		Pos:  i,
		Len:  1,
		Val:  input[i : i+1],
	}

	desc := "Unexpected symbol: expected second '" + input[i:i+1] + "', got '" + input[i+1:i+1+width] + "'."

	*warns = append(*warns, Warning{
		Node:        NodeText,
		Index:       i + 1,
		Issue:       IssueUnexpectedSymbol,
		Description: desc,
		Near:        input[i : i+1+width],
	})

	// explicitely signal the main loop that we have proccessed only the original symbol.
	stride = 1

	return

}
