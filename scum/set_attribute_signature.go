package scum

import (
	"fmt"
	"strings"
)

// SetAttributeSignature registers attribute symbols.
// All symbols must be printable ASCII characters.
// trigger must be unique among the Tags and the escape symbol.
// trigger must be different from payloadStart and payloadEnd.
// payloadStart and payloadEnd does not need to be unique and can be the same symbol.
func (d *Dictionary) SetAttributeSignature(trigger, payloadStart, payloadEnd byte) error {
	// checking trigger uniqueness
	if d.actions[trigger] != nil {
		return newDuplicateTagIDError(trigger)
	}

	if trigger == payloadStart || trigger == payloadEnd {
		return NewConfigError(
			IssueInvalidAttrSymbol,
			fmt.Errorf("attribute trigger symbol %q must be different from payload start and payload end symbols",
				trigger))
	}

	names := [3]string{"trigger", "payload start symbol", "payload end symbol"}
	chars := [3]byte{trigger, payloadStart, payloadEnd}
	for i := range 3 {
		if !isASCIIPrintable(chars[i]) {
			return newUnprintableError("attribute "+names[i], chars[i])
		}
	}

	d.attrTrigger = trigger
	d.attrPayloadStart = payloadStart
	d.attrPayloadEnd = payloadEnd

	d.actions[trigger] = ActAttribute
	return nil
}

// ActAttribute processes the input from the index of the Attribute trigger.
// It returns proper Attribute [Token], only if the Attribute is well-formed, that is, the payload is non-empty
// and its start and end symbols are present in the input, after the trigger, and in the correct order.
// Otherwise, the trigger is skipped as a plain text.
func ActAttribute(ac *ActionContext) (token Token, stride int, skip bool) {
	i := ac.Idx

	n := len(ac.Input)

	// 1. Check if the Attribute trigger is the last byte in the string
	if i+1 == n {
		return skipWithWarn(ac.Warns, 1, i, IssueUnexpectedEOL)
	}

	// 2. Find payload start index
	scanEnd := min(i+1+ac.Dictionary.Limits.MaxAttrKeyLen+1, n)
	relIdx := strings.IndexByte(ac.Input[i+1:scanEnd], ac.Dictionary.attrPayloadStart)

	// 3. No payload start
	// technically, all text after the trigger, which is not payload start or end symbol,
	// is treated as an Attribute's name, so when we hit into EOL, we have name but no payload.
	// this is why i've chosen [IssueUnexpectedEOL] as description.
	if relIdx == -1 {
		// if the input string was not scanned till the end, the payload start symbol
		// still might be there, so it's a limit issue
		if scanEnd < n {
			return skipWithWarn(ac.Warns, 1, i, IssueAttrKeyTooLong)
		}

		return skipWithWarn(ac.Warns, 1, n, IssueUnexpectedEOL)
	}

	payloadStartIdx := i + 1 + relIdx

	// 4. Payload start at EOL
	if payloadStartIdx+1 == n {
		return skipWithWarn(ac.Warns, 1, payloadStartIdx, IssueUnexpectedEOL)
	}

	// 5. Find payload end
	payloadEndIdx := -1
	startIdx := payloadStartIdx + 1
	scanEnd = min(startIdx+ac.Dictionary.Limits.MaxAttrPayloadLen+1, n)

	if ac.Dictionary.escapeTrigger != 0 {
		payloadEndIdx = findPayloadEndWithEscape(ac.Input, startIdx, scanEnd, ac.Dictionary.attrPayloadEnd, ac.Dictionary.escapeTrigger)
	} else {
		payloadEndIdx = findPayloadEnd(ac.Input, startIdx, scanEnd, ac.Dictionary.attrPayloadEnd)
	}

	// 6. No payload end
	if payloadEndIdx == -1 {
		// same logic as with payload start idx
		if scanEnd < n {
			return skipWithWarn(ac.Warns, 1, i, IssueAttrPayloadTooLong)
		}

		return skipWithWarn(ac.Warns, 1, n, IssueUnclosedAttrPayload)
	}

	// 7. Empty attribute payload
	if payloadEndIdx == startIdx {
		return skipWithWarn(ac.Warns, 1, startIdx, IssueEmptyAttrPayload)
	}

	width := payloadEndIdx - i + 1

	stride = width
	skip = false

	payload := NewSpan(startIdx, payloadEndIdx-startIdx)

	token = Token{
		Trigger: ac.Trigger,
		Pos:     i,
		Width:   width,
		Payload: payload,
	}

	// increment attributes counter
	ac.State.Attributes++

	// 8. Attribute is a flag
	if payloadStartIdx-i-1 == 0 {
		token.Type = TokenAttributeFlag
		return
	}

	// 9. Attribute is key-value
	token.Type = TokenAttributeKV
	token.AttrKey = NewSpan(i+1, payloadStartIdx-i-1)
	return
}

// findPayloadEndWithEscape returns the index of the payload end symbol, with an account of it possibly
// being escaped.
// Returns -1 if the unescaped symbol was not found.
func findPayloadEndWithEscape(input string, start, scanEnd int, payloadEnd, esc byte) int {
	escaped := false
	for j := start; j < scanEnd; j++ {
		c := input[j]

		if escaped {
			escaped = false
			continue
		}

		if c == esc {
			escaped = true
			continue
		}

		if c == payloadEnd {
			return j
		}
	}

	return -1
}

// findPayloadEnd returns the index of the payload end symbol.
func findPayloadEnd(input string, start, scanEnd int, payloadEnd byte) int {
	relIdx := strings.IndexByte(input[start:scanEnd], payloadEnd)
	if relIdx != -1 {
		return start + relIdx
	}

	return -1
}
