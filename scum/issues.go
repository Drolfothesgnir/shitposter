package scum

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

	// IssueMisplacedClosingTag means that the closing [Tag] is placed at the very beginning of the input, or
	// the Tag has missing opening Tag.
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

	// IssueTagKeyTooLong occurs when the opening Tag sequence is longer [Limits.MaxKeyLen].
	IssueTagKeyTooLong

	// IssueTagPayloadTooLong occurs when the Tag's payload is longer than [Limits.MaxPayloadLen].
	IssueTagPayloadTooLong

	// IssueOpenCloseTagMismatch occurs when the opening Tag's [Tag.CloseID] != closing Tag's [Tag.OpenID].
	IssueOpenCloseTagMismatch

	// FIXME: refactor doc comment
	// IssueDuplicateNestedTag occurs when the Tag is nested in the Tag with the same ID.
	IssueDuplicateNestedTag
)
