package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func testDict(t *testing.T) Dictionary {
	d, err := NewDictionary(Limits{})
	require.NoError(t, err)

	err = d.AddUniversalTag("BOLD", []byte("$$"), NonGreedy, RuleNA)
	require.NoError(t, err)

	err = d.AddUniversalTag("ITALIC", []byte("*"), NonGreedy, RuleNA)
	require.NoError(t, err)

	err = d.AddUniversalTag("UNDERLINE", []byte("_"), NonGreedy, RuleInfraWord)
	require.NoError(t, err)

	err = d.AddTag("LINK_TEXT_START", []byte("["), NonGreedy, RuleNA, 0, ']')
	require.NoError(t, err)

	err = d.AddTag("IMAGE", []byte(":["), NonGreedy, RuleNA, 0, ']')
	require.NoError(t, err)

	err = d.AddTag("LINK_TEXT_END", []byte("]"), NonGreedy, RuleNA, '\r', 0)
	require.NoError(t, err)

	err = d.AddUniversalTag("CODE", []byte("`"), Greedy, RuleTagVsContent)
	require.NoError(t, err)

	err = d.SetEscapeTrigger('\\')
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	return d
}

func TestBoldItalic(t *testing.T) {
	d := testDict(t)

	w := newWarnings(t)

	inp := "$$*text*$$"

	expected := []Token{
		{
			Type:    TokenTag,
			Trigger: '$',
			Pos:     0,
			Width:   2,
			Payload: NewSpan(2, 0),
		},
		{
			Type:    TokenTag,
			Trigger: '*',
			Pos:     2,
			Width:   1,
			Payload: NewSpan(3, 0),
		},
		{
			Type:    TokenText,
			Pos:     3,
			Width:   4,
			Payload: NewSpan(3, 4),
		},
		{
			Type:    TokenTag,
			Trigger: '*',
			Pos:     7,
			Width:   1,
			Payload: NewSpan(8, 0),
		},
		{
			Type:    TokenTag,
			Trigger: '$',
			Pos:     8,
			Width:   2,
			Payload: NewSpan(10, 0),
		},
	}

	toks := Tokenize(&d, inp, w)

	require.Len(t, toks, 5)
	require.Equal(t, expected, toks)
	require.Len(t, w.List(), 0)
}

func TestLink(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := "[$$link_with_undesrcores$$]!URL{https://google.com}"

	expected := []Token{
		{
			Type:    TokenTag,
			Trigger: '[',
			Pos:     0,
			Width:   1,
			Payload: NewSpan(1, 0),
		},
		{
			Type:    TokenTag,
			Trigger: '$',
			Pos:     1,
			Width:   2,
			Payload: NewSpan(3, 0),
		},
		{
			Type:    TokenText,
			Pos:     3,
			Width:   21,
			Payload: NewSpan(3, 21),
		},
		{
			Type:    TokenTag,
			Trigger: '$',
			Pos:     24,
			Width:   2,
			Payload: NewSpan(26, 0),
		},
		{
			Type:    TokenTag,
			Trigger: ']',
			Pos:     26,
			Width:   1,
			Payload: NewSpan(27, 0),
		},
		{
			Type:    TokenAttributeKV,
			Trigger: '!',
			Pos:     27,
			Width:   24,
			Payload: NewSpan(32, 18),
			AttrKey: NewSpan(28, 3),
		},
	}

	toks := Tokenize(&d, inp, w)
	require.Len(t, toks, 6)
	require.Equal(t, expected, toks)
	require.Len(t, w.List(), 0)
}
