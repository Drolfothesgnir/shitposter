package db

import "encoding/json"

type PostBody struct {
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

func GetPostBodyFromJSON(body_bytes []byte) (*PostBody, error) {
	var body PostBody
	err := json.Unmarshal(body_bytes, &body)
	if err != nil {
		return nil, err
	}

	return &body, nil
}

func GetPostTopicsFromJSON(topics_bytes []byte) ([]string, error) {
	var topics []string
	err := json.Unmarshal(topics_bytes, &topics)
	if err != nil {
		return nil, err
	}

	return topics, nil
}
