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

type Parser struct {
	dict     scum.Dictionary
	ast      scum.AST
	tree     scum.SerializableNode
	Warnings scum.Warnings
	Issues   []string
}

func (p *Parser) Eat(input string) {
	ast := scum.Parse(input, &p.dict, &p.Warnings)
	tree := ast.Serialize(&p.dict)
	p.ast = ast
	p.tree = tree
}

func (p *Parser) HTML() string {
	var b strings.Builder
	for _, n := range p.tree.Children {
		handleNode(&b, &p.Issues, n)
	}
	return b.String()
}

func (p Parser) TextLength() int {
	return p.ast.TextLength
}

func NewParser(warnPol scum.WarningOverflowPolicy, warnCap int) (Parser, error) {
	d, _ := scum.NewDictionary(scum.Limits{})

	_ = d.AddUniversalTag(Bold, []byte{'$'}, scum.NonGreedy, scum.RuleNA)

	_ = d.AddUniversalTag(Italic, []byte{'*'}, scum.NonGreedy, scum.RuleNA)

	_ = d.AddUniversalTag(Underline, []byte{'_'}, scum.NonGreedy, scum.RuleInfraWord)

	_ = d.AddTag(Link, []byte{'['}, scum.NonGreedy, scum.RuleNA, 0, ']')

	_ = d.AddTag(Link, []byte{']'}, scum.NonGreedy, scum.RuleNA, '[', 0)

	_ = d.SetAttributeSignature('!', '{', '}')

	_ = d.SetEscapeTrigger('\\')

	w, err := scum.NewWarnings(warnPol, warnCap)
	if err != nil {
		return Parser{}, err
	}

	return Parser{
		dict:     d,
		Warnings: w,
		// TODO: it should not be like that, better warning handling needed
		Issues: make([]string, 0, warnCap),
	}, nil
}
