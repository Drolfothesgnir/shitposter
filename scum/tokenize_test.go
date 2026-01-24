package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requireTokenizeInvariants(t *testing.T, inp string, out TokenizerOutput) {
	t.Helper()

	toks := out.Tokens

	end := 0
	textLen := 0
	textTokens := 0

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
			textLen += tok.Width
			textTokens++
		}
	}

	require.Equal(t, len(inp), end, "tokens do not cover full input")

	// Verify TokenizerOutput counters
	require.Equal(t, textLen, out.TextLen, "TextLen mismatch")
	require.Equal(t, textTokens, out.TextTokens, "TextTokens mismatch")
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

	out := Tokenize(&d, inp, w)

	require.Len(t, out.Tokens, 5)
	require.Equal(t, expected, out.Tokens)
	require.Len(t, w.List(), 0)

	// Verify state counters
	require.Equal(t, 4, out.TextLen)
	require.Equal(t, 1, out.TextTokens)
	require.Equal(t, 4, out.TagsTotal)
	require.Equal(t, 4, out.UniversalTags)
	require.Equal(t, 0, out.Attributes)
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

	out := Tokenize(&d, inp, w)
	require.Len(t, out.Tokens, 6)
	require.Equal(t, expected, out.Tokens)
	require.Len(t, w.List(), 0)

	// Verify state counters
	require.Equal(t, 21, out.TextLen)
	require.Equal(t, 1, out.TextTokens)
	require.Equal(t, 4, out.TagsTotal)
	require.Equal(t, 2, out.UniversalTags) // $$ tags
	require.Equal(t, 1, out.OpenTags)      // [
	require.Equal(t, 1, out.CloseTags)     // ]
	require.Equal(t, 1, out.Attributes)    // !URL{...}
}

// GPT-generated test

func TestTokenize_EscapeBeforeSpecial_TagIsNotTriggered(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `\*hi*`
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	// Escaped '*' at position 1 should NOT be a tag
	for _, tok := range out.Tokens {
		if tok.Pos == 1 {
			require.NotEqual(t, TokenTag, tok.Type, "escaped '*' must not produce tag token")
		}
	}

	// No RedundantEscape warning since '*' is special
	for _, ww := range w.List() {
		require.NotEqual(t, IssueRedundantEscape, ww.Issue)
	}
}

func TestTokenize_RedundantEscape_BeforeNonSpecial(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `\a`
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	ws := w.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueRedundantEscape, ws[0].Issue)
	require.Equal(t, 0, ws[0].Pos)
}

