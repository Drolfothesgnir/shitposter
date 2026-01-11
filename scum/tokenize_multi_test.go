package scum

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// initMultiDict registers a set of multi-char tags used by tests.
//
// "**" : universal, non-greedy (e.g., bold markers)
// "~~" : universal, greedy (e.g., strikethrough)
// "```" : universal, grasping (e.g., code blocks)
// "[[" : opening tag
// "]]" : closing tag
func initMultiDict(t *testing.T) *Dictionary {
	t.Helper()

	d, err := NewDictionary(Limits{})
	require.NoError(t, err)
	dp := &d

	mustAddTag(t, dp, "BOLD", "**", NonGreedy, RuleNA, '*', '*')
	mustAddTag(t, dp, "STRIKE", "~~", Greedy, RuleNA, '~', '~')
	mustAddTag(t, dp, "CODE_BLOCK", "```", Grasping, RuleNA, '`', '`')

	mustAddTag(t, dp, "WIKILINK_START", "[[", NonGreedy, RuleNA, 0, ']')
	mustAddTag(t, dp, "WIKILINK_END", "]]", NonGreedy, RuleNA, '[', 0)

	return dp
}

// ---------- multi-char universal non-greedy tests ----------

func TestTokenizeMulti_UniversalNonGreedy_ValidSequence(t *testing.T) {
	d := initMultiDict(t)
	in := "****" // Two consecutive ** pairs

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)
	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")
	require.Len(t, warns.List(), 0, "expected no warnings for valid sequences")

	// Should have 2 ** tag tokens
	tagCount := 0
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '*' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			require.Equal(t, "**", raw, "expected ** tag")
			tagCount++
		}
	}
	require.Equal(t, 2, tagCount, "expected 2 ** tag tokens")
}

func TestTokenizeMulti_UniversalNonGreedy_InvalidSeqWarnsAndEmitsPartial(t *testing.T) {
	d := initMultiDict(t)
	// When sequence is invalid (e.g., *b instead of **), only the valid prefix is emitted
	in := "*bold*"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Warnings expected for invalid sequences
	require.True(t, hasIssue(&warns, IssueUnexpectedSymbol), "expected IssueUnexpectedSymbol warnings")

	// Should have single-char tag tokens since ** sequence is broken
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '*' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			require.Equal(t, "*", raw, "expected single * when sequence is invalid")
		}
	}
}

func TestTokenizeMulti_UniversalNonGreedy_SingleCharBecomesText(t *testing.T) {
	d := initMultiDict(t)
	in := "a * b * c"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Single * should become text, not a tag
	for _, tok := range toks {
		if tok.Type == TokenTag {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			require.NotEqual(t, "*", raw, "single * should be text, not tag")
		}
	}
}

func TestTokenizeMulti_UniversalNonGreedy_InvalidSeqWarns(t *testing.T) {
	d := initMultiDict(t)
	in := "*x rest"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Should have warning for invalid sequence
	require.True(t, hasIssue(&warns, IssueUnexpectedSymbol), "expected IssueUnexpectedSymbol, got warnings=%#v", warns)
}

// ---------- multi-char universal greedy tests ----------

func TestTokenizeMulti_UniversalGreedy_StrikethroughCapture(t *testing.T) {
	d := initMultiDict(t)
	in := "some ~~deleted~~ text"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")
	require.Len(t, warns.List(), 0, "expected no warnings")

	// Should have one tag token capturing ~~deleted~~
	found := false
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '~' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			if strings.HasPrefix(raw, "~~") && strings.HasSuffix(raw, "~~") {
				found = true
				require.Equal(t, "~~deleted~~", raw, "expected full strikethrough capture")
			}
		}
	}
	require.True(t, found, "expected a strikethrough tag token")
}

func TestTokenizeMulti_UniversalGreedy_UnclosedBecomesText(t *testing.T) {
	d := initMultiDict(t)
	in := "start ~~unclosed"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Should have warning for unclosed tag
	require.True(t, hasIssue(&warns, IssueUnclosedTag), "expected IssueUnclosedTag, got warnings=%#v", warns)

	// The ~~ should be treated as text since unclosed greedy tags skip
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '~' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			// If it's a tag, it should be a proper closed one
			require.True(t, strings.HasSuffix(raw, "~~"), "greedy unclosed should not emit partial tag")
		}
	}
}

// ---------- multi-char universal grasping tests ----------

func TestTokenizeMulti_UniversalGrasping_CodeBlockCapture(t *testing.T) {
	d := initMultiDict(t)
	in := "text ```code here``` more"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")
	require.Len(t, warns.List(), 0, "expected no warnings")

	found := false
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '`' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			if strings.HasPrefix(raw, "```") && strings.HasSuffix(raw, "```") {
				found = true
				require.Equal(t, "```code here```", raw, "expected full code block capture")
			}
		}
	}
	require.True(t, found, "expected a code block tag token")
}

