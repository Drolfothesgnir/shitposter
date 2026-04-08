package shit

type Body struct {
	Blocks  []Block `json:"blocks"`
	Version int     `json:"version"`
}
