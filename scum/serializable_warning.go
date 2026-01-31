package scum

import "strconv"

// SerializableWarning is a serializable human-readable description of the issue found in the input.
type SerializableWarning struct {
	// ByteIdx is the position of the starting byte of the erroneous sequence in the input.
	ByteIdx int `json:"byte_idx"`
	// SymbolIdx is the position of the symbol/letter causing the issue.
	SymbolIdx int `json:"symbol_idx"`
	// Issue is the name of the issue.
	Issue string `json:"issue"`
	// Description is a human-readable description of the issue.
	Description string `json:"description"`
}

var mapIssueToName [NumIssues]string

func init() {
	mapIssueToName[IssueAmbiguousTagType] = "Ambiguous Tag Type"
	mapIssueToName[IssueAttrKeyTooLong] = "Attribute Key Too Long"
	mapIssueToName[IssueAttrPayloadTooLong] = "Attribute Payload Too Long"
	mapIssueToName[IssueDuplicateNestedTag] = "Duplicate Nested Tag"
	mapIssueToName[IssueDuplicateTagID] = "Duplicate Tag ID"
	mapIssueToName[IssueEmptyAttrPayload] = "Empty Atribute Payload"
	mapIssueToName[IssueInvalidAttrSymbol] = "Invalid Attribute Symbol"
	mapIssueToName[IssueInvalidGreedLevel] = "Invalid Greed Level"
	mapIssueToName[IssueInvalidTagSeq] = "Invalid Tag Sequence"
	mapIssueToName[IssueInvalidTagSeqLen] = "Invalid Tag Sequence Length"
	mapIssueToName[IssueMisplacedClosingTag] = "Misplaced Closing Tag"
	mapIssueToName[IssueNegativeLimit] = "Negative Limit"
	mapIssueToName[IssueNegativeWarningsCap] = "Negative Warnings Cap"
	mapIssueToName[IssueOpenCloseTagMismatch] = "Open/Close Tag Mismatch"
	mapIssueToName[IssueRedundantEscape] = "Redundant Escape"
	mapIssueToName[IssueRuleInapplicable] = "Rule Inapplicable"
	mapIssueToName[IssueTagKeyTooLong] = "Tag Key Too Long"
	mapIssueToName[IssueTagPayloadTooLong] = "Tag Payload Too Long"
	mapIssueToName[IssueUnexpectedEOL] = "Unexpected End of Line"
	mapIssueToName[IssueUnexpectedSymbol] = "Unexpected Symbol"
	mapIssueToName[IssueUnclosedAttrPayload] = "Unclosed Attribute Payload"
	mapIssueToName[IssueUnclosedTag] = "Unclosed Tag"
	mapIssueToName[IssueUnprintableChar] = "Unprintable Character"
	mapIssueToName[IssueWarningsTruncated] = "Warnings Truncated"
	mapIssueToName[IssueInvalidRule] = "Invalid Rule"
	mapIssueToName[IssueInvalidTagNameLen] = "Invalud Tag Name Length"

	serializers[IssueUnexpectedEOL] = serializeUnexpectedEOL
	serializers[IssueUnexpectedSymbol] = serializeUnexpectedSymbol
	serializers[IssueUnclosedTag] = serializeUnclosedTag
	serializers[IssueMisplacedClosingTag] = serializeMisplacedClosingTag
	serializers[IssueInvalidGreedLevel] = serializeGeneric
	serializers[IssueInvalidRule] = serializeGeneric
	serializers[IssueAmbiguousTagType] = serializeGeneric
	serializers[IssueInvalidTagNameLen] = serializeGeneric
	serializers[IssueInvalidTagSeqLen] = serializeGeneric
	serializers[IssueDuplicateTagID] = serializeGeneric
	serializers[IssueInvalidTagSeq] = serializeGeneric
	serializers[IssueRuleInapplicable] = serializeGeneric
	serializers[IssueRedundantEscape] = serializeRedundantEscape
	serializers[IssueUnprintableChar] = serializeGeneric
	serializers[IssueWarningsTruncated] = serializeWarningsTruncated
	serializers[IssueNegativeWarningsCap] = serializeGeneric
	serializers[IssueEmptyAttrPayload] = serializeGeneric
	serializers[IssueUnclosedAttrPayload] = serializeGeneric
	serializers[IssueAttrKeyTooLong] = serializeGeneric
	serializers[IssueAttrPayloadTooLong] = serializeGeneric
	serializers[IssueInvalidAttrSymbol] = serializeGeneric
	serializers[IssueNegativeLimit] = serializeGeneric
	serializers[IssueTagKeyTooLong] = serializeTagKeyTooLong
	serializers[IssueTagPayloadTooLong] = serializeTagPayloadTooLong
	serializers[IssueOpenCloseTagMismatch] = serializeOpenCloseTagMismatch
	serializers[IssueDuplicateNestedTag] = serializeDuplicateNestedTag
}

