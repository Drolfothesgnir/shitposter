package scum

// Span defines bounds of the window view of a string.
type Span struct {
	Start, End int
}

// Token is the result of the first stage processing of a part of the input string.
// It contains metadata and value of the processed sequence of bytes.
type Token struct {
	// Type defines the type of the Token, e.g. opening, closing, or universal tag, or an escape sequence.
	Type TokenType

	// TagID a unique leading byte of the tag byte sequence, defined by the User.
	TagID byte

	// Pos defines the starting byte position of the tag's sequence in the input string.
	// Usually is the same as Raw.Start value.
	Pos int

	// Width defines count of bytes in the tag's sequence.
	//
	// Example: Imagine, you have defined a universal tag with name 'BOLD' and a byte sequence of "$$". The sequence has
	// 2 bytes in it, 1 per each '$', so the corresponding token will have width of 2.
	Width int

	// Raw defines the [Span] associated with the tag's value including both tag strings and the inner plain text.
	//
	// Example: Imagine, you have defined a greedy tag with name 'URL' and a pattern like this: "(...)", where
	// '(' is the opening tag and the ')' is the closing tag. When interpreting string "(https://some-address.com)",
	// the Raw will have [Span.Start] equal to 0 - the index of "(" and [Span.End] equal to 25 - the index of ")" , that is the bounds of
	// the entire input. For Text tokens Raw and Inner fields are the same.
	Raw Span

	// Inner defines the bounds of the plain text content, stripped of Tag symbols inside the [Tag].
	//
	// Example: Imagine, you have defined a greedy tag with name 'URL' and a pattern like this: "(...)", where
	// '(' is the opening tag and the ')' is the closing tag. When interpreting string "(https://some-address.com)",
	// the Inner field will have [Span.Start] equal to 1 - the index of "h" and [Span.End] - 24, the index of "m".
	Inner Span
}
