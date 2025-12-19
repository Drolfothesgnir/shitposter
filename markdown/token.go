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
	TypeCodeInline
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
	SymbolCode          Symbol = '`'
	SymbolItalic        Symbol = '*'
)

// isSpecialSymbolReturns true if the input rune is registerd in the
// specialSymbolMap.
func isSpecialSymbol(r rune) bool {
	switch Symbol(r) {
	case
		SymbolEscape,
		SymbolStrikethrough,
		SymbolCode:
		return true
	}

	return false
}

// Tokenize processes the input string rune-wise and outputs a slice of Tokens and a slice of Warnings.
func Tokenize(input string) (tokens []Token, warnings []Warning) {

	// guessing the token number to minimize the number of the slice resizes
	tokens = make([]Token, 0, len(input)/4)

	n := len(input)

	for i, w := 0, 0; i < n; i += w {

		// IMPORTANT: we looking at each rune instead of each byte and
		// incrementing the iterator variable by the actual byte length of the rune.
		runeValue, width := utf8.DecodeRuneInString(input[i:])
		isLastRune := i+width == n

		// making default stride 'w' to equal the current rune's width
		w = width

		act := actText

		// TODO: add behaviour docs to each action function
		switch Symbol(runeValue) {
		case SymbolStrikethrough:
			act = actStrikethrough
		case SymbolEscape:
			act = actEscape
		case SymbolCode:
			act = actCode
		case SymbolItalic:
			act = actBoldOrItalic
		}

		// checking if the action returned some token.
		token, warns, stride, ok := act(input[i:], runeValue, width, i, isLastRune)
		if ok {
			tokens = append(tokens, token)
		}

		// check for warnings
		if len(warns) > 0 {
			warnings = append(warnings, warns...)
		}

		// IMPORTANT: making sure we've accounted for all processed bytes
		if stride > 0 {
			w = stride
		}
	}

	return
}
