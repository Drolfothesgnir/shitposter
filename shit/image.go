package shit

type Image struct {
	ID      string `json:"id"`
	Src     string `json:"src"`
	Alt     string `json:"alt"`
	Caption string `json:"caption"`
	AssetID string `json:"asset_id"`
}

func (i Image) Type() string {
	return "image"
}
