package shit

import "io"

type Paragraph struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func (p Paragraph) Type() string {
	return "paragraph"
}

func (p Paragraph) Render(w io.Writer) error {
	return nil
}
