package markdown

import (
	"unicode/utf8"
)

// Token represents a single markdown token.
type Token struct {
	Type Type // Token type: bold, italic, text, etc.
	Pos  int  // Starting byte position of the token in the original markdown string.

	// Len is the length of the token's byte sequence.
	//
	// WARNING: This is not the visible text length.
	//
	// It is the byte length of the underlying string as used internally by the tokenizer.
	// Do not use this field for visible text length calculations; use rune counting instead.
	Len int

	// Val is the exact token string:
	// - for tags: the literal tag/delimiter ("**", "*", "[", etc.),
	// - for text: the raw text content.
	Val string
}

// Type defines the kind of token, e.g. bold, italic, link start, etc.
type Type int

const (
	TypeBold Type = iota
	TypeItalic
	TypeStrikethrough
	TypeCodeBlock
	TypeEscapeSequence
	TypeLinkTextStart
	TypeLinkTextEnd
	TypeLinkURLStart
	TypeLinkURLEnd
	TypeImageMarker
	TypeText
)

// Tag defines the string representation of markdown tags,
// e.g. "**" for bold, "*" / "_" for italic.
type Tag string

const (
	TagBold          Tag = "**"
	TagStrikethrough Tag = "~~"
	TagItalic        Tag = "*"
	TagItalicAlt     Tag = "_"
	TagCode          Tag = "`"
	TagLinkTextStart Tag = "["
	TagLinkTextEnd   Tag = "]"
	TagLinkURLStart  Tag = "("
	TagLinkURLEnd    Tag = ")"
	TagImageMarker   Tag = "!"
	TagEscape        Tag = "\\"
)

type Symbol rune

const (
	SymbolStrikethrough Symbol = '~'
	SymbolEscape        Symbol = '\\'
)

var specialSymbolMap = map[Symbol]struct{}{
	SymbolStrikethrough: {},
	SymbolEscape:        {},
	// ...other symbols
}

// isSpecialSymbolReturns true if the input rune is registerd in the
// specialSymbolMap.
func isSpecialSymbol(r rune) bool {
	s := Symbol(r)
	_, ok := specialSymbolMap[s]
	return ok
}

// runeToAction is far from complete rune to its action mapper.
var runeToAction = map[rune]action{
	'~':  actStrikethrough,
	'\\': actEscape,
}

// Tokenize processes the input string rune-wise and outputs a slice of Tokens and a slice of Warnings.
func Tokenize(input string) (tokens []Token, warnigs []Warning) {

	n := len(input)

	for i, w := 0, 0; i < n; i += w {

		// IMPORTANT: we looking at each rune instead of each byte and
		// incrementing the iterator variable by the actual byte length of the rune.
		runeValue, width := utf8.DecodeRuneInString(input[i:])
		isLastRune := i+width == n

		// making default stride 'w' to equal the current rune's width
		w = width

		act, ok := runeToAction[runeValue]
		// if the rune doesn't have corresponding action, then it must be a plain text
		if !ok {
			act = actText
		}

		substr := input[i:]

		// checking if the action returned some token.
		token, warnings, stride, ok := act(substr, runeValue, width, i, isLastRune)
		if ok {
			tokens = append(tokens, token)
		}

		// check for warnings
		if len(warnings) > 0 {
			warnigs = append(warnigs, warnings...)
		}

		// IMPORTANT: making sure we've accounted for all processed bytes
		if stride > 0 {
			w = stride
		}
	}

	return
}
