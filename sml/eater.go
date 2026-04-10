// # Shitposter's Markup Language
//
// It is created for the Shiposter users to be able to create tag-based rich text in their posts.
// The parsed rich text can be transformed to an HTML or plain text string.
//
// # Tags
//
// Tags can be embedded.
//
// $ - Bold:
//
//   - Simple bold text.
//   - Accepts no attributes.
//   - Written as $text$ and rendered in HTML as <strong>text</strong>.
//
// * - Italic:
//
//   - Simple italic/emphasized text.
//   - Accepts no attributes.
//   - Written as *text* and rendered in HTML as <em>text</em>.
//
// _ - Underline:
//
//   - Represents text with a line below.
//   - Accepts no attributes.
//   - Written as _text_ and rendered in HTML as <span class="sml-underline">text</span>.
//
// [...] - Link:
//
//   - Represents hyperlink.
//
//   - Accepts attributes:
//
//     href - Required, for link to be valid.
//     Must not contain forbidden control characters.
//     Must not be protocol-relative.
//     Schema must be one of "http", "https" or "mailto".
//
//     target - Optional.
//     Must be one of "_blank" or "_self".
//     In case of "_blank" value `rel="noopener noreferrer"` will be added as an attribute.
//
//     title - Optional.
//     Must not contain forbidden control characters.
//     Must be at most [MaxTitleLength] characters long.
//
//   - Written as [text]!href{https://address.com}!target{_blank}!title{this is a link} and
//     rendered in HTML as <a href="https://address.com" target="_blank" rel="noopener noreferrer" title="this is a link">text</a>.
//     When rendered as plain text, only the text in the [] will be rendered.
package sml

import (
	"strings"

	"github.com/Drolfothesgnir/shitposter/scum"
)

const (
	Bold      = "BOLD"
	Italic    = "ITALIC"
	Underline = "UNDERLINE"
	Link      = "LINK"
)

type Poop struct {
	ast      scum.AST
	tree     scum.SerializableNode
	Warnings scum.Warnings
}

func (p *Poop) HTML(w *[]string) string {
	var b strings.Builder
	for _, n := range p.tree.Children {
		handleNode(&b, w, n)
	}
	return b.String()
}

func (p Poop) TextLength() int {
	return p.ast.TextLength
}

type Eater struct {
	dict                  scum.Dictionary
	WarningOverflowPolicy scum.WarningOverflowPolicy
	WarnCap               int
}

func (p Eater) Munch(input string) (Poop, error) {
	w, err := scum.NewWarnings(p.WarningOverflowPolicy, p.WarnCap)
	if err != nil {
		return Poop{}, err
	}
	ast := scum.Parse(input, &p.dict, &w)
	tree := ast.Serialize(&p.dict)

	return Poop{
		ast:      ast,
		tree:     tree,
		Warnings: w,
	}, nil
}

func NewEater(warnPol scum.WarningOverflowPolicy, warnCap int) Eater {
	d, _ := scum.NewDictionary(scum.Limits{})

	_ = d.AddUniversalTag(Bold, []byte{'$'}, scum.NonGreedy, scum.RuleNA)

	_ = d.AddUniversalTag(Italic, []byte{'*'}, scum.NonGreedy, scum.RuleNA)

	_ = d.AddUniversalTag(Underline, []byte{'_'}, scum.NonGreedy, scum.RuleInfraWord)

	_ = d.AddTag(Link, []byte{'['}, scum.NonGreedy, scum.RuleNA, 0, ']')

	_ = d.AddTag(Link, []byte{']'}, scum.NonGreedy, scum.RuleNA, '[', 0)

	_ = d.SetAttributeSignature('!', '{', '}')

	_ = d.SetEscapeTrigger('\\')

	return Eater{
		dict:                  d,
		WarningOverflowPolicy: warnPol,
		WarnCap:               warnCap,
	}
}