type warnSerializer func(w Warning, d *Dictionary) SerializableWarning

var serializers [NumIssues]warnSerializer

// serialize converts a Warning to a SerializableWarning using the appropriate serializer.
func serialize(w Warning, d *Dictionary) SerializableWarning {
	if serializers[w.Issue] != nil {
		return serializers[w.Issue](w, d)
	}
	return serializeGeneric(w, d)
}

// SerializeAll converts a slice of Warnings to SerializableWarnings.
func (w Warnings) SerializeAll(target *[]SerializableWarning, d *Dictionary) {
	for _, w := range w.list {
		(*target) = append(*target, serialize(w, d))
	}
}

func serializeGeneric(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		ByteIdx: w.Pos,
		Issue:   mapIssueToName[w.Issue],
	}
}

func serializeWarningsTruncated(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: "too many warnings; further warnings suppressed",
	}
}

func serializeUnclosedTag(w Warning, d *Dictionary) SerializableWarning {
	desc := `unclosed tag with name "` +
		d.tags[w.TagID].Name +
		`": expected closing tag with name "` +
		d.tags[w.CloseTagID].Name + `".`

	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: desc,
	}
}

func serializeTagPayloadTooLong(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: "tag payload's length limit reached.",
	}
}

func serializeTagKeyTooLong(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: "tag's opening/closing sequence length limit reached.",
	}
}

func serializeUnexpectedEOL(w Warning, d *Dictionary) SerializableWarning {
	var desc string
	if w.TagID != 0 {
		desc = `opening tag with name "` +
			d.tags[w.TagID].Name +
			`" was found at the very end of the input and will be treated as plain text.`
	} else {
		desc = "redundant escape symbol found at the very end of the input."
	}
	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: desc,
	}
}

func serializeUnexpectedSymbol(w Warning, d *Dictionary) SerializableWarning {
	desc := `unexpected symbol while processing the tag with name "` +
		d.tags[w.TagID].Name +
		`": expected to get "` + string(w.Expected) +
		`", but got "` + string(w.Got) + `".`

	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: desc,
	}
}

func serializeMisplacedClosingTag(w Warning, d *Dictionary) SerializableWarning {
	var desc string
	if w.TagID != 0 {
		tag := d.tags[w.TagID]
		if w.Expected == 0 {
			desc = `closing tag with name "` +
				tag.Name +
				`" found at the very start of the input and will be treated as plain text.`
		} else {
			desc = `closing tag with name "` +
				d.tags[w.TagID].Name +
				`" expected to have an opening counterpart with name "` +
				d.tags[tag.OpenID].Name + `" which is missing in the input.`
		}
	} else {
		desc = "misplaced closing tag."
	}
	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: desc,
	}
}

func serializeRedundantEscape(w Warning, d *Dictionary) SerializableWarning {
	var got string
	if w.Got != 0 {
		got = string(w.Got)
	} else {
		got = "invalid UTF-8 sequence"
	}
	desc := "redundant escape symbol found at index " +
		strconv.Itoa(w.Pos) + ", before non-special " + got + "."

	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: desc,
	}
}

func serializeOpenCloseTagMismatch(w Warning, d *Dictionary) SerializableWarning {
	desc := `closing tag with name "` +
		d.tags[w.TagID].Name +
		`" cannot match with opening tag with name "` +
		d.tags[w.Expected].Name + `".`

	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: desc,
	}
}

func serializeDuplicateNestedTag(w Warning, d *Dictionary) SerializableWarning {
	desc := `tag with name "` +
		d.tags[w.TagID].Name +
		`" is a descendant of the tag with the same name.`

	return SerializableWarning{
		ByteIdx:     w.Pos,
		Issue:       mapIssueToName[w.Issue],
		Description: desc,
	}
}
