package scum

// Span defines bounds of the window view of a string.
type Span struct {
	// Start defines the inclusive start of the view.
	Start int

	// End defines the exclusive end of the view.
	End int
}

// NewSpan creates new Span from the startIdx and the width.
// End index is calculated as startIdx + width.
func NewSpan(startIdx int, width int) Span {
	return Span{startIdx, startIdx + width}
}

// Token is the result of the first stage processing of a part of the input string.
// It contains metadata and value of the processed sequence of bytes.
type Token struct {
	// Type defines the type of the Token, e.g. opening, closing, or universal tag, or an escape sequence.
	Type TokenType

	// Trigger is the leading special byte that started this Token.
	// It is always a 1-byte printable ASCII character from the input that matched a registered Action.
	//
	// Examples:
	//   - for a Tag token, Trigger is the first byte of the tag sequence (its ID).
	//   - for an EscapeSequence token, Trigger is the escape symbol.
	//   - for an Attribute token, Trigger is the attribute signature symbol.
	Trigger byte

	// Pos defines the starting byte position of the tag's sequence in the input string.
	// Usually is the same as Raw.Start value.
	Pos int

	// Width defines count of bytes in the Tag's sequence, including the Tag itself and the payload.
	//
	// Example: Imagine, you have defined a universal tag with name 'BOLD' and a byte sequence of "$$". The sequence has
	// 2 bytes in it, 1 per each '$', so the corresponding token will have width of 2.
	Width int

	// Payload defines the bounds of the Token's main semantic content within the input string.
	//
	// The meaning depends on Token type:
	//   - Tag tokens: Payload spans the plain text content inside the tag (without opening/closing symbols).
	//   - Text tokens: Payload is equal to Raw.
	//   - EscapeSequence tokens: Payload spans the escaped UTF-8 code point.
	//   - AttributeKV tokens: Payload spans the attribute value.
	//   - AttributeFlag tokens: Payload is empty.
	Payload Span

	// AttrKey defines the bounds of the attribute name when Token type is AttributeKV or AttributeFlag.
	// For non-attribute tokens AttrKey is empty.
	AttrKey Span
}
