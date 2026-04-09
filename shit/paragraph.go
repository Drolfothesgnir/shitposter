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

func (p *Paragraph) Parse(eater sml.Eater, w *[]string) error {
	outpoop, err := eater.Munch(p.Content)
	if err != nil {
		return err
	}

	p.parsedContent = outpoop.HTML(w)
	return nil
}
