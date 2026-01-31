package scum

// TokenType can define a Text, a Tag, an EscapeSequence, or an Attribute.
type TokenType int

const (
	// TokenText means the [Token] contains plain text.
	TokenText TokenType = iota

	// TokenTag means the [Token] contains some [Tag] string representation.
	// The Tag can be Opening, Closing, or Universal.
	TokenTag

	// TokenAttributeKV means an Attribute of kind key-value.
	TokenAttributeKV

	// TokenAttributeFlag means a named Attribute without a value (a boolean flag).
	TokenAttributeFlag
)

// Rule defines the optional check, available only for single-char universal tags.
type Rule uint8

const (
	// RuleNA means no additional checks.
	RuleNA Rule = iota

	// RuleInfraWord allows to check if the single char non-greedy universal [Tag] is an actual Tag or a plain text.
	// If the both sides of the Tag's symbol contain an alphanumeric, a punctuation, or the symbol itself, the symbol
	// is cnsidered a plain text.
	RuleInfraWord

	// RuleTagVsContent allows to avoid the confusion of the starting and closing sequences of a greedy tag and it's content,
	// by ensuring that the Tag sequences and the content sequence have different length.
	RuleTagVsContent
)

// Greed defines the next-char-consuming policy of the [Tag] during the tokenization.
type Greed uint8

const (
	// NonGreedy Tags will be tokenized only as Tag's opening or closing sequences.
	NonGreedy Greed = iota

	// Greedy Tags will be tokenized as one [Token] along with all the next characters, between the opening and closing Tags,
	// but only if the closing Tag is available. Otherwise, the behaviour is the same as with Non-greedy Tags.
	Greedy

	// Grasping Tags will be tokenized along with all the next characters, between the opening and closing Tags. In case
	// of the missing closing Tag, the entire rest of the string after the starting Tag will be part of the Token.
	Grasping
)

const (
	MaxTagLen                int   = 4 // Max length in bytes of the Tag's string representation.
	MaxGreedLevel            Greed = Grasping
	MaxRule                  Rule  = RuleTagVsContent
	MaxTagNameLen            int   = 20   // Max count of UTF-8 chars, not bytes, that the Tag's name can contain.
	DefaultMaxAttrKeyLen     int   = 128  // Default max number of bytes in the attribute's name.
	DefaultMaxAttrPayloadLen int   = 512  // Default max number of bytes in the attribute's payload.
	DefaultMaxPayloadLen     int   = 1024 // Default max number of bytes in the Tag's payload.
	DefaultMaxKeyLen         int   = 128  // Default max number of bytes in the Tag-Vs-Content opening and closing sequences.
)

// ByteToTokenRatio is the ratio of the number of bytes in the input string to the number of Tokens.
// It is used to estimate the initial capacity of the Tokens slice.
const ByteToTokenRatio = 3 / 1
