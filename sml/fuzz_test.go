package sml

import (
	"strings"
	"testing"

	"github.com/Drolfothesgnir/shitposter/scum"
)

func FuzzPoopHTML_DeterministicAndTextLengthInvariant(f *testing.F) {
	seeds := []string{
		"",
		"plain text",
		`<&>"'`,
		`$bold$`,
		`$bold *italic _under_$*$`,
		`[link]!href{https://example.com?q=1&x=<y>}!target{_blank}!title{tiny}`,
		`[link]!href{javascript:alert(1)}!target{_parent}`,
		`$unclosed *nested [link]!href{//evil.example}`,
		`slashes \ \$ \* \_ \[ \]`,
		string([]byte{'[', 'x', ']', '!', 'h', 'r', 'e', 'f', '{', 0xff, '}'}),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	eater, err := NewEater(scum.WarnOverflowTrunc, 32)
	if err != nil {
		f.Fatalf("NewEater: %v", err)
	}

	f.Fuzz(func(t *testing.T, input string) {
		poop := eater.Munch(input)

		text := poop.Text()
		if len(text) != poop.TextByteLen() {
			t.Fatalf("TextByteLen mismatch: got %d, want len(%q)=%d", poop.TextByteLen(), text, len(text))
		}

		html1, issues1 := poop.HTML()
		html2, issues2 := poop.HTML()
		if html1 != html2 {
			t.Fatalf("HTML is not deterministic:\nfirst:  %q\nsecond: %q", html1, html2)
		}
		assertSameIssues(t, issues1, issues2)

		for _, issue := range issues1 {
			assertIssueMethodsDoNotReturnEmptyCodename(t, issue)
		}
		for _, issue := range poop.Warnings {
			assertIssueMethodsDoNotReturnEmptyCodename(t, issue)
		}
	})
}

func FuzzAttrHref_Invariants(f *testing.F) {
	seeds := []string{
		"",
		"   ",
		"https://example.com/path?q=1&x=<y>",
		"HTTPS://EXAMPLE.COM",
		"mailto:hello@example.com?subject=<hi>",
		"/relative/path?x=<y>",
		"//evil.example/path",
		"javascript:alert(1)",
		"https://example.com/%zz",
		"https://example.com/\nwat",
		string([]byte{0xff, 0xfe}),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		var b strings.Builder
		issues := Issues{}

		ok := attrHref(&b, &issues, scum.SerializableAttribute{
			Name:    "href",
			Payload: payload,
		})
		out := b.String()

		if !ok {
			if out != "" {
				t.Fatalf("rejected href wrote output: %q", out)
			}
			if len(issues.List) == 0 {
				t.Fatalf("rejected href produced no issue for payload %q", payload)
			}
			return
		}

		if len(issues.List) != 0 {
			t.Fatalf("accepted href produced issues: %#v", issues.List)
		}
		if !strings.HasPrefix(out, `href="`) || !strings.HasSuffix(out, `"`) {
			t.Fatalf("accepted href has malformed attribute output: %q", out)
		}
		if strings.ContainsAny(out, "<>\x00\r\n\t") {
			t.Fatalf("accepted href output contains raw forbidden characters: %q", out)
		}
		if strings.HasPrefix(strings.TrimSpace(payload), "//") {
			t.Fatalf("protocol-relative href was accepted: payload=%q output=%q", payload, out)
		}
	})
}

func assertSameIssues(t *testing.T, a, b []SyntaxIssue) {
	t.Helper()

	if len(a) != len(b) {
		t.Fatalf("issue count mismatch: got %d and %d", len(a), len(b))
	}
	for idx := range a {
		if a[idx].Code() != b[idx].Code() ||
			a[idx].Codename() != b[idx].Codename() ||
			a[idx].Description() != b[idx].Description() {
			t.Fatalf("issue %d mismatch: %#v != %#v", idx, a[idx], b[idx])
		}
	}
}

func assertIssueMethodsDoNotReturnEmptyCodename(t *testing.T, issue SyntaxIssue) {
	t.Helper()

	if issue.Codename() == "" {
		t.Fatalf("issue has empty codename: %#v", issue)
	}
	_ = issue.String()
	_ = issue.Code()
	_ = issue.Description()
}
