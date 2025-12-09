package content

import (
	"encoding/json"
	"errors"
	"fmt"
)

// PseudoAST implements Schema interface and used to parse and sanitize simple,
// block/section based post body content schema.
//
// Pseudo-AST stands for Pseudo Abstract Syntax Tree. It's a simplified post body schema, designed to be client/editor-agnostic.
type PseudoAST struct {
	version  int32
	sections []Section
}

func (s *PseudoAST) Name() string {
	return "pseudo-ast"
}

func (s *PseudoAST) Version() int32 {
	return s.version
}

// these unexported DTOs must have exported fields and json name tags
// to ensure encoding/json will parse raw data into these structs
type rawSchema struct {
	Version  int32        `json:"version"`
	Sections []rawSection `json:"sections"`
}

type rawSection struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Kind    Kind      `json:"kind"`
	Content []RawItem `json:"content"`
}

// Parse transforms raw json into PseudoAST with raw content items,
// parses them as specific content items, saves the tree internally
// and returns new marshaled tree as json.
func (s *PseudoAST) Parse(body []byte) ([]byte, error) {

	// 1) parse raw json
	var rawParsed rawSchema
	if err := json.Unmarshal(body, &rawParsed); err != nil {
		return nil, err // deal with error later
	}

	if rawParsed.Version != s.version {
		return nil, fmt.Errorf("unsupported body version: got %d, want %d", rawParsed.Version, s.version)
	}
	if len(rawParsed.Sections) == 0 {
		return nil, errors.New("body.sections must not be empty")
	}

	// 2) parse and sanitize contents and store into []Section
	parsedContent := make([]Section, len(rawParsed.Sections))

	for i, sec := range rawParsed.Sections {
		if sec.ID == "" {
			return nil, fmt.Errorf("section[%d]: id is required", i)
		}
		if len(sec.Content) == 0 {
			return nil, fmt.Errorf("section[%d]: content must not be empty", i)
		}
		if sec.Kind == "" {
			sec.Kind = KindDefault
		}

		section := Section{
			ID:      sec.ID,
			Title:   sec.Title,
			Kind:    sec.Kind,
			Content: make([]ContentItem, len(sec.Content)),
		}

		for j, rawItem := range sec.Content {
			var (
				item ContentItem
				err  error
			)

			switch rawItem.Type {
			case TypeParagraph:
				item, err = NewParagraph(rawItem.Raw)

			default:
				err = fmt.Errorf("section[%d].content[%d]: unknown type %q", i, j, rawItem.Type)
			}

			if err != nil {
				return nil, err
			}

			section.Content[j] = item
		}

		parsedContent[i] = section
	}

	s.sections = parsedContent

	canonical := struct {
		Version  int32     `json:"version"`
		Sections []Section `json:"sections"`
	}{
		Version:  s.version,
		Sections: parsedContent,
	}

	out, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}

	return out, nil

}

// Section defines a separate block of the content.
// TODO: refine the comments
type Section struct {
	ID      string        `json:"id"`      // Required. Must be unique across all sections.
	Title   string        `json:"title"`   // Optional. Defines the display name of each section.
	Kind    Kind          `json:"kind"`    // Required. Defines the type of the block. "default" by default.
	Content []ContentItem `json:"content"` // Required. Actual body of the block.
}

// RawItem defines not-fully parsed json content item to be later parsed as ContentItem based on the Type field.
type RawItem struct {
	Type Type            `json:"type"`
	Raw  json.RawMessage // the whole JSON object for this item
}

// UnmarshalJSON helps saving all the data in the Raw field
// and still be able to access the content type via Type field
func (ri *RawItem) UnmarshalJSON(data []byte) error {
	// 1) first copy all the data into the Raw field
	ri.Raw = make(json.RawMessage, len(data))
	copy(ri.Raw, data)

	// 2) extract only type and save it in the Type field
	var aux struct {
		Type Type `json:"type"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	ri.Type = aux.Type
	return nil
}

type Image struct {
	Typed
	URL     string `json:"url"`     // Required. URL to the image.
	Alt     string `json:"alt"`     // Optional. Alternative image caption, provided for accessibility.
	Caption string `json:"caption"` // Optional. Image caption markdown.
}

type List struct {
	Typed
	Style string   `json:"style"` // Required. Can be one of "bullet" or "numbered".
	Items []string `json:"items"` // Required, not empty. Each element can be a markdown.
}

type Code struct {
	Typed
	Language string `json:"language"` // Optional. Can be "go", "js", "sql", etc.
	Code     string `json:"code"`     // Required.
}

type Quote struct {
	Typed
	Markdown string `json:"markdown"` // Required. Quote's body.
	Author   string `json:"author"`   // Optional.
}

// Content divider.
type Divider struct {
	Typed
}
