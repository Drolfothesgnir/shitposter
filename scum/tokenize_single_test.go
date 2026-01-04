package scum

import (
	"strings"
	"testing"
)

// ---------- dictionary fixture ----------

func mustAddTag(t *testing.T, d *Dictionary, name string, seq string, greed Greed, rule Rule, openID, closeID byte) {
	t.Helper()
	if err := d.AddTag(name, []byte(seq), greed, rule, openID, closeID); err != nil {
		t.Fatalf("AddTag(%q,%q) failed: %v", name, seq, err)
	}
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

func tokenize(t *testing.T, d *Dictionary, in string) ([]Token, []Warning) {
	t.Helper()
	var warns []Warning
	toks := Tokenize(d, in, &warns)
	return toks, warns
}

// ---------- warning helpers ----------

func hasIssue(warns []Warning, issue Issue) bool {
	for _, w := range warns {
		if w.Issue == issue {
			return true
		}
	}
	return false
}

func assertWarningInvariants(t *testing.T, in string, warns []Warning) {
	t.Helper()
	n := len(in)

	// Pos is a byte index "at which the problem occurred".
	// For EOL-like issues, it's reasonable to allow Pos == len(input).
	for i, w := range warns {
		if w.Pos < 0 || w.Pos > n {
			t.Fatalf("warning[%d] pos out of bounds: pos=%d n=%d warning=%#v input=%q", i, w.Pos, n, w, in)
		}
		// Optional: description should not be empty (depends on your style)
		// if strings.TrimSpace(w.Description) == "" {
		// 	t.Fatalf("warning[%d] empty description: %#v", i, w)
		// }
	}
}

// ---------- token invariants ----------

func assertTokenInvariants(t *testing.T, in string, toks []Token) {
	t.Helper()
	n := len(in)
	prevEnd := 0

	for i, tok := range toks {
		// Raw bounds
		if tok.Raw.Start < 0 || tok.Raw.End < 0 || tok.Raw.Start > tok.Raw.End || tok.Raw.End > n {
			t.Fatalf("token[%d] raw out of bounds: %#v n=%d input=%q", i, tok, n, in)
		}
		// Inner bounds
		if tok.Inner.Start < 0 || tok.Inner.End < 0 || tok.Inner.Start > tok.Inner.End || tok.Inner.End > n {
			t.Fatalf("token[%d] inner out of bounds: %#v n=%d input=%q", i, tok, n, in)
		}
		// Consistency
		if tok.Pos != tok.Raw.Start {
			t.Fatalf("token[%d] pos mismatch: pos=%d raw.start=%d token=%#v input=%q", i, tok.Pos, tok.Raw.Start, tok, in)
		}
		if tok.Width != (tok.Raw.End - tok.Raw.Start) {
			t.Fatalf("token[%d] width mismatch: width=%d rawWidth=%d token=%#v input=%q",
				i, tok.Width, tok.Raw.End-tok.Raw.Start, tok, in)
		}

		// Tokenize() produces contiguous coverage (it flushes text between tags and appends tag tokens).
		if tok.Raw.Start != prevEnd {
			t.Fatalf("token[%d] non-contiguous: prevEnd=%d start=%d token=%#v input=%q",
				i, prevEnd, tok.Raw.Start, tok, in)
		}
		prevEnd = tok.Raw.End

		// Text token convention in your Tokenize(): Raw == Inner
		if tok.Type == TokenText && tok.Raw != tok.Inner {
			t.Fatalf("token[%d] text token raw!=inner: %#v input=%q", i, tok, in)
		}
	}

	if prevEnd != n {
		t.Fatalf("tokens do not cover input: covered=%d n=%d input=%q toks=%#v", prevEnd, n, in, toks)
	}
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
	assertWarningInvariants(t, in, warns)

	// Lossless round-trip
	if out := sliceByRaw(in, toks); out != in {
		t.Fatalf("round-trip mismatch: in=%q out=%q", in, out)
	}

	// Typically no warns here.
	// If you decide to warn on "ignored tag", adjust accordingly.
	if len(warns) != 0 {
		t.Fatalf("expected no warnings, got: %#v", warns)
	}
}

func TestTokenizeSingle_Greedy_TagVsContent_TripleBackticksCapture(t *testing.T) {
	d := initSingleDict(t)
	in := "here is ```const rawStr = `hello`;``` ok"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, warns)

	// We expect a Tag token that spans triple backticks
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
	if !found {
		t.Fatalf("expected a CODE Tag token spanning triple backticks; toks=%#v", toks)
	}

	if len(warns) != 0 {
		t.Fatalf("expected no warnings, got: %#v", warns)
	}
}

func TestTokenizeSingle_Greedy_UnclosedBecomesTextAndWarns(t *testing.T) {
	d := initSingleDict(t)
	in := "start ```abc"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, warns)

	// Unclosed greedy should degrade to plain text (per your docs and singleCharGreedyPlan behavior)
	if out := sliceByRaw(in, toks); out != in {
		t.Fatalf("round-trip mismatch: in=%q out=%q", in, out)
	}

	// This one should warn
	if !hasIssue(warns, IssueUnclosedTag) {
		t.Fatalf("expected IssueUnclosedTag, got warnings: %#v", warns)
	}
}

func TestTokenizeSingle_OpeningBeforeEOL_WarnsUnexpectedEOL(t *testing.T) {
	d := initSingleDict(t)
	in := "["

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, warns)

	// Behavior: should not panic; should still round-trip
	if out := sliceByRaw(in, toks); out != in {
		t.Fatalf("round-trip mismatch: in=%q out=%q", in, out)
	}

	// Based on your Issue docs and step name step_open_tag_before_eol.go,
	// the most sensible warning is UnexpectedEOL
	if !hasIssue(warns, IssueUnexpectedEOL) {
		t.Fatalf("expected IssueUnexpectedEOL, got warnings: %#v", warns)
	}
}

func TestTokenizeSingle_ClosingAtStart_WarnsMisplacedClosingTag(t *testing.T) {
	d := initSingleDict(t)
	in := "]hello"

	toks, warns := tokenize(t, d, in)
	assertTokenInvariants(t, in, toks)
	assertWarningInvariants(t, in, warns)

	if out := sliceByRaw(in, toks); out != in {
		t.Fatalf("round-trip mismatch: in=%q out=%q", in, out)
	}

	if !hasIssue(warns, IssueMisplacedClosingTag) {
		t.Fatalf("expected IssueMisplacedClosingTag, got warnings: %#v", warns)
	}
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

		// Must not panic; tokens must be well-formed and cover the entire input.
		assertTokenInvariants(t, in, toks)

		// Warnings (if any) must have reasonable positions.
		assertWarningInvariants(t, in, warns)

		// Tokenize() is designed to be lossless at the tokenization stage.
		// If you later add normalization, relax this.
		if out := sliceByRaw(in, toks); out != in {
			t.Fatalf("round-trip mismatch: in=%q out=%q toks=%#v warns=%#v", in, out, toks, warns)
		}
	})
}
