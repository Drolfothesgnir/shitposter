package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requireTokenizeInvariants(t *testing.T, inp string, toks []Token) {
	t.Helper()

	end := 0
	for i, tok := range toks {
		require.GreaterOrEqual(t, tok.Width, 1, "token %d has zero width", i)
		require.GreaterOrEqual(t, tok.Pos, 0, "token %d pos<0", i)
		require.LessOrEqual(t, tok.Pos+tok.Width, len(inp), "token %d out of bounds", i)

		// No gaps / no overlaps
		require.Equal(t, end, tok.Pos, "gap/overlap at token %d", i)
		end = tok.Pos + tok.Width

		// Spans valid
		require.LessOrEqual(t, tok.Payload.Start, tok.Payload.End)
		require.LessOrEqual(t, tok.AttrKey.Start, tok.AttrKey.End)
		require.LessOrEqual(t, tok.Payload.End, len(inp))
		require.LessOrEqual(t, tok.AttrKey.End, len(inp))

		// Text payload equals raw span by contract
		if tok.Type == TokenText {
			require.Equal(t, Span{Start: tok.Pos, End: tok.Pos + tok.Width}, tok.Payload)
		}
	}

	require.Equal(t, len(inp), end, "tokens do not cover full input")
}

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

// GPT-generated test

func TestTokenize_EscapeBeforeSpecial_TagIsNotTriggered(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `\*hi*`
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	// Ожидаем: escape sequence, затем текст "*hi", затем тег '*' (закрывающий/универсальный)
	// Точное разбиение текста у тебя может отличаться, поэтому проверим ключевое:
	// - на позиции 1 НЕ должно быть TokenTag '*'
	for _, tok := range toks {
		if tok.Pos == 1 {
			require.NotEqual(t, TokenTag, tok.Type, "escaped '*' must not produce tag token")
		}
	}

	// И warning не должен быть RedundantEscape, потому что '*' special
	for _, ww := range w.List() {
		require.NotEqual(t, IssueRedundantEscape, ww.Issue)
	}
}

func TestTokenize_RedundantEscape_BeforeNonSpecial(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `\a`
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	ws := w.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueRedundantEscape, ws[0].Issue)
	require.Equal(t, 0, ws[0].Pos)
}

