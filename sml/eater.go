// # Shitposter's Markup Language
//
// It is created for the Shitposter users to be able to create tag-based rich text in their posts.
// The parsed rich text can be transformed to an HTML or plain text string.
// Attribute names are case-insensetive.
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
//   - Written as _text_ and rendered in HTML as <span class="sml-internal-underline">text</span>.
//
// [...] - Link:
//
//   - Represents hyperlink. In case of multiple attributes with the same name (case-insensetive), the first one will be used
//     and others discarded.
//
//   - Accepts attributes:
//
//     href - Optional.
//     SML will not enforce the user to add the url attribute. User must provide it
//     by themselves otherwise the link will be non-functional.
//     Must not contain forbidden control characters.
//     Must not be protocol-relative.
//     Scheme must be one of "http", "https" or "mailto".
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
	"slices"

	"github.com/Drolfothesgnir/shitposter/scum"
)

// Tag names
const (
	Bold      = "BOLD"
	Italic    = "ITALIC"
	Underline = "UNDERLINE"
	Link      = "LINK"
)

// Poop is the result of the input parsing, returned by [Eater.Munch].
// It contains the parsed tree and methods for rendering it as HTML or plain text.
// Syntax issues are returned separately by [Eater.Munch].
type Poop struct {
	Input string
	AST   scum.AST
	Tree  scum.SerializableNode
}

// Text returns the parsed input as plain text string.
func (p Poop) Text() string {
	return p.AST.Text()
}

// TextByteLength returns the byte count of the plain text in the input, that is the non-tag and non-attribute parts.
func (p Poop) TextByteLen() int {
	return p.AST.TextByteLen
}

// Eater is the main SML parser object.
type Eater struct {
	dict scum.Dictionary
	// warningOverflowPolicy determines what happends when the maximum Warning capacity is reached.
	warningOverflowPolicy scum.WarningOverflowPolicy
	// warnCap is the maximum number of warnings which will be processed during parsing.
	warnCap int
}

// Munch parses and normalizes the input, returning a [Poop] and all syntax issues
// found while parsing, validating and normalizing.
func (p *Eater) Munch(input string) (Poop, []SyntaxIssue) {
	w, _ := scum.NewWarnings(p.warningOverflowPolicy, p.warnCap)
	ast := scum.Parse(input, &p.dict, &w)
	tree := ast.Serialize(&p.dict)
	issues := NewIssues(len(input) / 10)
	normalizeRenderTree(&tree, &issues)
	scumWarns := make([]scum.SerializableWarning, 0, w.WarnCount())
	w.SerializeAll(&scumWarns, &p.dict)
	warns := make([]SyntaxIssue, 0, w.WarnCount())
	for _, w := range scumWarns {
		warns = append(warns, Warning{w})
	}
	allIssues := slices.Concat(warns, issues.List)
	return Poop{
		Input: input,
		AST:   ast,
		Tree:  tree,
	}, allIssues
}

// It will return a *[ConfigError] if invalid arguments passed.
func NewEater(warnPol scum.WarningOverflowPolicy, warnCap int) (Eater, error) {
	d, err := scum.NewDictionary(scum.Limits{})
	if err != nil {
		// this should not happen
		panic(err.Error())
	}

	// checking if provided arguments are valid.
	// this helps to avoid error check in the Munch method
	_, err = scum.NewWarnings(warnPol, warnCap)
	if err != nil {
		return Eater{}, NewConfigError("SML Parser", ReasonInvalidParams, err)
	}

	err = d.AddUniversalTag(Bold, []byte{'$'}, scum.NonGreedy, scum.RuleNA)
	if err != nil {
		// this should not happen
		panic(err.Error())
	}
	err = d.AddUniversalTag(Italic, []byte{'*'}, scum.NonGreedy, scum.RuleNA)
	if err != nil {
		// this should not happen
		panic(err.Error())
	}
	err = d.AddUniversalTag(Underline, []byte{'_'}, scum.NonGreedy, scum.RuleInfraWord)
	if err != nil {
		// this should not happen
		panic(err.Error())
	}
	err = d.AddTag(Link, []byte{'['}, scum.NonGreedy, scum.RuleNA, 0, ']')
	if err != nil {
		// this should not happen
		panic(err.Error())
	}
	err = d.AddTag(Link, []byte{']'}, scum.NonGreedy, scum.RuleNA, '[', 0)
	if err != nil {
		// this should not happen
		panic(err.Error())
	}
	err = d.SetAttributeSignature('!', '{', '}')
	if err != nil {
		// this should not happen
		panic(err.Error())
	}
	err = d.SetEscapeTrigger('\\')
	if err != nil {
		// this should not happen
		panic(err.Error())
	}

	return Eater{
		dict:                  d,
		warningOverflowPolicy: warnPol,
		warnCap:               warnCap,
	}, nil
}
