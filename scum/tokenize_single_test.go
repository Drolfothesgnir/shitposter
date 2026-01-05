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
		// Raw bounds
		require.GreaterOrEqualf(t, tok.Raw.Start, 0, "token[%d] raw.start < 0: %#v input=%q", i, tok, in)
		require.GreaterOrEqualf(t, tok.Raw.End, 0, "token[%d] raw.end < 0: %#v input=%q", i, tok, in)
		require.LessOrEqualf(t, tok.Raw.Start, tok.Raw.End, "token[%d] raw.start > raw.end: %#v input=%q", i, tok, in)
		require.LessOrEqualf(t, tok.Raw.End, n, "token[%d] raw.end > len(input): %#v input=%q", i, tok, in)

		// Inner bounds
		require.GreaterOrEqualf(t, tok.Inner.Start, 0, "token[%d] inner.start < 0: %#v input=%q", i, tok, in)
		require.GreaterOrEqualf(t, tok.Inner.End, 0, "token[%d] inner.end < 0: %#v input=%q", i, tok, in)
		require.LessOrEqualf(t, tok.Inner.Start, tok.Inner.End, "token[%d] inner.start > inner.end: %#v input=%q", i, tok, in)
		require.LessOrEqualf(t, tok.Inner.End, n, "token[%d] inner.end > len(input): %#v input=%q", i, tok, in)

		// Consistency
		require.Equalf(t, tok.Raw.Start, tok.Pos, "token[%d] pos mismatch: token=%#v input=%q", i, tok, in)
		require.Equalf(t, tok.Raw.End-tok.Raw.Start, tok.Width, "token[%d] width mismatch: token=%#v input=%q", i, tok, in)

		// Tokenize() produces contiguous coverage
		require.Equalf(t, prevEnd, tok.Raw.Start, "token[%d] non-contiguous: prevEnd=%d token=%#v input=%q", i, prevEnd, tok, in)
		prevEnd = tok.Raw.End

		// Text token convention: Raw == Inner
		if tok.Type == TokenText {
			require.Equalf(t, tok.Raw, tok.Inner, "token[%d] text token raw!=inner: %#v input=%q", i, tok, in)
		}
	}

	require.Equalf(t, n, prevEnd, "tokens do not cover input: covered=%d n=%d input=%q toks=%#v", prevEnd, n, in, toks)
}

func sliceByRaw(in string, toks []Token) string {
	var b strings.Builder
	for _, tok := range toks {
		b.WriteString(in[tok.Raw.Start:tok.Raw.End])
	}
	return b.String()
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
		if tok.Type != TokenTag || tok.TagID != '`' {
			continue
		}
		raw := in[tok.Raw.Start:tok.Raw.End]
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
