package scum

import "fmt"

// Limits define upper bounds used during tokenization and parsing to prevent
// excessive scanning, memory growth and potential denial-of-service scenarios.
type Limits struct {
	// MaxAttrKeyLen defines the maximum number of bytes scanned for an attribute key
	// (from the attribute trigger up to the payload start symbol).
	//
	// If the payload start symbol is not found within this limit, the attribute trigger
	// is treated as plain text and [IssueAttrKeyTooLong] is recorded.
	//
	// Measured in bytes, not UTF-8 runes.
	MaxAttrKeyLen int

	// MaxAttrPayloadLen defines the maximum number of bytes scanned for an attribute payload
	// (from the payload start symbol up to the payload end symbol).
	//
	// If the payload end symbol is not found within this limit, the attribute trigger
	// is treated as plain text and [IssueAttrPayloadTooLong] is recorded.
	//
	// Measured in bytes, not UTF-8 runes.
	MaxAttrPayloadLen int

	// MaxPayloadLen defines the maximum number of bytes scanned for a Greedy
	// Tag's body. It also applies to the Tag-Vs-Content based search.
	//
	// A value of 0 is replaced with [DefaultMaxPayloadLen] by [NewDictionary].
	MaxPayloadLen int

	// MaxKeyLen defines the maximum number of bytes scanned for opening and
	// closing sequences used by the Tag-Vs-Content rule.
	//
	// A value of 0 is replaced with [DefaultMaxKeyLen] by [NewDictionary].
	MaxKeyLen int

	// MaxNodes defines the maximum number of [Node] entries [ParseInto] may keep
	// in the returned [AST]. The root node counts toward this limit.
	//
	// When the limit is reached, further nodes are omitted and
	// [IssueMaxNodesExceeded] is recorded. A value of 0 means no node count
	// limit.
	MaxNodes int

	// MaxAttributes defines the maximum number of [Attribute] entries [ParseInto]
	// may keep in the returned [AST].
	//
	// When the limit is reached, further attributes are omitted and
	// [IssueMaxAttributesExceeded] is recorded. A value of 0 means no attribute
	// count limit.
	MaxAttributes int

	// MaxParseDepth defines the maximum number of simultaneously open tag nodes.
	// The root node is not counted.
	//
	// When the limit is reached, further opening tags are omitted and
	// [IssueMaxParseDepthExceeded] is recorded. A value of 0 means no parse
	// depth limit.
	MaxParseDepth int
}

// Validate checks that all limits are non-negative.
// It returns [ConfigError] if at least one value is negative.
func (l Limits) Validate() error {

	values := [...]int{
		l.MaxAttrKeyLen,
		l.MaxAttrPayloadLen,
		l.MaxPayloadLen,
		l.MaxKeyLen,
		l.MaxNodes,
		l.MaxAttributes,
		l.MaxParseDepth,
	}

	names := [...]string{
		"MaxAttrKeyLen",
		"MaxAttrPayloadLen",
		"MaxPayloadLen",
		"MaxKeyLen",
		"MaxNodes",
		"MaxAttributes",
		"MaxParseDepth",
	}

	for i := range values {
		if values[i] < 0 {
			err := fmt.Errorf("%s must be >= 0, got %d", names[i], values[i])
			return NewConfigError(IssueNegativeLimit, err)
		}
	}

	return nil
}
