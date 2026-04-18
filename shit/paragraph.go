package shit

import (
	"io"

	"github.com/Drolfothesgnir/shitposter/sml"
)

type Paragraph struct {
	ID            string `json:"id"`
	Content       string `json:"content"`
	parsedContent string
}

func (p Paragraph) Type() string {
	return "paragraph"
}

func (p Paragraph) Render(w io.Writer) error {
	return nil
}

func (p *Paragraph) Parse(eater sml.Eater, i *sml.Issues) error {
	outpoop, issues := eater.Munch(p.Content)

	html := outpoop.HTML()
	p.parsedContent = html
	if i != nil {
		for _, issue := range issues {
			i.Add(issue)
		}
	}
	return nil
}
