package scum

// TokenType can define a Text, a Tag, an EscapeSequence, or an Attribute.
type TokenType int

const (
	// TokenText means the [Token] contains plain text.
	TokenText TokenType = iota

	// TokenTag means the [Token] contains some [Tag] string representation.
	// The Tag can be Opening, Closing, or Universal.
	TokenTag

	// TokenEscapeSequence means the [Token] contains the escape character and the next character
	// after it.
	TokenEscapeSequence

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
	MaxTagLen                  int   = 4 // Max length in bytes of the Tag's string representation.
	MaxGreedLevel              Greed = Grasping
	MaxRule                    Rule  = RuleTagVsContent
	MaxTagNameLen              int   = 20   // Max count of UTF-8 chars, not bytes, that the Tag's name can contain.
	DefaultMaxAttrKeyLen       int   = 128  // Default max number of bytes in the attribute's name.
	DefaultMaxAttrPayloadLen   int   = 512  // Default max number of bytes in the attribute's payload.
	DefaultMaxGreedyPayloadLen int   = 1024 // Default max number of bytes in the Greedy Tag's payload.
)

// Issue defines types of problems we might encounter during the tokenizing or the parsing processes.
type Issue int

const (
	// IssueUnexpectedEOL means that either the specific symbol or a plain text was expected,
	// but the input string was terminated.
	IssueUnexpectedEOL Issue = iota

	// IssueUnexpectedSymbol usually occurs during the tokenization, when the occured symbol breaks the Tag's
	// opening or closing sequence.
	IssueUnexpectedSymbol

	// IssueUnclosedTag means that a Tag needs to be closed, but no closing Tag is found in the input.
	IssueUnclosedTag

	// IssueMisplacedClosingTag means that the closing [Tag] is placed at the very beginning of the input.
	IssueMisplacedClosingTag

	// IssueInvalidGreedLevel means the Tag's Greed level is greater than [MaxGreedLevel].
	IssueInvalidGreedLevel

	// IssueInvalidRule means the [Rule] is not applicable to the current [Tag], or it's value is higher than [MaxRule].
	IssueInvalidRule

	// IssueAmbiguousTagType means that the [Tag] has both [Tag.OpenID] and [Tag.CloseID] set, which means
	// the Tag can't be classified as either opening or closing and the fields are not equal to the Tag's ID,
	// which means the Tag can't be classified as Universal.
	IssueAmbiguousTagType

	// IssueInvalidTagNameLen occurs when the Tag's name is either empty string or it has length greater than
	// [MaxTagNameLen].
	IssueInvalidTagNameLen

	// IssueInvalidTagSeqLen occurs when the Tag's string representation is either empty or is longer than [MaxTagLen].
	IssueInvalidTagSeqLen

	// IssueDuplicateTagID occurs when the [Tag] with the same ID is already registered.
	IssueDuplicateTagID

	// IssueInvalidTagSeq occurs when the Tag's string representation contains unprintable chars.
	IssueInvalidTagSeq

	// IssueRuleInapplicable occurs when the [Rule] is not avaliable due to [Greed] level or the [Tag] being multi-char.
	IssueRuleInapplicable

	// IssueRedundantEscape occurs when the next byte after the escape symbol does not trigger any [Action], and
	// considered a plain text.
	IssueRedundantEscape

	// IssueUnprintableChar occurs when the symbol is not printable ASCII character
	IssueUnprintableChar

	// IssueWarningsTruncated occurs when there are too many Warnings recorded.
	IssueWarningsTruncated

	// IssueNegativeWarningsCap reports an invalid (negative) warnings capacity.
	IssueNegativeWarningsCap

	// IssueEmptyAttrPayload occurs when an attribute payload is present but empty, e.g. "!k{}" or "!{}".
	IssueEmptyAttrPayload

	// IssueUnclosedAttrPayload occurs when the attribute payload start is found, but the payload end symbol is missing.
	IssueUnclosedAttrPayload

	// IssueAttrKeyTooLong occurs when the attribute payload start symbol is not found within [Limits.MaxAttrKeyLen].
	IssueAttrKeyTooLong

	// IssueAttrPayloadTooLong occurs when the attribute payload end symbol is not found within [Limits.MaxAttrPayloadLen].
	IssueAttrPayloadTooLong

	// IssueInvalidAttrSymbol occurs during configuration when attribute symbols form an invalid signature
	// (e.g. trigger equals payload start/end).
	IssueInvalidAttrSymbol

	// IssueNegativeLimit occurs during configuration when any value in [Limits] is negative.
	IssueNegativeLimit
)
