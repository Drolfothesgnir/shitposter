package scum

import "strconv"

// SerializableWarning is a JSON-friendly description of an issue found in the input.
type SerializableWarning struct {
	// Code is the numeric ID of the [Issue].
	Code Issue `json:"code"`
	// Codename is the stable string name of the [Issue].
	Codename string `json:"codename"`
	// ByteIdx is the position of the starting byte of the erroneous sequence in the input.
	ByteIdx int `json:"byte_idx"`
	// SymbolIdx is the position of the symbol/letter causing the issue.
	SymbolIdx int `json:"symbol_idx"`
	// Description is a human-readable description of the issue.
	Description string `json:"description"`
}

var (
	// mapIssueToSumm     [NumIssues]string
	mapIssueToCodename [NumIssues]string
)

func init() {
	// mapIssueToSumm[IssueAmbiguousTagType] = "Ambiguous Tag Type"
	// mapIssueToSumm[IssueAttrKeyTooLong] = "Attribute Key Too Long"
	// mapIssueToSumm[IssueAttrPayloadTooLong] = "Attribute Payload Too Long"
	// mapIssueToSumm[IssueDuplicateNestedTag] = "Duplicate Nested Tag"
	// mapIssueToSumm[IssueDuplicateTagID] = "Duplicate Tag ID"
	// mapIssueToSumm[IssueEmptyAttrPayload] = "Empty Atribute Payload"
	// mapIssueToSumm[IssueInvalidAttrSymbol] = "Invalid Attribute Symbol"
	// mapIssueToSumm[IssueInvalidGreedLevel] = "Invalid Greed Level"
	// mapIssueToSumm[IssueInvalidTagSeq] = "Invalid Tag Sequence"
	// mapIssueToSumm[IssueInvalidTagSeqLen] = "Invalid Tag Sequence Length"
	// mapIssueToSumm[IssueMisplacedClosingTag] = "Misplaced Closing Tag"
	// mapIssueToSumm[IssueNegativeLimit] = "Negative Limit"
	// mapIssueToSumm[IssueNegativeWarningsCap] = "Negative Warnings Cap"
	// mapIssueToSumm[IssueOpenCloseTagMismatch] = "Open/Close Tag Mismatch"
	// mapIssueToSumm[IssueRedundantEscape] = "Redundant Escape"
	// mapIssueToSumm[IssueRuleInapplicable] = "Rule Inapplicable"
	// mapIssueToSumm[IssueTagKeyTooLong] = "Tag Key Too Long"
	// mapIssueToSumm[IssueTagPayloadTooLong] = "Tag Payload Too Long"
	// mapIssueToSumm[IssueUnexpectedEOL] = "Unexpected End of Line"
	// mapIssueToSumm[IssueUnexpectedSymbol] = "Unexpected Symbol"
	// mapIssueToSumm[IssueUnclosedAttrPayload] = "Unclosed Attribute Payload"
	// mapIssueToSumm[IssueUnclosedTag] = "Unclosed Tag"
	// mapIssueToSumm[IssueUnprintableChar] = "Unprintable Character"
	// mapIssueToSumm[IssueWarningsTruncated] = "Warnings Truncated"
	// mapIssueToSumm[IssueInvalidRule] = "Invalid Rule"
	// mapIssueToSumm[IssueInvalidTagNameLen] = "Invalid Tag Name Length"

	mapIssueToCodename[issueIndex(IssueAmbiguousTagType)] = "AMBIGUOUS_TAG_TYPE"
	mapIssueToCodename[issueIndex(IssueAttrKeyTooLong)] = "ATTR_KEY_TOO_LONG"
	mapIssueToCodename[issueIndex(IssueAttrPayloadTooLong)] = "ATTR_PAYLOAD_TOO_LONG"
	mapIssueToCodename[issueIndex(IssueDuplicateNestedTag)] = "DUPLICATE_NESTED_TAG"
	mapIssueToCodename[issueIndex(IssueDuplicateTagID)] = "DUPLICATE_TAG_ID"
	mapIssueToCodename[issueIndex(IssueEmptyAttrPayload)] = "EMPTY_ATTR_PAYLOAD"
	mapIssueToCodename[issueIndex(IssueInvalidAttrSymbol)] = "INVALID_ATTR_SYMBOL"
	mapIssueToCodename[issueIndex(IssueInvalidGreedLevel)] = "INVALID_GREED_LEVEL"
	mapIssueToCodename[issueIndex(IssueInvalidTagSeq)] = "INVALID_TAG_SEQ"
	mapIssueToCodename[issueIndex(IssueInvalidTagSeqLen)] = "INVALID_TAG_SEQ_LEN"
	mapIssueToCodename[issueIndex(IssueMisplacedClosingTag)] = "MISPLACED_CLOSING_TAG"
	mapIssueToCodename[issueIndex(IssueNegativeLimit)] = "NEGATIVE_LIMIT"
	mapIssueToCodename[issueIndex(IssueNegativeWarningsCap)] = "NEGATIVE_WARNINGS_CAP"
	mapIssueToCodename[issueIndex(IssueOpenCloseTagMismatch)] = "OPEN_CLOSE_TAG_MISMATCH"
	mapIssueToCodename[issueIndex(IssueRedundantEscape)] = "REDUNDANT_ESCAPE"
	mapIssueToCodename[issueIndex(IssueRuleInapplicable)] = "RULE_INAPPLICABLE"
	mapIssueToCodename[issueIndex(IssueTagKeyTooLong)] = "TAG_KEY_TOO_LONG"
	mapIssueToCodename[issueIndex(IssueTagPayloadTooLong)] = "TAG_PAYLOAD_TOO_LONG"
	mapIssueToCodename[issueIndex(IssueUnexpectedEOL)] = "UNEXPECTED_EOL"
	mapIssueToCodename[issueIndex(IssueUnexpectedSymbol)] = "UNEXPECTED_SYMBOL"
	mapIssueToCodename[issueIndex(IssueUnclosedAttrPayload)] = "UNCLOSED_ATTR_PAYLOAD"
	mapIssueToCodename[issueIndex(IssueUnclosedTag)] = "UNCLOSED_TAG"
	mapIssueToCodename[issueIndex(IssueUnprintableChar)] = "UNPRINTABLE_CHAR"
	mapIssueToCodename[issueIndex(IssueWarningsTruncated)] = "WARNINGS_TRUNCATED"
	mapIssueToCodename[issueIndex(IssueInvalidRule)] = "INVALID_RULE"
	mapIssueToCodename[issueIndex(IssueInvalidTagNameLen)] = "INVALID_TAG_NAME_LEN"
	mapIssueToCodename[issueIndex(IssueMaxNodesExceeded)] = "MAX_NODES_EXCEEDED"
	mapIssueToCodename[issueIndex(IssueMaxAttributesExceeded)] = "MAX_ATTRIBUTES_EXCEEDED"
	mapIssueToCodename[issueIndex(IssueMaxParseDepthExceeded)] = "MAX_PARSE_DEPTH_EXCEEDED"

	serializers[issueIndex(IssueUnexpectedEOL)] = serializeUnexpectedEOL
	serializers[issueIndex(IssueUnexpectedSymbol)] = serializeUnexpectedSymbol
	serializers[issueIndex(IssueUnclosedTag)] = serializeUnclosedTag
	serializers[issueIndex(IssueMisplacedClosingTag)] = serializeMisplacedClosingTag
	serializers[issueIndex(IssueInvalidGreedLevel)] = serializeGeneric
	serializers[issueIndex(IssueInvalidRule)] = serializeGeneric
	serializers[issueIndex(IssueAmbiguousTagType)] = serializeGeneric
	serializers[issueIndex(IssueInvalidTagNameLen)] = serializeGeneric
	serializers[issueIndex(IssueInvalidTagSeqLen)] = serializeGeneric
	serializers[issueIndex(IssueDuplicateTagID)] = serializeGeneric
	serializers[issueIndex(IssueInvalidTagSeq)] = serializeGeneric
	serializers[issueIndex(IssueRuleInapplicable)] = serializeGeneric
	serializers[issueIndex(IssueRedundantEscape)] = serializeRedundantEscape
	serializers[issueIndex(IssueUnprintableChar)] = serializeGeneric
	serializers[issueIndex(IssueWarningsTruncated)] = serializeWarningsTruncated
	serializers[issueIndex(IssueNegativeWarningsCap)] = serializeGeneric
	serializers[issueIndex(IssueEmptyAttrPayload)] = serializeGeneric
	serializers[issueIndex(IssueUnclosedAttrPayload)] = serializeGeneric
	serializers[issueIndex(IssueAttrKeyTooLong)] = serializeGeneric
	serializers[issueIndex(IssueAttrPayloadTooLong)] = serializeGeneric
	serializers[issueIndex(IssueInvalidAttrSymbol)] = serializeGeneric
	serializers[issueIndex(IssueNegativeLimit)] = serializeGeneric
	serializers[issueIndex(IssueTagKeyTooLong)] = serializeTagKeyTooLong
	serializers[issueIndex(IssueTagPayloadTooLong)] = serializeTagPayloadTooLong
	serializers[issueIndex(IssueOpenCloseTagMismatch)] = serializeOpenCloseTagMismatch
	serializers[issueIndex(IssueDuplicateNestedTag)] = serializeDuplicateNestedTag
	serializers[issueIndex(IssueMaxNodesExceeded)] = serializeMaxNodesExceeded
	serializers[issueIndex(IssueMaxAttributesExceeded)] = serializeMaxAttributesExceeded
	serializers[issueIndex(IssueMaxParseDepthExceeded)] = serializeMaxParseDepthExceeded
}

