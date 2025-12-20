package markdown

import (
	"fmt"
	"unicode/utf8"
)

// actEscape proccesses the next rune after the escape symbol '\' and returns either
// text or escape sequence tokens.
//
// WARNING: actEscape assumes that SymbolEscape is 1-byte long ASCII character.
func actEscape(input string, i int, warns *[]Warning) (token Token, stride int, ok bool) {

	// actEscape returns token anyway so ok = true
	ok = true

	// if the escape symbol is the last in line
	// return it as a plain text and add a Warning.
	if i+1 == len(input) {
		token = Token{
			Type: TypeText,
			Pos:  i,
			Len:  1,
			Val:  input[i:],
		}

		*warns = append(*warns, Warning{
			Node:        NodeText,
			Index:       i,
			Issue:       IssueRedundantEscape,
			Description: fmt.Sprintf("Redundant escape symbol %q at the end of the string.", input[i]),
		})

		// signaling the main loop that we have processed only 1 byte
		stride = 1

		return
	}

	// getting the next rune
	next, w := utf8.DecodeRuneInString(input[i+1:])

	sequence := input[i : i+1+w]

	nextIndex := i + 1

	token = Token{
		Type: TypeEscapeSequence,
		Pos:  i,
		Len:  w + 1,
		Val:  sequence,
	}

	// if the next byte is not a special symbol but is a plain text instead
	// also add warning
	if symToAction[input[i+1]] == nil {
		*warns = append(*warns, Warning{
			Node:        NodeText,
			Index:       nextIndex,
			Near:        sequence,
			Issue:       IssueRedundantEscape,
			Description: fmt.Sprintf("Redundant escape before the character %q at byte index %d", next, nextIndex),
		})
	}

	// signalling the main loop that we've proccessed escape and the next rune
	stride = w + 1

	return
}
