package scum

import "testing"

// Benchmark functions must start with "Benchmark" and take *testing.B

func BenchmarkTokenize(b *testing.B) {
	d := benchDict(b)
	input := "Hello *world* this is $$bold text$$ and [link text] with :[image]"

	b.ResetTimer() // exclude setup from timing
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Tokenize(&d, input, warns)
	}
}

func BenchmarkParse(b *testing.B) {
	d := benchDict(b)
	input := "Hello *world* this is $$bold text$$ and [link text] with :[image]"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Parse(input, &d, warns)
	}
}

func BenchmarkSerialize(b *testing.B) {
	d := benchDict(b)
	warns := &Warnings{}
	input := "Hello *world* this is $$bold text$$ and [link text] with :[image]"
	tree := Parse(input, &d, warns)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Serialize()
	}
}

func BenchmarkParse_LongInput(b *testing.B) {
	d := benchDict(b)
	// ~1KB of mixed content
	chunk := "Hello *world* this is $$bold text$$ and [link text] end. "
	input := ""
	for i := 0; i < 20; i++ {
		input += chunk
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Parse(input, &d, warns)
	}
}

func BenchmarkParse_DeeplyNested(b *testing.B) {
	d := benchDict(b)
	input := "[$$*deeply nested content*$$]"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Parse(input, &d, warns)
	}
}

// Chaos benchmarks - stress tests with malformed/complex input

func BenchmarkParse_Chaos_UnclosedTagStorm(b *testing.B) {
	d := benchDict(b)
	// 50 unclosed tags in a row - parser must handle all warnings
	input := ""
	for i := 0; i < 50; i++ {
		input += "[*$$:["
	}
	input += "the abyss stares back"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Parse(input, &d, warns)
	}
}

func BenchmarkParse_Chaos_NestingMatryoshka(b *testing.B) {
	d := benchDict(b)
	// 20 levels of properly nested tags - tests stack depth
	input := ""
	for i := 0; i < 20; i++ {
		input += "[*$$"
	}
	input += "core"
	for i := 0; i < 20; i++ {
		input += "$$*]"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Parse(input, &d, warns)
	}
}

func BenchmarkParse_Chaos_AlternatingOpenClose(b *testing.B) {
	d := benchDict(b)
	// Rapid open-close-open-close pattern
	input := ""
	for i := 0; i < 100; i++ {
		input += "*x*$$y$$[z]"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Parse(input, &d, warns)
	}
}

func BenchmarkParse_Chaos_EscapeHell(b *testing.B) {
	d := benchDict(b)
	// Lots of escape sequences mixed with real tags
	input := ""
	for i := 0; i < 50; i++ {
		input += "\\*real*\\$$fake$$\\[trap]*escaped*"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Parse(input, &d, warns)
	}
}

func BenchmarkParse_Chaos_WarningFlood(b *testing.B) {
	d := benchDict(b)
	// Mismatched tags generating tons of warnings
	// ] without [ , closing wrong tags, duplicates
	input := ""
	for i := 0; i < 30; i++ {
		input += "]*text[*more]*again[$$oops]"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Parse(input, &d, warns)
	}
}

func BenchmarkParse_Chaos_MixedMayhem(b *testing.B) {
	d := benchDict(b)
	// Everything at once: nested, unclosed, escaped, attributes, greedy
	input := `[intro *bold $$nested :[image inside] more text
		\*escaped\* back to $$double bold$$ and *italic
		unclosed [ another [ and ` + "`code block with *ignored* tags`" + `
		]finally] closing *some* $$tags$$ but not [all`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warns := &Warnings{}
		Parse(input, &d, warns)
	}
}

// Helper to create dictionary for benchmarks
func benchDict(b *testing.B) Dictionary {
	b.Helper()
	d, err := NewDictionary(Limits{})
	if err != nil {
		b.Fatal(err)
	}

	d.AddUniversalTag("BOLD", []byte("$$"), NonGreedy, RuleNA)
	d.AddUniversalTag("ITALIC", []byte("*"), NonGreedy, RuleNA)
	d.AddUniversalTag("UNDERLINE", []byte("_"), NonGreedy, RuleInfraWord)
	d.AddTag("LINK_TEXT_START", []byte("["), NonGreedy, RuleNA, 0, ']')
	d.AddTag("IMAGE", []byte(":["), NonGreedy, RuleNA, 0, ']')
	d.AddTag("LINK_TEXT_END", []byte("]"), NonGreedy, RuleNA, '\r', 0)
	d.AddUniversalTag("CODE", []byte("`"), Greedy, RuleTagVsContent)
	d.SetEscapeTrigger('\\')

	return d
}
