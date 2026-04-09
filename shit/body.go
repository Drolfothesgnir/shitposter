package shit

type Body struct {
	Blocks  []TypedBlock `json:"blocks"`
	Version int          `json:"version"`
}
