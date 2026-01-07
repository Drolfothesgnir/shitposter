package scum

import "fmt"

// Limits define upper bounds used during tokenization to prevent excessive scanning
// and potential denial-of-service scenarios.
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
}

// Validate checks if the limits are not negative.
// Return [ConfigError] if at least on of the values is negative.
func (l Limits) Validate() error {

	values := [2]int{
		l.MaxAttrKeyLen,
		l.MaxAttrPayloadLen,
	}

	names := [2]string{
		"MaxAttrKeyLen",
		"MaxAttrPayloadLen",
	}

	for i := range values {
		if values[i] < 0 {
			err := fmt.Errorf("%s must be >= 0, got %d", names[i], values[i])
			return NewConfigError(IssueNegativeLimit, err)
		}
	}

	return nil
}
