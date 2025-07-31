package db

type PostBody struct {
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}
