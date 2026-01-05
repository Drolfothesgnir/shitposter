package scum

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetEscapeTrigger_SetsAction(t *testing.T) {
	var d Dictionary

	err := d.SetEscapeTrigger('\\')
	require.NoError(t, err)

	a, ok := d.Action('\\')
	require.True(t, ok)
	require.NotNil(t, a)

	// Smoke-check the action does what we expect.
	in := `\a`
	var warns []Warning
	tok, stride, skip := a(&d, '\\', in, 0, &warns)

	require.False(t, skip)
	require.Equal(t, 2, stride)
	require.Equal(t, TokenEscapeSequence, tok.Type)
	require.Equal(t, byte('\\'), tok.TagID)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, 2, tok.Width)
	require.Equal(t, NewSpan(0, 2), tok.Raw)
	require.Equal(t, NewSpan(1, 1), tok.Inner)
}

func TestSetEscapeTrigger_UnprintableChar(t *testing.T) {
	var d Dictionary
	err := d.SetEscapeTrigger(0x1B)
	require.Error(t, err)

	var ce *ConfigError
	require.ErrorAs(t, err, &ce)
	require.Equal(t, IssueUnprintableChar, ce.Issue)
}

func TestSetEscapeTrigger_DuplicateWithExistingTagID_ReturnsDuplicateError(t *testing.T) {
	var d Dictionary

	// Register a normal tag at the same ID.
	err := d.AddTag("X", []byte{'\\'}, NonGreedy, RuleNA, '\\', '\\')
	require.NoError(t, err)

	err = d.SetEscapeTrigger('\\')
	require.Error(t, err)

	var ce *ConfigError
	require.True(t, errors.As(err, &ce), "expected ConfigError, got: %T (%v)", err, err)
	require.Equal(t, IssueDuplicateTagID, ce.Issue)
}

func TestSetEscapeTrigger_DuplicateEscape_ReturnsDuplicateError(t *testing.T) {
	var d Dictionary

	require.NoError(t, d.SetEscapeTrigger('\\'))

	err := d.SetEscapeTrigger('\\')
	require.Error(t, err)

	var ce *ConfigError
	require.True(t, errors.As(err, &ce), "expected ConfigError, got: %T (%v)", err, err)
	require.Equal(t, IssueDuplicateTagID, ce.Issue)
}

func TestActEscape_EscapeAtEnd_WarnsUnexpectedEOLAndSkips(t *testing.T) {
	var d Dictionary
	require.NoError(t, d.SetEscapeTrigger('\\'))

	in := `\`
	var warns []Warning
	tok, stride, skip := ActEscape(&d, '\\', in, 0, &warns)

	require.True(t, skip)
	require.Equal(t, 1, stride)
	require.Len(t, warns, 1)

	require.Equal(t, IssueUnexpectedEOL, warns[0].Issue)
	require.Equal(t, 0, warns[0].Pos) // you said you changed this to i
	require.NotEmpty(t, warns[0].Description)

	// token is expected to be empty when skip == true (zero-value is fine)
	require.Equal(t, TokenType(0), tok.Type)
}

func TestActEscape_RedundantEscape_WhenNextIsNotSpecial_Warns(t *testing.T) {
	var d Dictionary
	require.NoError(t, d.SetEscapeTrigger('\\'))

	in := `\a`
	var warns []Warning
	tok, stride, skip := ActEscape(&d, '\\', in, 0, &warns)

	require.False(t, skip)
	require.Equal(t, 2, stride)
	require.Equal(t, TokenEscapeSequence, tok.Type)
	require.Equal(t, byte('\\'), tok.TagID)
	require.Equal(t, 2, tok.Width)
	require.Equal(t, NewSpan(0, 2), tok.Raw)
	require.Equal(t, NewSpan(1, 1), tok.Inner)

	require.Len(t, warns, 1)
	require.Equal(t, IssueRedundantEscape, warns[0].Issue)
	require.Equal(t, 1, warns[0].Pos)
	require.NotEmpty(t, warns[0].Description)
}

func TestActEscape_InvalidUTF8Rune(t *testing.T) {
	var d Dictionary
	require.NoError(t, d.SetEscapeTrigger('\\'))

	// invalid UTF-8 byte sequence
	in := string([]byte{'\\', 0xff})

	var warns []Warning
	tok, stride, skip := ActEscape(&d, '\\', in, 0, &warns)

	require.False(t, skip)
	require.Equal(t, 2, stride)
	require.Equal(t, TokenEscapeSequence, tok.Type)
	require.Equal(t, NewSpan(0, 2), tok.Raw)
	require.Equal(t, NewSpan(1, 1), tok.Inner)

	require.Len(t, warns, 1)
	require.Equal(t, IssueRedundantEscape, warns[0].Issue)
}

func TestActEscape_NextIsSpecial_NoRedundantWarning(t *testing.T) {
	var d Dictionary
	require.NoError(t, d.SetEscapeTrigger('\\'))

	// Make '*' special by registering a tag for it.
	require.NoError(t, d.AddTag("STAR", []byte{'*'}, NonGreedy, RuleNA, '*', '*'))

	in := `\*`
	var warns []Warning
	tok, stride, skip := ActEscape(&d, '\\', in, 0, &warns)

	require.False(t, skip)
	require.Equal(t, 2, stride)
	require.Equal(t, TokenEscapeSequence, tok.Type)
	require.Equal(t, NewSpan(0, 2), tok.Raw)
	require.Equal(t, NewSpan(1, 1), tok.Inner)

	require.Len(t, warns, 0)
}

func TestActEscape_MultiByteRune_ConsumesWholeRuneAndWarnsIfNotSpecial(t *testing.T) {
	var d Dictionary
	require.NoError(t, d.SetEscapeTrigger('\\'))

	// "ß" is 2 bytes in UTF-8.
	in := "\\ß"
	var warns []Warning
	tok, stride, skip := ActEscape(&d, '\\', in, 0, &warns)

	require.False(t, skip)
	require.Equal(t, 3, stride) // '\' + 2 bytes
	require.Equal(t, 3, tok.Width)
	require.Equal(t, TokenEscapeSequence, tok.Type)
	require.Equal(t, NewSpan(0, 3), tok.Raw)
	require.Equal(t, NewSpan(1, 2), tok.Inner)

	require.Len(t, warns, 1)
	require.Equal(t, IssueRedundantEscape, warns[0].Issue)
	require.Equal(t, 1, warns[0].Pos)
}