func TestTokenize_EscapeAtEOL_WarnsUnexpectedEOL(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `\`
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	ws := w.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueUnexpectedEOL, ws[0].Issue)
	require.Equal(t, len(inp)-1, ws[0].Pos)
}

func TestTokenize_AttributePayload_EscapedEndBrace(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `!k{a\}}`
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)
	require.Empty(t, w.List())

	require.Len(t, toks, 1)
	tok := toks[0]
	require.Equal(t, TokenAttributeKV, tok.Type)
	require.Equal(t, byte('!'), tok.Trigger)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len(inp), tok.Width)

	require.Equal(t, "k", spanStr(inp, tok.AttrKey))
	require.Equal(t, `a\}`, spanStr(inp, tok.Payload)) // raw payload includes backslash
}

func TestTokenize_EscapedAttributeTrigger_DoesNotStartAttribute(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `\!k{v}`
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	// Ключевое: НЕ должно быть TokenAttributeKV/Flag, потому что '!' экранирован.
	for _, tok := range toks {
		require.NotEqual(t, TokenAttributeKV, tok.Type)
		require.NotEqual(t, TokenAttributeFlag, tok.Type)
	}

	// И redundant warning не должен быть, потому что '!' special
	for _, ww := range w.List() {
		require.NotEqual(t, IssueRedundantEscape, ww.Issue)
	}
}

// --- Attribute key/payload limit tests ---

func TestTokenize_AttrKeyTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 3})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Key "toolong" exceeds the 3-byte limit
	inp := "!toolong{value}"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	// The '!' should be treated as plain text since key limit was reached
	require.Len(t, toks, 1)
	require.Equal(t, TokenText, toks[0].Type)

	ws := w.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueAttrKeyTooLong, ws[0].Issue)
	require.Equal(t, 0, ws[0].Pos)
}

func TestTokenize_AttrKeyExactlyAtLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 3})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Key "abc" is exactly 3 bytes, should work
	inp := "!abc{value}"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	require.Len(t, toks, 1)
	require.Equal(t, TokenAttributeKV, toks[0].Type)
	require.Equal(t, "abc", spanStr(inp, toks[0].AttrKey))
	require.Equal(t, "value", spanStr(inp, toks[0].Payload))
	require.Empty(t, w.List())
}

func TestTokenize_AttrPayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrPayloadLen: 5})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload "toolongvalue" exceeds the 5-byte limit
	inp := "!k{toolongvalue}"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	// The '!' should be treated as plain text since payload limit was reached
	require.Len(t, toks, 1)
	require.Equal(t, TokenText, toks[0].Type)

	ws := w.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueAttrPayloadTooLong, ws[0].Issue)
	require.Equal(t, 0, ws[0].Pos)
}

func TestTokenize_AttrPayloadExactlyAtLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrPayloadLen: 5})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload "12345" is exactly 5 bytes, should work
	inp := "!k{12345}"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	require.Len(t, toks, 1)
	require.Equal(t, TokenAttributeKV, toks[0].Type)
	require.Equal(t, "k", spanStr(inp, toks[0].AttrKey))
	require.Equal(t, "12345", spanStr(inp, toks[0].Payload))
	require.Empty(t, w.List())
}

// --- Greedy tag payload limit tests ---

func TestTokenize_GreedyPayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 5})
	require.NoError(t, err)

	// Simple greedy tag (not TagVsContent)
	err = d.AddTag("URL_START", []byte("("), Greedy, RuleNA, 0, ')')
	require.NoError(t, err)
	err = d.AddTag("URL_END", []byte(")"), NonGreedy, RuleNA, '(', 0)
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload exceeds the 5-byte limit, closing tag not visible within limit
	inp := "(toolongurl)"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	// The '(' should be treated as plain text since closing tag not found within limit
	ws := w.List()
	require.NotEmpty(t, ws)

	// Only IssueTagPayloadTooLong (IssueUnclosedTag is suppressed when payload limit reached)
	foundPayloadTooLong := false
	for _, ww := range ws {
		if ww.Issue == IssueTagPayloadTooLong {
			foundPayloadTooLong = true
			break
		}
	}
	require.True(t, foundPayloadTooLong, "expected IssueTagPayloadTooLong warning")
}

func TestTokenize_GreedyPayloadExactlyAtLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 5})
	require.NoError(t, err)

	err = d.AddTag("URL_START", []byte("("), Greedy, RuleNA, 0, ')')
	require.NoError(t, err)
	err = d.AddTag("URL_END", []byte(")"), NonGreedy, RuleNA, '(', 0)
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload "12345" is exactly 5 bytes, should work
	inp := "(12345)"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	require.Len(t, toks, 1)
	require.Equal(t, TokenTag, toks[0].Type)
	require.Equal(t, byte('('), toks[0].Trigger)
	require.Equal(t, "12345", spanStr(inp, toks[0].Payload))
	require.Empty(t, w.List())
}

func TestTokenize_GraspingPayloadTooLong_ConsumesRest(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 5})
	require.NoError(t, err)

	// Grasping tag consumes rest even when limit reached
	err = d.AddTag("URL_START", []byte("("), Grasping, RuleNA, 0, ')')
	require.NoError(t, err)
	err = d.AddTag("URL_END", []byte(")"), NonGreedy, RuleNA, '(', 0)
	require.NoError(t, err)

	w := newWarnings(t)

	// No closing tag, payload exceeds limit, but grasping should still produce tag token
	inp := "(toolongurl"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	// Grasping tag should still produce a tag token even when limit reached
	require.Len(t, toks, 1)
	require.Equal(t, TokenTag, toks[0].Type)
	require.Equal(t, byte('('), toks[0].Trigger)

	// Only IssueTagPayloadTooLong (IssueUnclosedTag is suppressed when payload limit reached)
	ws := w.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueTagPayloadTooLong, ws[0].Issue)
}

// --- Tag-Vs-Content key/payload limit tests ---

func TestTokenize_TagVsContent_KeyTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxKeyLen: 3})
	require.NoError(t, err)

	err = d.AddUniversalTag("CODE", []byte("`"), Greedy, RuleTagVsContent)
	require.NoError(t, err)

	w := newWarnings(t)

	// Opening sequence "````" (4 backticks) exceeds the 3-byte limit
	inp := "````code````"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	// The opening sequence should be treated as plain text
	ws := w.List()
	require.NotEmpty(t, ws)

	found := false
	for _, ww := range ws {
		if ww.Issue == IssueTagKeyTooLong {
			found = true
			break
		}
	}
	require.True(t, found, "expected IssueTagKeyTooLong warning")
}

