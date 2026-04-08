package shit

import "io"

type Code struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Language string `json:"language"`
}

func (c Code) Type() string {
	return "code"
}

func (c Code) Render(w io.Writer) error {
	return nil
}
