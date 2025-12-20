package markdown

import (
	"fmt"
	"unicode/utf8"
)

// actStrikethrough processes rest of the string starting from the next rune after first occurance of the
// Strikethrough rune, '~'.
//
// WARNING: actStrikeThrough assumes SymbolStrikethrough is 1-byte long ASCII character.
func actStrikethrough(input string, i int, warns *[]Warning) (token Token, stride int, ok bool) {

	// actStrikethrough returns a token anyway so 'ok' is always true
	ok = true

	// if the first '~' occured at the very end of the string ->
	// create a token with plain text and return a warning
	if i+1 == len(input) {
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
			Description: fmt.Sprintf("Unexpected end of the line: expected to get %q, got EOL instead.", SymbolStrikethrough),
		})

		// explicitely signal the main loop that we have proccessed only the original symbol.
		stride = 1

		return
	}

	rest := input[i+1:]

	// checking next rune
	nextRune, nextRuneWidth := utf8.DecodeRuneInString(rest)

	// if the next rune is not '~' -> make a first '~' as a plain text token, and
	// add a warning
	if Symbol(nextRune) != SymbolStrikethrough {
		token = Token{
			Type: TypeText,
			Pos:  i,
			Len:  1,
			Val:  input[i : i+1],
		}

		*warns = append(*warns, Warning{
			Node:  NodeText,
			Index: i + 1,
			Issue: IssueUnexpectedSymbol,
			Description: fmt.Sprintf(
				"Unexpected symbol: expected second %q to form %s, got %q",
				SymbolStrikethrough,
				TagStrikethrough,
				nextRune,
			),
			Near: rest[:nextRuneWidth],
		})

		// explicitely signal the main loop that we have proccessed only the original symbol.
		stride = 1

		return
	}
	// OK case, create token and return
	token = Token{
		Type: TypeStrikethrough,
		Pos:  i,
		Len:  nextRuneWidth + 1,
		Val:  input[i : i+1+nextRuneWidth],
	}

	stride = nextRuneWidth + 1

	return
}