func TestTokenize_TagVsContent_KeyExactlyAtLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxKeyLen: 3})
	require.NoError(t, err)

	err = d.AddUniversalTag("CODE", []byte("`"), Greedy, RuleTagVsContent)
	require.NoError(t, err)

	w := newWarnings(t)

	// Opening sequence "```" (3 backticks) is exactly at the limit
	inp := "```code```"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	require.Len(t, toks, 1)
	require.Equal(t, TokenTag, toks[0].Type)
	require.Equal(t, byte('`'), toks[0].Trigger)
	require.Equal(t, "code", spanStr(inp, toks[0].Payload))
	require.Empty(t, w.List())
}

func TestTokenize_TagVsContent_PayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 5})
	require.NoError(t, err)

	err = d.AddUniversalTag("CODE", []byte("`"), Greedy, RuleTagVsContent)
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload "toolongcode" exceeds the 5-byte limit
	inp := "`toolongcode`"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	// The tag should be treated as plain text since payload limit was reached
	ws := w.List()
	require.NotEmpty(t, ws)

	found := false
	for _, ww := range ws {
		if ww.Issue == IssueTagPayloadTooLong || ww.Issue == IssueUnclosedTag {
			found = true
			break
		}
	}
	require.True(t, found, "expected IssueTagPayloadTooLong or IssueUnclosedTag warning")
}

func TestTokenize_TagVsContent_PayloadExactlyAtLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 5})
	require.NoError(t, err)

	err = d.AddUniversalTag("CODE", []byte("`"), Greedy, RuleTagVsContent)
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload "12345" is exactly 5 bytes
	inp := "`12345`"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	require.Len(t, toks, 1)
	require.Equal(t, TokenTag, toks[0].Type)
	require.Equal(t, byte('`'), toks[0].Trigger)
	require.Equal(t, "12345", spanStr(inp, toks[0].Payload))
	require.Empty(t, w.List())
}

func TestTokenize_TagVsContent_MultiBacktick_KeyTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxKeyLen: 2})
	require.NoError(t, err)

	err = d.AddUniversalTag("CODE", []byte("`"), Greedy, RuleTagVsContent)
	require.NoError(t, err)

	w := newWarnings(t)

	// Opening sequence "```" (3 backticks) exceeds the 2-byte limit
	inp := "```code```"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	ws := w.List()
	found := false
	for _, ww := range ws {
		if ww.Issue == IssueTagKeyTooLong {
			found = true
			break
		}
	}
	require.True(t, found, "expected IssueTagKeyTooLong warning")
}

// --- Additional edge case tests ---

func TestTokenize_GreedyPayloadTooLong_WarnsPayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 5})
	require.NoError(t, err)

	err = d.AddTag("URL_START", []byte("("), Greedy, RuleNA, 0, ')')
	require.NoError(t, err)
	err = d.AddTag("URL_END", []byte(")"), NonGreedy, RuleNA, '(', 0)
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload exceeds limit, closing tag exists but beyond limit
	inp := "(toolongurl)"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	ws := w.List()
	foundPayloadTooLong := false
	for _, ww := range ws {
		if ww.Issue == IssueTagPayloadTooLong {
			foundPayloadTooLong = true
			break
		}
	}
	require.True(t, foundPayloadTooLong, "expected IssueTagPayloadTooLong warning")
}

func TestTokenize_TagVsContent_PayloadTooLong_WarnsPayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 5})
	require.NoError(t, err)

	err = d.AddUniversalTag("CODE", []byte("`"), Greedy, RuleTagVsContent)
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload exceeds limit, closing tag exists but beyond limit
	inp := "`toolongcode`"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	ws := w.List()
	foundPayloadTooLong := false
	for _, ww := range ws {
		if ww.Issue == IssueTagPayloadTooLong {
			foundPayloadTooLong = true
			break
		}
	}
	require.True(t, foundPayloadTooLong, "expected IssueTagPayloadTooLong warning")
}

