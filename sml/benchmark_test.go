package sml

import (
	"strings"
	"testing"

	"github.com/Drolfothesgnir/shitposter/scum"
)

var (
	benchSinkPoop      Poop
	benchSinkHTML      string
	benchSinkIssues    []SyntaxIssue
	benchSinkText      string
	benchSinkTextLen   int
	benchSinkIssuesLen int
)

func benchEater(b *testing.B) Eater {
	b.Helper()

	e, err := NewEater(scum.WarnOverflowTrunc, 256)
	if err != nil {
		b.Fatal(err)
	}

	return e
}

func benchLongInput(repeat int) string {
	chunk := `plain <&> $bold *italic _under [link]!href{https://example.com}!target{_blank}!title{safe title}_*$ tail `
	return strings.Repeat(chunk, repeat)
}

func benchDeeplyNested(level int) string {
	var b strings.Builder
	b.Grow((level * 6) + 128)

	for i := 0; i < level; i++ {
		b.WriteString("$*_")
	}

	b.WriteString("core [link]!href{https://example.com}!target{_self}")

	for i := 0; i < level; i++ {
		b.WriteString("_*$")
	}

	return b.String()
}

func benchChaosInput(repeat int) string {
	chunk := `$*_[broken link]!href{//evil.com}!target{popup}!title{` + strings.Repeat("x", MaxTitleLength+8) + `} plain\\*text`
	return strings.Repeat(chunk, repeat)
}

func BenchmarkMunch_Simple(b *testing.B) {
	e := benchEater(b)
	input := `plain <&> $bold *italic _under [link]!href{https://example.com}!target{_blank}!title{safe title}_*$ tail`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkPoop, benchSinkIssues = e.Munch(input)
	}
}

func BenchmarkMunch_LongInput(b *testing.B) {
	e := benchEater(b)
	input := benchLongInput(64)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkPoop, benchSinkIssues = e.Munch(input)
	}
}

func BenchmarkMunch_DeeplyNested(b *testing.B) {
	e := benchEater(b)
	input := benchDeeplyNested(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkPoop, benchSinkIssues = e.Munch(input)
	}
}

func BenchmarkMunch_ChaosIssues(b *testing.B) {
	e := benchEater(b)
	input := benchChaosInput(32)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkPoop, benchSinkIssues = e.Munch(input)
		benchSinkIssuesLen = len(benchSinkIssues)
	}
}

func BenchmarkHTML_Render_Simple(b *testing.B) {
	e := benchEater(b)
	input := `plain <&> $bold *italic _under [link]!href{https://example.com}!target{_blank}!title{safe title}_*$ tail`
	poop, _ := e.Munch(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkHTML = poop.HTML()
	}
}

func BenchmarkHTML_Render_LongInput(b *testing.B) {
	e := benchEater(b)
	input := benchLongInput(64)
	poop, _ := e.Munch(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkHTML = poop.HTML()
	}
}

func BenchmarkText_Render_LongInput(b *testing.B) {
	e := benchEater(b)
	input := benchLongInput(64)
	poop, _ := e.Munch(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkText = poop.Text()
	}
}

func BenchmarkTextByteLen_Read_LongInput(b *testing.B) {
	e := benchEater(b)
	input := benchLongInput(64)
	poop, _ := e.Munch(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkTextLen = poop.TextByteLen()
	}
}

func BenchmarkMunchAndHTML_EndToEnd(b *testing.B) {
	e := benchEater(b)
	input := benchLongInput(32)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, issues := e.Munch(input)
		benchSinkIssues = issues
		benchSinkHTML = p.HTML()
	}
}
