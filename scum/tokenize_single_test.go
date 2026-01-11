package scum

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------- dictionary fixture ----------

func mustAddTag(t *testing.T, d *Dictionary, name string, seq string, greed Greed, rule Rule, openID, closeID byte) {
	t.Helper()
	err := d.AddTag(name, []byte(seq), greed, rule, openID, closeID)
	require.NoError(t, err, "AddTag(%q,%q) failed", name, seq)
}

// initSingleDict registers a small, intentional set of single-char tags used by tests.
//
// '_' : universal, non-greedy, infra-word rule
// '`' : universal, greedy, tag-vs-content rule (we'll use triple backticks in inputs)
// '[' : opening tag
// ']' : closing tag
func initSingleDict(t *testing.T) *Dictionary {
	t.Helper()

	d := &Dictionary{}

	mustAddTag(t, d, "UNDERLINE", "_", NonGreedy, RuleInfraWord, '_', '_')
	mustAddTag(t, d, "CODE", "`", Greedy, RuleTagVsContent, '`', '`')

	mustAddTag(t, d, "LINK_TEXT_START", "[", NonGreedy, RuleNA, 0, ']')
	mustAddTag(t, d, "LINK_TEXT_END", "]", NonGreedy, RuleNA, '[', 0)

	return d
}

func tokenize(t *testing.T, d *Dictionary, in string) ([]Token, Warnings) {
	t.Helper()

	warns, err := NewWarnings(WarnOverflowNoCap, 1)
	require.NoError(t, err)

	toks := Tokenize(d, in, &warns)
	return toks, warns
}

// ---------- warning helpers ----------

func hasIssue(warns *Warnings, issue Issue) bool {
	for _, w := range warns.List() {
		if w.Issue == issue {
			return true
		}
	}
	return false
}

func assertWarningInvariants(t *testing.T, in string, warns *Warnings) {
	t.Helper()
	n := len(in)

	// Pos is a byte index "at which the problem occurred".
	// For EOL-like issues, allowing Pos == len(input) is reasonable.
	for i, w := range warns.List() {
		require.GreaterOrEqualf(t, w.Pos, 0, "warning[%d] pos < 0: %#v input=%q", i, w, in)
		require.LessOrEqualf(t, w.Pos, n, "warning[%d] pos > len(input): %#v input=%q", i, w, in)
	}
}

// ---------- token invariants ----------

func assertTokenInvariants(t *testing.T, in string, toks []Token) {
	t.Helper()
	n := len(in)
	prevEnd := 0

	for i, tok := range toks {
		// Inner bounds
		require.GreaterOrEqualf(t, tok.Payload.Start, 0, "token[%d] inner.start < 0: %#v input=%q", i, tok, in)
		require.GreaterOrEqualf(t, tok.Payload.End, 0, "token[%d] inner.end < 0: %#v input=%q", i, tok, in)
		require.LessOrEqualf(t, tok.Payload.Start, tok.Payload.End, "token[%d] inner.start > inner.end: %#v input=%q", i, tok, in)
		require.LessOrEqualf(t, tok.Payload.End, n, "token[%d] inner.end > len(input): %#v input=%q", i, tok, in)

		// Tokenize() produces contiguous coverage
		require.Equalf(t, prevEnd, tok.Pos, "token[%d] non-contiguous: prevEnd=%d token=%#v input=%q", i, prevEnd, tok, in)
		prevEnd = tok.Pos + tok.Width

		// Text token convention: Raw == Inner
		if tok.Type == TokenText {
			require.Equalf(t, NewSpan(tok.Pos, tok.Width), tok.Payload, "token[%d] text token raw!=inner: %#v input=%q", i, tok, in)
		}
	}

	require.Equalf(t, n, prevEnd, "tokens do not cover input: covered=%d n=%d input=%q toks=%#v", prevEnd, n, in, toks)
}

func sliceByRaw(in string, toks []Token) string {
	var b strings.Builder
	for _, tok := range toks {
		b.WriteString(in[tok.Pos : tok.Pos+tok.Width])
	}
	return b.String()
}

// ---------- single-char grasping tests ----------

func TestTokenizeSingle_Grasping_CapturesEvenWhenUnclosed(t *testing.T) {
	d := &Dictionary{}
	// Single-char grasping tag with RuleNA (tests addCloseTagCheck else branch)
	mustAddTag(t, d, "EMPHASIS", "*", Grasping, RuleNA, '*', '*')

	in := "*unclosed text"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Grasping emits tag even when unclosed, but warns
	require.True(t, hasIssue(&warns, IssueUnclosedTag), "expected IssueUnclosedTag warning")

	// Should have captured from * to end
	found := false
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '*' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			require.Equal(t, "*unclosed text", raw, "grasping should capture to end")
			found = true
		}
	}
	require.True(t, found, "expected grasping tag token")
}