type warnSerializer func(w Warning, d *Dictionary) SerializableWarning

var serializers [NumIssues]warnSerializer

// serialize converts a Warning to a SerializableWarning using the appropriate serializer.
func serialize(w Warning, d *Dictionary) SerializableWarning {
	idx := issueIndex(w.Issue)
	if serializers[idx] != nil {
		return serializers[idx](w, d)
	}
	return serializeGeneric(w, d)
}

// WarnCount returns total number of Warnings in the list.
func (w Warnings) WarnCount() int {
	return len(w.list)
}

// SerializeAll converts a slice of Warnings to SerializableWarnings.
func (w Warnings) SerializeAll(target *[]SerializableWarning, d *Dictionary) {
	for _, w := range w.list {
		(*target) = append(*target, serialize(w, d))
	}
}

func serializeGeneric(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:  mapIssueToSumm[w.Issue],
	}
}

func serializeWarningsTruncated(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
		Description: "too many warnings; further warnings suppressed",
	}
}

func serializeMaxNodesExceeded(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		Code:        w.Issue,
		Codename:    mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:     w.Pos,
		Description: "maximum AST node count reached; further nodes were omitted.",
	}
}

func serializeMaxAttributesExceeded(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		Code:        w.Issue,
		Codename:    mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:     w.Pos,
		Description: "maximum AST attribute count reached; further attributes were omitted.",
	}
}

