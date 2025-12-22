package markup

import (
	"strconv"
	"unicode/utf8"
)

// actEscape creates a [Token] [TypeEscapeSequence] with value of the [SymbolEscape] followed by the next character, or
// [TypeText] from only the escape char, and adds a [Warning] if the trigger symbol is the last in the string, or
// the escaped character is a plain text.
//
// Triggered by [SymbolEscape].
//
// Behaviour:
//
// If the trigger symbol is the last in the string, or the next character is no a special one, actEscape will return
// token [TypeText] with value of only the escape symbol and add a [Warning] with [IssueRedundantEscape].
func actEscape(input string, i int, warns *[]Warning) (token Token, stride int) {

	// 1. Checking the last symbol case

	// if the escape symbol is the last in line
	// return it as a plain text and add a Warning.
	if i+1 == len(input) {
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

	// 2. Creating the escape sequence

	// if the char is not the last symbol in the string
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

	// 3. Checking if the next char is a plain text

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
