package markdown

import (
	"strconv"
	"unicode/utf8"
)

// actEscape proccesses the next rune after the escape symbol '\' and returns either
// text or escape sequence tokens.
//
// Designed happy path first.
//
// WARNING: actEscape assumes that SymbolEscape is 1-byte long ASCII character.
func actEscape(input string, i int, warns *[]Warning) (token Token, stride int) {

	// happy case
	// if the char is not the last symbol in the string
	if i+1 < len(input) {
		// getting the next rune
		width := 1

		next := input[i+1]
		// if the next byte is not a simple ASCII, decode the next rune
		if next >= 128 {
			_, width = utf8.DecodeRuneInString(input[i+1:])
		}

		sequence := input[i : i+1+width]

		nextIndex := i + 1

		token = Token{
			Type: TypeEscapeSequence,
			Pos:  i,
			Len:  width + 1,
			Val:  sequence,
		}

		// if the next byte is not a special symbol but is a plain text instead
		// also add warning
		if symToAction[input[i+1]] == nil {
			desc := "Redundant escape before the character '" +
				input[nextIndex:nextIndex+width] + "' at byte index " + strconv.Itoa(nextIndex) + "."

			*warns = append(*warns, Warning{
				Node:        NodeText,
				Index:       nextIndex,
				Near:        sequence,
				Issue:       IssueRedundantEscape,
				Description: desc,
			})
		}

		// signalling the main loop that we've proccessed escape and the next rune
		stride = width + 1

		return
	}

	// if the escape symbol is the last in line
	// return it as a plain text and add a Warning.
	token = Token{
		Type: TypeText,
		Pos:  i,
		Len:  1,
		Val:  input[i:],
	}

	desc := "Redundant escape symbol '" + input[i:i+1] + "' at the end of the string."

	*warns = append(*warns, Warning{
		Node:        NodeText,
		Index:       i,
		Issue:       IssueRedundantEscape,
		Description: desc,
	})

	// signaling the main loop that we have processed only 1 byte
	stride = 1

	return

}
