package scum

// Token is the result of the first stage processing of a part of the input string.
// It contains metadata and value of the processed sequence of bytes.
type Token struct {
	// Type defines the type of the Token, e.g. opening, closing, or universal tag, or an escape sequence.
	Type TokenType

	// TagID a unique leading byte of the tag byte sequence, defined by the User.
	TagID byte

	// Pos defines the starting byte position of the tag's sequence in the input string.
	Pos int

	// Width defines count of bytes in the tag's sequence.
	//
	// Example: Imagine, you have defined a universal tag with name 'BOLD' and a byte sequence of "$$". The sequence has
	// 2 bytes in it, 1 per each '$', so the corresponding token will have width of 2.
	Width int

	// Raw defines the substring associated with the tag's value including both tag strings and the inner plain text.
	//
	// Example: Imagine, you have defined a greedy tag with name 'URL' and a pattern like this: "(...)", where
	// '(' is the opening tag and the ')' is the closing tag. When interpreting string "(https://some-address.com)",
	// the Raw field will consist of the entire matched string, that is the "(https://some-address.com)".
	// For Text tokens Raw and Inner fields are the same.
	Raw string

	// Inner defines the plain text, in case of token with type [TokenText], or the matched string, stripped of tags.
	//
	// Example: Imagine, you have defined a greedy tag with name 'URL' and a pattern like this: "(...)", where
	// '(' is the opening tag and the ')' is the closing tag. When interpreting string "(https://some-address.com)",
	// the Inner field will consist of only the "https://some-address.com".
	Inner string
}
