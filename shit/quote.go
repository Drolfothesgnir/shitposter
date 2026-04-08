package shit

type Quote struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Author  string `json:"author"`
}

func (q Quote) Type() string {
	return "quote"
}
