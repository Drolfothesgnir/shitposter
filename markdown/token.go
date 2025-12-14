package markdown

import (
	"fmt"
	"unicode/utf8"
)

// import (
// 	"fmt"
// 	"strings"
// )

// Type defines the kind of token, e.g. bold, italic, link start, etc.
type Type int

const (
	TypeBold Type = iota
	TypeItalic
	TypeStrikethrough
	TypeCode
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
)

// ActionResult represents a result of a processing string by a particular action function.
type ActionResult struct {
	// Possible Token created by an action during the processing.
	Tok *Token

	// Possible list of issues occured during string proccessing,
	// e.g. redundant escape ('\'), or malformed tag, like '~' instead of '~~'.
	Warns []Warning

	// Number of bytes processed. Used for adjusting the main loop pointer.
	Stride int
}

// Token returns the token value from the action result and an indicator,
// which will be true if the token is available.
func (ar *ActionResult) Token() (Token, bool) {
	if ar.Tok != nil {
		return *ar.Tok, true
	}

	return Token{}, false
}

// Warings return list of warnings occured during the action and an indicator
// which will be true if the list is not empty.
func (ar *ActionResult) Warnings() ([]Warning, bool) {
	return ar.Warns, len(ar.Warns) > 0
}

// Action defines a function used to process the substring after the
// corresponding special symbol occured.
//
// Recieves rest of the string, starting from the occurance index - `substr`,
// index of the occurance in the original string - `i`
// and indicator if the occured rune was last in the sequence - `isLastRune`.
type Action func(substr string, cur rune, i int, isLastRune bool) ActionResult

// actStrikethrough processes rest of the string after first occurance of the
// Strikethrough rune, '~'.
func actStrikethrough(substr string, _ rune, i int, isLastRune bool) ActionResult {
	var res ActionResult

	symLen := utf8.RuneLen(rune(SymbolStrikethrough))

	// if the first '~' occured at the very end of the string ->
	// create a token with plain text and return a warning
	if isLastRune {
		res.Tok = &Token{
			Type: TypeText,
			Pos:  i,
			Len:  symLen,
			Val:  string(SymbolStrikethrough),
		}

		w := Warning{
			Node:        NodeText,
			Index:       i + symLen,
			Issue:       IssueUnexpectedEOL,
			Description: fmt.Sprintf("Unexpected end of the line: expected to get %q, got EOL instead.", SymbolStrikethrough),
		}

		// making stride equal to the byte length of the striketrhough symbol ('~') to
		// signal the main loop we effectively processed the last item
		// and the loop should terminate by violating the `i < n` condition.
		res.Stride = symLen

		res.Warns = append(res.Warns, w)
		return res
	}

	// checking next rune
	nextRune, _ := utf8.DecodeRuneInString(substr[symLen:])

	// if the next rune is not '~' -> make a first '~' as a plain text token, and
	// add a warning
	if Symbol(nextRune) != SymbolStrikethrough {
		res.Tok = &Token{
			Type: TypeText,
			Pos:  i,
			Len:  symLen,
			Val:  string(SymbolStrikethrough),
		}

		w := Warning{
			Node:        NodeText,
			Index:       i + symLen,
			Issue:       IssueUnexpectedSymbol,
			Description: fmt.Sprintf("Unexpected symbol near %q: expected %q, got %q", nextRune, SymbolStrikethrough, nextRune),
			Near:        string(nextRune),
		}

		res.Stride = symLen

		res.Warns = append(res.Warns, w)
		return res
	}
	// OK case, create token and return
	res.Tok = &Token{
		Type: TypeStrikethrough,
		Pos:  i,
		Len:  utf8.RuneLen(rune(SymbolStrikethrough)) * 2,
		Val:  string(SymbolStrikethrough) + string(SymbolStrikethrough),
	}

	res.Stride = symLen * 2

	return res
}

// actText is an oversimplified version of the text accumulator.
func actText(substr string, cur rune, i int, isLastRune bool) ActionResult {
	var res ActionResult

	// NOTE: complete until-next-special-symbol text search should be happening here
	// Don't think i went full retard

	textLen := utf8.RuneLen(cur)

	res.Tok = &Token{
		Type: TypeText,
		Pos:  i,
		Len:  textLen,
		Val:  string(cur),
	}

	res.Stride = textLen
	return res
}

// runeToAction is far from complete rune to its action mapper.
var runeToAction = map[rune]Action{
	'~': actStrikethrough,
}

func Tokenize(input string) (tokens []Token, warnigs []Warning) {

	n := len(input)

	for i, w := 0, 0; i < n; i += w {
		substr := input[i:]
		runeValue, width := utf8.DecodeRuneInString(substr)
		isLastRune := i+width == n

		// making default stride 'w' to equal the current rune's width
		w = width

		act, ok := runeToAction[runeValue]
		// if the rune doesn't have corresponding action, then it must be a plain text
		if !ok {
			act = actText
		}

		res := act(substr, runeValue, i, isLastRune)
		if token, ok := res.Token(); ok {
			tokens = append(tokens, token)
		}

		if warns, ok := res.Warnings(); ok {
			warnigs = append(warnigs, warns...)
		}

		// IMPORTANT: making sure we've accounted for all processed bytes
		if res.Stride > 0 {
			w = res.Stride
		}
	}

	return
}

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
