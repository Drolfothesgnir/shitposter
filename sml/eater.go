// Shitposter Markup Language
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