func TestTokenizeSingle_Grasping_ClosedNormally(t *testing.T) {
	d := &Dictionary{}
	mustAddTag(t, d, "EMPHASIS", "*", Grasping, RuleNA, '*', '*')

	in := "*bold* rest"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")
	require.Len(t, warns.List(), 0, "expected no warnings for properly closed tag")

	// Should have captured *bold*
	found := false
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '*' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			require.Equal(t, "*bold*", raw, "should capture closed tag")
			found = true
		}
	}
	require.True(t, found, "expected grasping tag token")
}

// ---------- single-char greedy with RuleNA (else branch of addCloseTagCheck) ----------

func TestTokenizeSingle_Greedy_RuleNA_ClosedNormally(t *testing.T) {
	d := &Dictionary{}
	// Greedy tag with RuleNA tests the else branch in addCloseTagCheck
	mustAddTag(t, d, "STAR", "*", Greedy, RuleNA, '*', '*')

	in := "*content* more"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")
	require.Len(t, warns.List(), 0, "expected no warnings")

	found := false
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '*' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			require.Equal(t, "*content*", raw, "should capture closed tag")
			found = true
		}
	}
	require.True(t, found, "expected greedy tag token")
}

func TestTokenizeSingle_Greedy_RuleNA_UnclosedBecomesText(t *testing.T) {
	d := &Dictionary{}
	mustAddTag(t, d, "STAR", "*", Greedy, RuleNA, '*', '*')

	in := "*unclosed"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Greedy skips unclosed tags (unlike grasping which captures anyway)
	require.True(t, hasIssue(&warns, IssueUnclosedTag), "expected IssueUnclosedTag warning")

	// The * should be treated as text since greedy unclosed tags skip
	for _, tok := range toks {
		if tok.Type == TokenTag && tok.Trigger == '*' {
			raw := in[tok.Pos : tok.Pos+tok.Width]
			// If any tag is emitted, it must be properly closed
			require.True(t, strings.HasSuffix(raw, "*") && len(raw) > 1,
				"greedy unclosed should not emit partial tag")
		}
	}
}

// ---------- edge cases ----------

func TestTokenizeSingle_InfraWord_UnderscoreInsideWordIsText(t *testing.T) {
	d := initSingleDict(t)
	in := "image_from_.png"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// Usually no warnings here.
	require.Len(t, warns.List(), 0, "expected no warnings")
}

func TestTokenizeSingle_Greedy_TagVsContent_TripleBackticksCapture(t *testing.T) {
	d := initSingleDict(t)
	in := "here is ```const rawStr = `hello`;``` ok"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	found := false
	for _, tok := range toks {
		if tok.Type != TokenTag || tok.Trigger != '`' {
			continue
		}
		raw := in[tok.Pos : tok.Pos+tok.Width]
		if strings.HasPrefix(raw, "```") && strings.HasSuffix(raw, "```") && len(raw) >= 6 {
			found = true
			break
		}
	}
	require.True(t, found, "expected a CODE Tag token spanning triple backticks; toks=%#v", toks)
	require.Len(t, warns.List(), 0, "expected no warnings")
}

func TestTokenizeSingle_Greedy_UnclosedBecomesTextAndWarns(t *testing.T) {
	d := initSingleDict(t)
	in := "start ```abc"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	require.True(t, hasIssue(&warns, IssueUnclosedTag), "expected IssueUnclosedTag, got warnings=%#v", warns)
}

func TestTokenizeSingle_OpeningBeforeEOL_WarnsUnexpectedEOL(t *testing.T) {
	d := initSingleDict(t)
	in := "["

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// If your engine uses a different Issue for this case, change it here.
	require.True(t, hasIssue(&warns, IssueUnexpectedEOL), "expected IssueUnexpectedEOL, got warnings=%#v", warns)
}

func TestTokenizeSingle_ClosingAtStart_WarnsMisplacedClosingTag(t *testing.T) {
	d := initSingleDict(t)
	in := "]hello"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, &warns)

	require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch")

	// If your engine uses a different Issue for this case, change it here.
	require.True(t, hasIssue(&warns, IssueMisplacedClosingTag), "expected IssueMisplacedClosingTag, got warnings=%#v", warns)
}

// ---------- fuzz ----------

func FuzzTokenizeSingle_NoPanic_ValidSpans(f *testing.F) {
	seeds := []string{
		"",
		"plain",
		"image_from_.png",
		"_hello__",
		"here is ```const rawStr = `hello`;``` ok",
		"start ```abc",
		"[",
		"]hello",
		"___",
		"``````",
		"тест_юникод_",
		"[][][]",
		"]]]",
		"[[[",
		"mix _with_ words and ```code``` and ] closers",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		d := initSingleDict(t)
		toks, warns := tokenize(t, d, in)

		assertTokenInvariants(t, in, toks)
		assertWarningInvariants(t, in, &warns)

		// Tokenize is designed to be lossless.
		require.Equal(t, in, sliceByRaw(in, toks), "round-trip mismatch; toks=%#v warns=%#v", toks, warns)
	})
}