func TestTokenize_EscapeAtEOL_WarnsUnexpectedEOL(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `\`
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	ws := w.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueUnexpectedEOL, ws[0].Issue)
	require.Equal(t, len(inp)-1, ws[0].Pos)
}

func TestTokenize_AttributePayload_EscapedEndBrace(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `!k{a\}}`
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	require.Len(t, out.Tokens, 1)
	tok := out.Tokens[0]
	require.Equal(t, TokenAttributeKV, tok.Type)
	require.Equal(t, byte('!'), tok.Trigger)
	require.Equal(t, 0, tok.Pos)
	require.Equal(t, len(inp), tok.Width)

	require.Equal(t, "k", spanStr(inp, tok.AttrKey))
	require.Equal(t, `a\}`, spanStr(inp, tok.Payload)) // raw payload includes backslash

	// Verify state
	require.Equal(t, 1, out.Attributes)
	require.Equal(t, 0, out.TextLen)
}

func TestTokenize_EscapedAttributeTrigger_DoesNotStartAttribute(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := `\!k{v}`
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	// Escaped '!' should NOT produce attribute tokens
	for _, tok := range out.Tokens {
		require.NotEqual(t, TokenAttributeKV, tok.Type)
		require.NotEqual(t, TokenAttributeFlag, tok.Type)
	}

	// No RedundantEscape warning since '!' is special
	for _, ww := range w.List() {
		require.NotEqual(t, IssueRedundantEscape, ww.Issue)
	}

	// Verify state
	require.Equal(t, 0, out.Attributes)
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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	// The '!' should be treated as plain text since key limit was reached
	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenText, out.Tokens[0].Type)

	ws := w.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueAttrKeyTooLong, ws[0].Issue)
	require.Equal(t, 0, ws[0].Pos)

	require.Equal(t, 0, out.Attributes)
}

func TestTokenize_AttrKeyExactlyAtLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrKeyLen: 3})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Key "abc" is exactly 3 bytes, should work
	inp := "!abc{value}"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenAttributeKV, out.Tokens[0].Type)
	require.Equal(t, "abc", spanStr(inp, out.Tokens[0].AttrKey))
	require.Equal(t, "value", spanStr(inp, out.Tokens[0].Payload))
	require.Empty(t, w.List())

	require.Equal(t, 1, out.Attributes)
}

func TestTokenize_AttrPayloadTooLong(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrPayloadLen: 5})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload "toolongvalue" exceeds the 5-byte limit
	inp := "!k{toolongvalue}"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	// The '!' should be treated as plain text since payload limit was reached
	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenText, out.Tokens[0].Type)

	ws := w.List()
	require.Len(t, ws, 1)
	require.Equal(t, IssueAttrPayloadTooLong, ws[0].Issue)
	require.Equal(t, 0, ws[0].Pos)

	require.Equal(t, 0, out.Attributes)
}

func TestTokenize_AttrPayloadExactlyAtLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrPayloadLen: 5})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Payload "12345" is exactly 5 bytes, should work
	inp := "!k{12345}"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenAttributeKV, out.Tokens[0].Type)
	require.Equal(t, "k", spanStr(inp, out.Tokens[0].AttrKey))
	require.Equal(t, "12345", spanStr(inp, out.Tokens[0].Payload))
	require.Empty(t, w.List())

	require.Equal(t, 1, out.Attributes)
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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenTag, out.Tokens[0].Type)
	require.Equal(t, byte('('), out.Tokens[0].Trigger)
	require.Equal(t, "12345", spanStr(inp, out.Tokens[0].Payload))
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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	// Grasping tag should still produce a tag token even when limit reached
	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenTag, out.Tokens[0].Type)
	require.Equal(t, byte('('), out.Tokens[0].Trigger)

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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenTag, out.Tokens[0].Type)
	require.Equal(t, byte('`'), out.Tokens[0].Trigger)
	require.Equal(t, "code", spanStr(inp, out.Tokens[0].Payload))
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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenTag, out.Tokens[0].Type)
	require.Equal(t, byte('`'), out.Tokens[0].Trigger)
	require.Equal(t, "12345", spanStr(inp, out.Tokens[0].Payload))
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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenTag, out.Tokens[0].Type)
	require.Equal(t, "shorturl", spanStr(inp, out.Tokens[0].Payload))
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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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

	require.Equal(t, 0, out.Attributes)
}

func TestTokenize_AttrFlagPayloadExactlyAtLimit(t *testing.T) {
	d, err := NewDictionary(Limits{MaxAttrPayloadLen: 4})
	require.NoError(t, err)

	err = d.SetAttributeSignature('!', '{', '}')
	require.NoError(t, err)

	w := newWarnings(t)

	// Flag attribute payload exactly at limit
	inp := "!{flag}"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenAttributeFlag, out.Tokens[0].Type)
	require.Equal(t, "flag", spanStr(inp, out.Tokens[0].Payload))
	require.Empty(t, w.List())

	require.Equal(t, 1, out.Attributes)
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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)

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

// --- TokenizerState/TokenizerOutput counter tests ---

func TestTokenizerOutput_PlainTextOnly(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := "hello world"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenText, out.Tokens[0].Type)

	// All counters should reflect plain text only
	require.Equal(t, 11, out.TextLen)
	require.Equal(t, 1, out.TextTokens)
	require.Equal(t, 0, out.TagsTotal)
	require.Equal(t, 0, out.UniversalTags)
	require.Equal(t, 0, out.OpenTags)
	require.Equal(t, 0, out.CloseTags)
	require.Equal(t, 0, out.Attributes)
}

func TestTokenizerOutput_EmptyInput(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	inp := ""
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	require.Len(t, out.Tokens, 0)

	// All counters should be zero
	require.Equal(t, 0, out.TextLen)
	require.Equal(t, 0, out.TextTokens)
	require.Equal(t, 0, out.TagsTotal)
	require.Equal(t, 0, out.UniversalTags)
	require.Equal(t, 0, out.OpenTags)
	require.Equal(t, 0, out.CloseTags)
	require.Equal(t, 0, out.Attributes)
}

