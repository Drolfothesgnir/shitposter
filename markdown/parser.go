package markdown

// Warning represents details about a non-critical issue that occurred
// during parsing. Parsing still succeeded and produced an AST/output,
// but something about the input was inconsistent, ambiguous, or malformed.
type Warning struct {
	// Node is the type of the node or construct that is inconsistent in some way,
	// e.g. an unclosed BOLD span, a redundant ESCAPE, a malformed LINK, etc.
	Node NodeType `json:"node"`

	// Index is the byte offset in the raw input string where the problem was detected.
	//
	// IMPORTANT:
	// - Index is a byte position, not a character (rune) index.
	// - This matters when the string contains non-ASCII characters.
	//
	// For example:
	//
	//   s := "hi привет"
	//
	// The bytes look like:
	//   'h'(0) 'i'(1) ' '(2) 'п'(3..4) 'р'(5..6) 'и'(7..8) ...
	//
	// Here Index == 3 points to the *first byte* of 'п', not a whole "character"
	// in human terms. UI code that works with runes must convert the byte index
	// into a rune index (e.g. using utf8.RuneCountInString(s[:index])) before
	// highlighting or slicing by "characters".
	Index int `json:"index"`

	// Near is an optional short snippet (often a single character or a few characters)
	// from the raw input around Index that likely caused the problem.
	Near string `json:"near"`

	// Issue describes the category of the problem (e.g. unclosed tag,
	// mis-nested tag, redundant escape, malformed link).
	Issue Issue `json:"issue"`

	// Description is a human-readable explanation of what went wrong.
	Description string `json:"description"`

	// Suggestion is an optional human-readable hint that may help the user
	// fix or understand the issue.
	Suggestion string `json:"suggestion"`
}

// ParseResult defines the output of the Parser.Parse method. It contains:
//
//   - the original input string (RawInput)
//   - the processed / normalized representation (Output)
//   - the visible text length (TextLength)
//   - a list of non-critical issues (Warnings)
//   - the parsed inline AST (AST)
type ParseResult struct {
	// RawInput is the original input string passed into Parse.
	RawInput string `json:"raw_input"`

	// Output is the processed/normalized version of the input string.
	//
	// Examples of normalization:
	//   - removing useless escapes: "\" not followed by a special character
	//   - collapsing duplicated tags that cancel out or don't affect semantics
	//     (depending on how you define idempotency)
	//
	// Tags that are not closed properly should typically generate a Warning
	// rather than a fatal error, and the parser should still produce a
	// best-effort Output and AST.
	//
	// NOTE: Depending on the parser implementation, Output may still contain
	// raw user-supplied substrings (including things that look like HTML).
	// It must not be assumed safe for direct HTML injection without additional
	// escaping/sanitization at render time.
	Output string `json:"output"`

	// TextLength is the count of actual text characters (runes) that are
	// considered "visible content", excluding markdown control characters
	// such as '*', '**', '~~', etc.
	TextLength int `json:"text_length"`

	// Warnings is a list of non-critical issues detected during parsing.
	// Parsing still succeeded, but the input was not fully well-formed.
	Warnings []Warning `json:"warnings"`

	// AST is the inline abstract syntax tree parsed from the input markdown.
	// It represents the structured form of the content (TEXT, BOLD, ITALIC, LINK, etc.).
	AST Node `json:"ast"`
}

// Parser defines a markdown engine implementation.
type Parser interface {
	// Name returns the name of the markdown engine implementation.
	Name() string

	// Version returns the version of the markdown engine.
	Version() int32

	// Parse processes a raw markdown string and returns a ParseResult and
	// an optional fatal error.
	//
	// Typical behavior:
	//   - On success with only non-critical issues: return a populated
	//     ParseResult and a nil error, with details in Warnings.
	//
	//   - On fatal errors (e.g. serious internal bug, resource exhaustion,
	//     or input that cannot be reasonably parsed at all), return a
	//     non-nil error. In such cases, the ParseResult may be zero-valued
	//     or partially populated, depending on your design.
	Parse(input string) (ParseResult, error)
}
