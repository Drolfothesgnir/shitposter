package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Paragraph struct {
	Typed
	Markdown string `json:"markdown"` // Required. Inline rich text, e.g. ***bold+italic***
}

// NewParagraph parses raw json paragraph data, sanitizes it and returns new Paragraph.
func NewParagraph(raw json.RawMessage) (*Paragraph, error) {
	var p Paragraph
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}

	if p.Type != TypeParagraph {
		return nil, fmt.Errorf("paragraph: expected type %q, got %q", TypeParagraph, p.Type)
	}

	if strings.TrimSpace(p.Markdown) == "" {
		return nil, errors.New("paragraph: markdown is required")
	}

	// TODO: sanitize markdown (strip dangerous HTML tags)
	return &p, nil
}
