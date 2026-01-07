package scum

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func spanStr(input string, sp Span) string {
	if sp.End < sp.Start || sp.Start < 0 || sp.End > len(input) {
		return ""
	}
	return input[sp.Start:sp.End]
}

func newWarnings(t *testing.T) *Warnings {
	t.Helper()

	// Use any policy/cap you support; we just need something that records warnings.
	w, err := NewWarnings(WarnOverflowDrop, 128)
	require.NoError(t, err)

	return &w
}

func requireConfigIssue(t *testing.T, err error, want Issue) {
	t.Helper()
	require.Error(t, err)

	var ce *ConfigError
	require.True(t, errors.As(err, &ce), "expected ConfigError, got %T: %v", err, err)
	require.Equal(t, want, ce.Issue)
}

func TestSetAttributeSignature_OK(t *testing.T) {
	d, err := NewDictionary(Limits{
		MaxAttrKeyLen:     10,
		MaxAttrPayloadLen: 10,
	})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	// action registered
	act, ok := d.Action('!')
	require.True(t, ok)
	require.NotNil(t, act)

	// internal signature set
	require.Equal(t, byte('!'), d.attrTrigger)
	require.Equal(t, byte('{'), d.attrPayloadStart)
	require.Equal(t, byte('}'), d.attrPayloadEnd)

	// special now
	require.True(t, d.IsSpecial('!'))
}

func TestSetAttributeSignature_DuplicateTrigger(t *testing.T) {
	d, err := NewDictionary(Limits{})
	require.NoError(t, err)

	// First registration OK
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	// Second should fail (trigger uniqueness)
	err = d.SetAttributeSignature('!', '(', ')')
	requireConfigIssue(t, err, IssueDuplicateTagID)
}

func TestSetAttributeSignature_TriggerEqualsPayloadStartOrEnd(t *testing.T) {
	d, err := NewDictionary(Limits{})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '!', '}')
	requireConfigIssue(t, err, IssueInvalidAttrSymbol)

	err = d.SetAttributeSignature('!', '{', '!')
	requireConfigIssue(t, err, IssueInvalidAttrSymbol)
}

func TestSetAttributeSignature_Unprintable(t *testing.T) {
	d, err := NewDictionary(Limits{})
	require.NoError(t, err)

	// 0x1B is ESC (unprintable)
	err = d.SetAttributeSignature(0x1B, '{', '}')
	requireConfigIssue(t, err, IssueUnprintableChar)

	err = d.SetAttributeSignature('!', 0x1B, '}')
	requireConfigIssue(t, err, IssueUnprintableChar)

	err = d.SetAttributeSignature('!', '{', 0x1B)
	requireConfigIssue(t, err, IssueUnprintableChar)
}

func TestActAttribute_TriggerAtEnd(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 10, MaxAttrPayloadLen: 10})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!"
	warns := newWarnings(t)

	_, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.True(t, skip)
	require.Equal(t, 1, stride)

	ws := warns.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueUnexpectedEOL, ws[0].Issue)
	require.Equal(t, 0, ws[0].Pos)
}

func TestActAttribute_NoPayloadStart_EOF(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 100, MaxAttrPayloadLen: 10})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!abc" // no '{' and we reach EOL
	warns := newWarnings(t)

	_, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.True(t, skip)
	require.Equal(t, 1, stride)

	ws := warns.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueUnexpectedEOL, ws[0].Issue)
	require.Equal(t, len(input), ws[0].Pos)
}

func TestActAttribute_KeyTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 2, MaxAttrPayloadLen: 100})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!abcd{v}" // key "abcd" is longer than 2
	warns := newWarnings(t)

	_, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.True(t, skip)
	require.Equal(t, 1, stride)

	ws := warns.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueAttrKeyTooLong, ws[0].Issue)
	require.Equal(t, 0, ws[0].Pos) // your code uses i
}

func TestActAttribute_PayloadStartAtEnd(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 10, MaxAttrPayloadLen: 10})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!k{"
	warns := newWarnings(t)

	_, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.True(t, skip)
	require.Equal(t, 1, stride)

	ws := warns.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueUnexpectedEOL, ws[0].Issue)
	require.Equal(t, 2, ws[0].Pos) // payloadStartIdx
}

func TestActAttribute_UnclosedPayload_EOF(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 10, MaxAttrPayloadLen: 100})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!k{val" // no closing '}'
	warns := newWarnings(t)

	_, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.True(t, skip)
	require.Equal(t, 1, stride)

	ws := warns.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueUnclosedAttrPayload, ws[0].Issue)
	require.Equal(t, len(input), ws[0].Pos)
}

func TestActAttribute_PayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 10, MaxAttrPayloadLen: 2})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!k{abcd}" // payload "abcd" longer than 2
	warns := newWarnings(t)

	_, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.True(t, skip)
	require.Equal(t, 1, stride)

	ws := warns.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueAttrPayloadTooLong, ws[0].Issue)
	require.Equal(t, 0, ws[0].Pos) // your code uses i
}

func TestActAttribute_EmptyPayload(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 10, MaxAttrPayloadLen: 10})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!k{}"
	warns := newWarnings(t)

	_, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.True(t, skip)
	require.Equal(t, 1, stride)

	ws := warns.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueEmptyAttrPayload, ws[0].Issue)
	// payloadStartIdx is 2 -> payloadStartIdx+1 is 3
	require.Equal(t, 3, ws[0].Pos)
}

func TestActAttribute_Flag_OK(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 10, MaxAttrPayloadLen: 20})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!{IS_FLAG}"
	warns := newWarnings(t)

	tok, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.False(t, skip)
	require.Equal(t, tok.Width, stride)
	require.Equal(t, TokenAttributeFlag, tok.Type)
	require.Equal(t, byte('!'), tok.Trigger)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len(input), tok.Width)
	require.Equal(t, Span{Start: 0, End: len(input)}, tok.Raw)

	// For flags, your implementation stores the name in Payload.
	require.Equal(t, "IS_FLAG", spanStr(input, tok.Payload))

	require.Empty(t, warns.List())
}

func TestActAttribute_KV_OK(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 10, MaxAttrPayloadLen: 64})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!url{https://google.com}"
	warns := newWarnings(t)

	tok, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.False(t, skip)
	require.Equal(t, tok.Width, stride)
	require.Equal(t, TokenAttributeKV, tok.Type)
	require.Equal(t, byte('!'), tok.Trigger)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, Span{Start: 0, End: len(input)}, tok.Raw)

	require.Equal(t, "url", spanStr(input, tok.AttrKey))
	require.Equal(t, "https://google.com", spanStr(input, tok.Payload))

	require.Empty(t, warns.List())
}

func TestActAttribute_Boundaries_ExactLimits_OK(t *testing.T) {
	// key length exactly 3, payload length exactly 4
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 3, MaxAttrPayloadLen: 4})
	require.NoError(t, err)
	require.NoError(t, d.SetAttributeSignature('!', '{', '}'))

	input := "!abc{wxyz}"
	warns := newWarnings(t)

	tok, stride, skip := ActAttribute(&d, '!', input, 0, warns)

	require.False(t, skip)
	require.Equal(t, tok.Width, stride)
	require.Equal(t, TokenAttributeKV, tok.Type)
	require.Equal(t, "abc", spanStr(input, tok.AttrKey))
	require.Equal(t, "wxyz", spanStr(input, tok.Payload))

	require.Empty(t, warns.List())
}