func TestTokenizerOutput_MixedContent(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	// "hello $$bold$$ world" - text + universal tags + text
	inp := "hello $$bold$$ world"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	// Tokens: text("hello "), tag($$), text("bold"), tag($$), text(" world")
	require.Len(t, out.Tokens, 5)

	// TextLen = "hello " (6) + "bold" (4) + " world" (6) = 16
	require.Equal(t, 16, out.TextLen)
	require.Equal(t, 3, out.TextTokens)
	require.Equal(t, 2, out.TagsTotal)
	require.Equal(t, 2, out.UniversalTags)
	require.Equal(t, 0, out.OpenTags)
	require.Equal(t, 0, out.CloseTags)
	require.Equal(t, 0, out.Attributes)
}

func TestTokenizerOutput_OpenCloseTags(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	// "[link]" - open tag + text + close tag
	inp := "[link]"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	// Tokens: tag([), text("link"), tag(])
	require.Len(t, out.Tokens, 3)

	require.Equal(t, 4, out.TextLen)
	require.Equal(t, 1, out.TextTokens)
	require.Equal(t, 2, out.TagsTotal)
	require.Equal(t, 0, out.UniversalTags)
	require.Equal(t, 1, out.OpenTags)
	require.Equal(t, 1, out.CloseTags)
	require.Equal(t, 0, out.Attributes)
}

func TestTokenizerOutput_MultipleAttributes(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	// Two attributes in a row
	inp := "!a{1}!b{2}"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	require.Len(t, out.Tokens, 2)
	require.Equal(t, TokenAttributeKV, out.Tokens[0].Type)
	require.Equal(t, TokenAttributeKV, out.Tokens[1].Type)

	require.Equal(t, 0, out.TextLen)
	require.Equal(t, 0, out.TextTokens)
	require.Equal(t, 0, out.TagsTotal)
	require.Equal(t, 2, out.Attributes)
}

func TestTokenizerOutput_GreedyTagWithPayload(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	// Code tag with payload (greedy RuleTagVsContent)
	inp := "`code`"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenTag, out.Tokens[0].Type)
	require.Equal(t, "code", spanStr(inp, out.Tokens[0].Payload))

	require.Equal(t, 0, out.TextLen)
	require.Equal(t, 0, out.TextTokens)
	// Greedy RuleTagVsContent tags don't increment TagsTotal through normal path
	require.Equal(t, 0, out.Attributes)
}

func TestTokenizerOutput_ComplexDocument(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	// Complex document with multiple tag types and attributes
	inp := "Hello [$$world$$]!url{http://x}"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	// Tokens: text("Hello "), tag([), tag($$), text("world"), tag($$), tag(]), attr(!url{...})
	require.Len(t, out.Tokens, 7)

	// TextLen = "Hello " (6) + "world" (5) = 11
	require.Equal(t, 11, out.TextLen)
	require.Equal(t, 2, out.TextTokens)
	require.Equal(t, 4, out.TagsTotal)     // [, $$, $$, ]
	require.Equal(t, 2, out.UniversalTags) // $$ x2
	require.Equal(t, 1, out.OpenTags)      // [
	require.Equal(t, 1, out.CloseTags)     // ]
	require.Equal(t, 1, out.Attributes)    // !url{...}
}

func TestTokenizerOutput_NestedTags(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	// Nested formatting: bold containing italic
	inp := "$$*nested*$$"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	// Tokens: tag($$), tag(*), text("nested"), tag(*), tag($$)
	require.Len(t, out.Tokens, 5)

	require.Equal(t, 6, out.TextLen)
	require.Equal(t, 1, out.TextTokens)
	require.Equal(t, 4, out.TagsTotal)
	require.Equal(t, 4, out.UniversalTags) // $$, *, *, $$
	require.Equal(t, 0, out.OpenTags)
	require.Equal(t, 0, out.CloseTags)
	require.Equal(t, 0, out.Attributes)
}

func TestTokenizerOutput_OnlyTagsNoText(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	// Only tags, no text content
	inp := "$$$$"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	// Two universal tags
	require.Len(t, out.Tokens, 2)
	require.Equal(t, TokenTag, out.Tokens[0].Type)
	require.Equal(t, TokenTag, out.Tokens[1].Type)

	require.Equal(t, 0, out.TextLen)
	require.Equal(t, 0, out.TextTokens)
	require.Equal(t, 2, out.TagsTotal)
	require.Equal(t, 2, out.UniversalTags)
	require.Equal(t, 0, out.OpenTags)
	require.Equal(t, 0, out.CloseTags)
	require.Equal(t, 0, out.Attributes)
}

