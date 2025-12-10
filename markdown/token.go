package markdown

import (
	"strings"
)

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
	TypeEscape
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

// tagToType maps tag strings to their corresponding token types.
// This is useful for mapping characters to token types and avoiding
// large if-else / switch chains.
var tagToType = map[Tag]Type{
	TagBold:          TypeBold,
	TagStrikethrough: TypeStrikethrough,
	TagItalic:        TypeItalic,
	TagItalicAlt:     TypeItalic,
	TagCode:          TypeCode,
	TagLinkTextStart: TypeLinkTextStart,
	TagLinkTextEnd:   TypeLinkTextEnd,
	TagLinkURLStart:  TypeLinkURLStart,
	TagLinkURLEnd:    TypeLinkURLEnd,
	TagImageMarker:   TypeImageMarker,
	TagEscape:        TypeEscape,
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

// multiCharTagList stores tags whose string representation is longer than 1 byte.
// NOTE: This must be ordered by tag length in descending order if you ever add
// overlapping tags (e.g. "***" and "**") to ensure correct matching.
var multiCharTagList = []Tag{
	TagBold,
	TagStrikethrough,
}

// extractMultiCharTag checks if the provided string starts with any multi-character tag.
// It returns the matched tag, its string length, and ok = true if a match is found.
//
// Otherwise it returns zero values and ok = false.
func extractMultiCharTag(substr string) (tag Tag, length int, ok bool) {
	for _, t := range multiCharTagList {
		strTag := string(t)
		if strings.HasPrefix(substr, strTag) {
			tag = t
			length = len(strTag)
			ok = true
			break
		}
	}
	return
}

// charIsTag returns true if the character belongs to any known tag.
func charIsTag(char string) bool {
	_, found := tagToType[Tag(char)]
	return found
}

// tagStartRunes contains runes that can start a markdown tag (single- or multi-char).
var tagStartRunes = map[rune]struct{}{
	'*':  {},
	'_':  {},
	'~':  {},
	'`':  {},
	'[':  {},
	']':  {},
	'(':  {},
	')':  {},
	'!':  {},
	'\\': {},
}

func isTagStartRune(r rune) bool {
	_, ok := tagStartRunes[r]
	return ok
}

// findNextTagStart finds the byte offset of the next tag-like character
// inside substr. It returns the offset relative to substr, or -1 if no tag
// characters are found.
func findNextTagStart(substr string) int {
	for i, r := range substr {
		if isTagStartRune(r) {
			return i
		}
	}
	return -1
}

// Tokenize transforms a raw markdown string into a slice of tokens.
func Tokenize(s string) []*Token {
	result := make([]*Token, 0)

	// Current byte position in the input string.
	var p int

	n := len(s)

	for p < n {
		substr := s[p:]

		// 1) Check for multi-character tags (e.g. "**", "~~").
		if t, l, ok := extractMultiCharTag(substr); ok {
			token := newTag(p, t)
			result = append(result, token)

			// Move past the multi-character tag.
			p += l
			continue
		}

		// 2) Check for single-character tags.
		c := string(s[p])
		if ok := charIsTag(c); ok {
			token := newTag(p, Tag(c))
			result = append(result, token)
			p++
			continue
		}

		// 3) Otherwise, it's text. Consume as much text as possible
		// until the next tag-like character.

		var textEnd int
		nextTagRelativeIndex := findNextTagStart(substr)

		// If no tags are found, consume the rest of the string as text.
		if nextTagRelativeIndex == -1 {
			textEnd = n
		} else {
			textEnd = p + nextTagRelativeIndex
		}

		textValue := s[p:textEnd]
		token := newText(p, textValue)
		result = append(result, token)

		p = textEnd
	}

	return result
}

// newTag creates a new *Token for the given tag at byte position p.
func newTag(p int, tag Tag) *Token {
	t := tagToType[tag]
	s := string(tag)
	return &Token{
		Type: t,
		Pos:  p,
		Len:  len(s),
		Val:  s,
	}
}

// newText creates a new *Token of type TypeText at byte position p.
func newText(p int, val string) *Token {
	return &Token{
		Type: TypeText,
		Pos:  p,
		Len:  len(val),
		Val:  val,
	}
}