func TestTokenizeMulti_UniversalGrasping_UnclosedGraspsTillEnd(t *testing.T) {
	d := initMultiDict(t)
	in := "start ```unclosed code"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Should have warning for unclosed tag
	require.True(t, hasIssue(&warns, IssueUnclosedTag), "expected IssueUnclosedTag, got warnings=%#v", warns)

	// Grasping should still emit the tag, capturing everything till the end
	found := false
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '`' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			if strings.HasPrefix(raw, "```") {
				found = true
				require.Equal(t, "```unclosed code", raw, "grasping should capture to end")
			}
		}
	}
	require.True(t, found, "expected grasping tag to capture to end")
}

// ---------- multi-char opening/closing tests ----------

func TestTokenizeMulti_Opening_WikilinkStart(t *testing.T) {
	d := initMultiDict(t)
	in := "[[link]]"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Should have [[ and ]] tags
	openFound, closeFound := false, false
	for _, tok := range toks {
		if tok.Type != TokenTag {
			continue
		}
		raw := in[tok.Pos : tok.Pos+tok.Width]
		if raw == "[[" {
			openFound = true
		}
		if raw == "]]" {
			closeFound = true
		}
	}
	require.True(t, openFound, "expected [[ tag")
	require.True(t, closeFound, "expected ]] tag")
}

func TestTokenizeMulti_Opening_AtEOL_Warns(t *testing.T) {
	d := initMultiDict(t)
	in := "[["

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Opening tag at EOL should warn
	require.True(t, hasIssue(&warns, IssueUnexpectedEOL), "expected IssueUnexpectedEOL, got warnings=%#v", warns)
}

func TestTokenizeMulti_Opening_SingleCharBecomesText(t *testing.T) {
	d := initMultiDict(t)
	in := "[ not a wikilink"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Should warn about invalid sequence
	require.True(t, hasIssue(&warns, IssueUnexpectedSymbol), "expected IssueUnexpectedSymbol, got warnings=%#v", warns)
}

func TestTokenizeMulti_Closing_AtStart_Warns(t *testing.T) {
	d := initMultiDict(t)
	in := "]]hello"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Closing tag at start should warn
	require.True(t, hasIssue(&warns, IssueMisplacedClosingTag), "expected IssueMisplacedClosingTag, got warnings=%#v", warns)
}

func TestTokenizeMulti_Closing_SingleCharBecomesText(t *testing.T) {
	d := initMultiDict(t)
	in := "] not a close"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Should warn about invalid sequence
	require.True(t, hasIssue(&warns, IssueUnexpectedSymbol), "expected IssueUnexpectedSymbol, got warnings=%#v", warns)
}

// ---------- edge cases ----------

func TestTokenizeMulti_MixedSingleAndMulti(t *testing.T) {
	d := initMultiDict(t)
	// Add a single-char tag to test mixing
	mustAddTag(t, d, "UNDERSCORE", "_", NonGreedy, RuleNA, '_', '_')

	in := "**bold** and _underline_"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")
	require.Len(t, warns.List(), 0, "expected no warnings")

	// Should have both ** and _ tag tokens
	starCount, underscoreCount := 0, 0
	for _, tok := range toks {
		if tok.Type != TokenTag {
			continue
		}
		if tok.Trigger == '*' {
			starCount++
		}
		if tok.Trigger == '_' {
			underscoreCount++
		}
	}
	require.Equal(t, 2, starCount, "expected 2 ** tags")
	require.Equal(t, 2, underscoreCount, "expected 2 underscore tags")
}

func TestTokenizeMulti_NestedTags(t *testing.T) {
	d := initMultiDict(t)
	// Test nested tags - greedy inside non-greedy
	in := "~~inside~~"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Should have one greedy tag capturing the full span
	found := false
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '~' {
			found = true
			break
		}
	}
	require.True(t, found, "expected strikethrough tag")
}

func TestTokenizeMulti_ConsecutiveTags(t *testing.T) {
	d := initMultiDict(t)
	in := "****" // Two consecutive ** pairs

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")
	require.Len(t, warns.List(), 0, "expected no warnings")
}

func TestTokenizeMulti_GreedyEmptyContent(t *testing.T) {
	d := initMultiDict(t)
	// Greedy tag with empty content between open and close
	in := "~~~~" // ~~ followed by ~~ (empty content)

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")
	require.Len(t, warns.List(), 0, "expected no warnings")
}

// ---------- fuzz ----------

func FuzzTokenizeMulti_NoPanic_ValidSpans(f *testing.F) {
	seeds := []string{
		"",
		"plain text",
		"**bold**",
		"~~strike~~",
		"```code```",
		"[[link]]",
		"**unclosed",
		"~~unclosed",
		"```unclosed",
		"[[",
		"]]",
		"*",
		"~",
		"`",
		"[",
		"]",
		"****",
		"~~~~~~",
		"``````",
		"[[[[]]]]",
		"**nested ~~tags~~ here**",
		"mix **of** ~~different~~ ```tags```",
		"тест**юникод**",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		d := initMultiDict(t)
		toks, warns := tokenize(t, d, in)

		assertTokenInvariants(t, in, toks)
		assertWarningInvariants(t, in, &warns)

		require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch; toks=%#v warns=%#v", toks, warns)
	})
}