// --- Greedy tag counting tests ---
// Greedy tags should only increment TagsTotal, NOT UniversalTags/OpenTags/CloseTags.

func TestTokenizerOutput_SingleCharGreedyTag_NotCountedInUniversalTags(t *testing.T) {
	d := testDict(t)
	w := newWarnings(t)

	// Single-char greedy universal tag (CODE with RuleTagVsContent)
	inp := "`code`"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	require.Len(t, out.Tokens, 1)
	require.Equal(t, TokenTag, out.Tokens[0].Type)

	// Greedy tag should increment TagsTotal but NOT UniversalTags
	require.Equal(t, 1, out.TagsTotal)
	require.Equal(t, 0, out.UniversalTags, "greedy tags should not be counted in UniversalTags")
	require.Equal(t, 0, out.OpenTags)
	require.Equal(t, 0, out.CloseTags)
}

func TestTokenizerOutput_MultiCharGreedyTag_NotCountedInUniversalTags(t *testing.T) {
	d := initMultiDict(t)
	w := newWarnings(t)

	// Multi-char greedy universal tag (STRIKE ~~)
	inp := "~~strikethrough~~"
	out := Tokenize(d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	// Greedy tag should increment TagsTotal but NOT UniversalTags
	require.Equal(t, 1, out.TagsTotal)
	require.Equal(t, 0, out.UniversalTags, "greedy tags should not be counted in UniversalTags")
	require.Equal(t, 0, out.OpenTags)
	require.Equal(t, 0, out.CloseTags)
}

func TestTokenizerOutput_MultiCharGraspingTag_NotCountedInUniversalTags(t *testing.T) {
	d := initMultiDict(t)
	w := newWarnings(t)

	// Multi-char grasping universal tag (CODE_BLOCK ```)
	inp := "```code block```"
	out := Tokenize(d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	// Grasping tag should increment TagsTotal but NOT UniversalTags
	require.Equal(t, 1, out.TagsTotal)
	require.Equal(t, 0, out.UniversalTags, "grasping tags should not be counted in UniversalTags")
	require.Equal(t, 0, out.OpenTags)
	require.Equal(t, 0, out.CloseTags)
}

func TestTokenizerOutput_MixedGreedyAndNonGreedy(t *testing.T) {
	d := initMultiDict(t)
	w := newWarnings(t)

	// Mix of non-greedy (**) and greedy (~~) tags
	inp := "**bold** ~~strike~~"
	out := Tokenize(d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	// 2 non-greedy ** tags + 1 greedy ~~ tag = 3 TagsTotal
	// Only the 2 ** tags should be in UniversalTags
	require.Equal(t, 3, out.TagsTotal)
	require.Equal(t, 2, out.UniversalTags, "only non-greedy universal tags should be counted")
	require.Equal(t, 0, out.OpenTags)
	require.Equal(t, 0, out.CloseTags)
}

func TestTokenizerOutput_SingleCharGreedyOpeningTag_NotCountedInOpenTags(t *testing.T) {
	d, err := NewDictionary(Limits{})
	require.NoError(t, err)

	// Single-char greedy opening tag
	err = d.AddTag("PAREN_START", []byte("("), Greedy, RuleNA, 0, ')')
	require.NoError(t, err)
	err = d.AddTag("PAREN_END", []byte(")"), NonGreedy, RuleNA, '(', 0)
	require.NoError(t, err)

	w := newWarnings(t)

	inp := "(content)"
	out := Tokenize(&d, inp, w)

	requireTokenizeInvariants(t, inp, out)
	require.Empty(t, w.List())

	// Greedy opening tag consumes the content and closing tag as a single token
	require.Len(t, out.Tokens, 1)
	require.Equal(t, 1, out.TagsTotal)
	require.Equal(t, 0, out.UniversalTags)
	require.Equal(t, 0, out.OpenTags, "greedy opening tags should not be counted in OpenTags")
	require.Equal(t, 0, out.CloseTags)
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
			out := Tokenize(&d, inp, w)
			requireTokenizeInvariants(t, inp, out)
		})
	})
}
