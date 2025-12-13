package markdown

import (
	"fmt"
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
// TODO: refactor it with map
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
func Tokenize(s string) (result []*Token, warnings []*Warning) {

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

		consumed, step, warns := consumeUntilNextTag(substr, p)

		// if some bytes was consumed as text
		if step > 0 {
			token := newText(p, consumed)
			result = append(result, token)
			warnings = append(warnings, warns...)
			p += step
		} else {
			p++
		}
	}

	return
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

// consumeUntilNextTag iterates through the string and acumulates all non-tag bytes,
// including escaped characters, until the tag-start rune is found.
// Accepts substring and original string's current position index.
// Will create a warning if redundant escape is found.
//
// Returns accumulated string, number of processed bytes and possible Warnings.
func consumeUntilNextTag(substr string, i int) (consumed string, step int, warnings []*Warning) {

	var b strings.Builder

	n := len(substr)

	for step < n {
		cur := substr[step]

		// 1) check if the current byte is not a tag of any type
		// and write it to the builder if true
		if !isTagStartRune(rune(cur)) {
			b.WriteByte(cur)
			step++
			continue
		}

		// 2) if current byte is a tag but not an escape - return
		if isTagStartRune(rune(cur)) && Tag(cur) != TagEscape {
			consumed = b.String()
			return
		}

		// 3) if the tag is an escape there are 3 possible cases
		if Tag(cur) == TagEscape {
			// 3.1) escape as a last byte in the substring - add Warning
			if step+1 >= n {
				// TODO: create *Warning factory
				w := &Warning{
					Node:        NodeText, // since escape errors can occur only in plain text
					Index:       i + step,
					Issue:       IssueRedundantEscape,
					Description: fmt.Sprintf("Escape \"%s\" at the very end of the input string. It is a no-op and will be ignored.", TagEscape),
				}

				warnings = append(warnings, w)

				// 3.2) escaped character represents a tag - write it to the builder and continue
			} else if nextByte := substr[step+1]; isTagStartRune(rune(nextByte)) {
				b.WriteByte(nextByte)
				// moving to the next byte to account for it to be used already
				step++

				// 3.3) escaped character represents plain text - write it to the builder, add a Warning and continue
			} else {
				// nextByte := substr[step+1]
				b.WriteByte(nextByte)
				near := string(nextByte)

				w := &Warning{
					Node:        NodeText, // since escape errors can occur only in plain text
					Near:        near,
					Index:       i + step,
					Issue:       IssueRedundantEscape,
					Description: fmt.Sprintf("Escape \"%s\" before plain text at byte index %d near \"%s\"", TagEscape, i+step, near),
				}
				warnings = append(warnings, w)

				// moving to the next byte to account for it to be used already
				step++
			}

			// move forward anyway
			step++
		}
	}

	consumed = b.String()
	return
}
