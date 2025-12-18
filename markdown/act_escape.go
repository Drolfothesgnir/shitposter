package markdown

import (
	"fmt"
	"unicode/utf8"
)

// actEscape proccesses the next rune after the escape symbol '\' and returns either
// text or escape sequence tokens.
func actEscape(substr string, cur rune, width int, i int, isLastRune bool) (token Token, warnings []Warning, stride int, ok bool) {

	// actEscape returns token anyway so ok = true
	ok = true

	// if the escape symbol is the last in line
	// return it as a plain text and add a Warning.
	if isLastRune {
		token = Token{
			Type: TypeText,
			Pos:  i,
			Len:  width,
			Val:  string(cur),
		}

		warnings = []Warning{{
			Node:        NodeText,
			Index:       i,
			Issue:       IssueRedundantEscape,
			Description: fmt.Sprintf("Redundant escape symbol %q at the end of the string.", cur),
		}}

		// signaling the main loop that we haven't processed any new runes
		stride = width

		return
	}

	// getting the next rune
	next, w := utf8.DecodeRuneInString(substr[width:])

	sequence := substr[:width+w]

	nextIndex := width + i

	token = Token{
		Type: TypeEscapeSequence,
		Pos:  i,
		Len:  width + w,
		Val:  sequence,
	}

	// if the next rune is not a special symbol but is a plain text instead
	// also add warning
	if !isSpecialSymbol(next) {
		warnings = []Warning{{
			Node:        NodeText,
			Index:       nextIndex,
			Near:        sequence,
			Issue:       IssueRedundantEscape,
			Description: fmt.Sprintf("Redundant escape before the character %q at byte index %d", next, nextIndex),
		}}
	}

	// signalling the main loop that we've proccessed escape and the next rune
	stride = width + w

	return
}