func TestTokenize_GraspingPayloadTooLong_WarnsPayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 5})
	require.NoError(t, err)

	err = d.AddTag("URL_START", []byte("("), Grasping, RuleNA, 0, ')')
	require.NoError(t, err)
	err = d.AddTag("URL_END", []byte(")"), NonGreedy, RuleNA, '(', 0)
	require.NoError(t, err)

	w := newWarnings(t)

	// Grasping with no closing tag and payload exceeds limit
	inp := "(toolongurl"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	ws := w.List()
	foundPayloadTooLong := false
	for _, ww := range ws {
		if ww.Issue == IssueTagPayloadTooLong {
			foundPayloadTooLong = true
			break
		}
	}
	require.True(t, foundPayloadTooLong, "expected IssueTagPayloadTooLong warning")
}

func TestTokenize_GreedyPayloadWithinLimit_NoWarning(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 20})
	require.NoError(t, err)

	err = d.AddTag("URL_START", []byte("("), Greedy, RuleNA, 0, ')')
	require.NoError(t, err)
	err = d.AddTag("URL_END", []byte(")"), NonGreedy, RuleNA, '(', 0)
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload within limit
	inp := "(shorturl)"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	require.Len(t, toks, 1)
	require.Equal(t, TokenTag, toks[0].Type)
	require.Equal(t, "shorturl", spanStr(inp, toks[0].Payload))
	require.Empty(t, w.List())
}

func TestTokenize_AttrFlagPayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrPayloadLen: 3})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Flag attribute payload exceeds limit
	inp := "!{toolong}"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	ws := w.List()
	require.NotEmpty(t, ws)

	found := false
	for _, ww := range ws {
		if ww.Issue == IssueAttrPayloadTooLong {
			found = true
			break
		}
	}
	require.True(t, found, "expected IssueAttrPayloadTooLong warning")
}

func TestTokenize_AttrFlagPayloadExactlyAtLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrPayloadLen: 4})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Flag attribute payload exactly at limit
	inp := "!{flag}"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	require.Len(t, toks, 1)
	require.Equal(t, TokenAttributeFlag, toks[0].Type)
	require.Equal(t, "flag", spanStr(inp, toks[0].Payload))
	require.Empty(t, w.List())
}

func TestTokenize_MultiCharGreedyPayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 5})
	require.NoError(t, err)

	// Multi-char greedy tag
	err = d.AddTag("BLOCK_START", []byte("[["), Greedy, RuleNA, 0, ']')
	require.NoError(t, err)
	err = d.AddTag("BLOCK_END", []byte("]]"), NonGreedy, RuleNA, '[', 0)
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload exceeds limit
	inp := "[[toolongcontent]]"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	// Per docs: when payload limit is reached, IssueTagPayloadTooLong warning should be emitted
	ws := w.List()
	foundPayloadTooLong := false
	for _, ww := range ws {
		if ww.Issue == IssueTagPayloadTooLong {
			foundPayloadTooLong = true
			break
		}
	}
	t.Log(ws, len(ws))
	require.True(t, foundPayloadTooLong, "expected IssueTagPayloadTooLong warning")
}

func TestTokenize_TagVsContent_ClosingTagBeyondPayloadLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxPayloadLen: 10})
	require.NoError(t, err)

	err = d.AddUniversalTag("CODE", []byte("`"), Greedy, RuleTagVsContent)
	require.NoError(t, err)

	w := newWarnings(t)

	// Multi-backtick with closing beyond payload limit
	inp := "```this is way too long for the limit```"
	toks := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, toks)

	ws := w.List()
	foundPayloadTooLong := false
	for _, ww := range ws {
		if ww.Issue == IssueTagPayloadTooLong {
			foundPayloadTooLong = true
			break
		}
	}
	require.True(t, foundPayloadTooLong, "expected IssueTagPayloadTooLong warning")
}

func FuzzTokenize_Invariants(f *testing.F) {
	f.Add("")
	f.Add("plain")
	f.Add("$$*text*$$")
	f.Add(`\*`)
	f.Add(`\a`)
	f.Add(`\`)
	f.Add(`!k{a\}}`)
	f.Add("image_from_.png")
	f.Add("`const rawStr = `hello`;`") // tricky for CODE/tag-vs-content

	f.Fuzz(func(t *testing.T, inp string) {
		d := testDict(t)
		w := newWarnings(t)

		require.NotPanics(t, func() {
			toks := Tokenize(&d, inp, w)
			requireTokenizeInvariants(t, inp, toks)
		})
	})
}