func serializeMaxParseDepthExceeded(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		Code:        w.Issue,
		Codename:    mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:     w.Pos,
		Description: "maximum parse depth reached; further nested tag structure was omitted.",
	}
}

func serializeUnclosedTag(w Warning, d *Dictionary) SerializableWarning {
	desc := "unclosed tag with name " +
		d.tags[w.TagID].Name +
		": expected closing tag with name " +
		d.tags[w.CloseTagID].Name + "."

	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
		Description: desc,
	}
}

func serializeTagPayloadTooLong(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
		Description: "tag payload's length limit reached.",
	}
}

func serializeTagKeyTooLong(w Warning, d *Dictionary) SerializableWarning {
	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
		Description: "tag's opening/closing sequence length limit reached.",
	}
}

func serializeUnexpectedEOL(w Warning, d *Dictionary) SerializableWarning {
	var desc string
	if w.TagID != 0 {
		desc = "opening tag with name " +
			d.tags[w.TagID].Name +
			" was found at the very end of the input and will be treated as plain text."
	} else {
		desc = "redundant escape symbol found at the very end of the input."
	}
	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
		Description: desc,
	}
}

func serializeUnexpectedSymbol(w Warning, d *Dictionary) SerializableWarning {
	desc := "unexpected symbol while processing the tag with name " +
		d.tags[w.TagID].Name +
		": expected to get " + string(w.Expected) +
		", but got " + string(w.Got) + "."

	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
		Description: desc,
	}
}

func serializeMisplacedClosingTag(w Warning, d *Dictionary) SerializableWarning {
	var desc string
	if w.TagID != 0 {
		tag := d.tags[w.TagID]
		if w.Expected == 0 {
			desc = "closing tag with name " +
				tag.Name +
				" found at the very start of the input and will be treated as plain text."
		} else {
			desc = "closing tag with name " +
				d.tags[w.TagID].Name +
				" expected to have an opening counterpart with name " +
				d.tags[tag.OpenID].Name + " which is missing in the input."
		}
	} else {
		desc = "misplaced closing tag."
	}
	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
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
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
		Description: desc,
	}
}

func serializeOpenCloseTagMismatch(w Warning, d *Dictionary) SerializableWarning {
	desc := "closing tag with name " +
		d.tags[w.TagID].Name +
		" cannot match with opening tag with name " +
		d.tags[w.Expected].Name + "."

	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
		Description: desc,
	}
}

func serializeDuplicateNestedTag(w Warning, d *Dictionary) SerializableWarning {
	desc := "tag with name " +
		d.tags[w.TagID].Name +
		" is a descendant of the tag with the same name."

	return SerializableWarning{
		Code:     w.Issue,
		Codename: mapIssueToCodename[issueIndex(w.Issue)],
		ByteIdx:  w.Pos,
		// Summary:     mapIssueToSumm[w.Issue],
		Description: desc,
	}
}
