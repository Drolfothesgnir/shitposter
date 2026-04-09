package shit

import (
	"encoding/json"
	"fmt"
	"io"
)

type Block interface {
	Type() string
	Render(io.Writer) error
}

// A wrapper to handle the unmarshaling router
type TypedBlock struct {
	Block Block
}

func (b TypedBlock) Type() string {
	return b.Block.Type()
}

func (b TypedBlock) Render(w io.Writer) error {
	return nil
}

func (tb *TypedBlock) UnmarshalJSON(data []byte) error {
	// 1. Peek at the type
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}

	// 2. Unmarshal into the correct concrete struct
	switch base.Type {
	case "paragraph":
		var p Paragraph
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		tb.Block = p
	case "code":
		var c Code
		if err := json.Unmarshal(data, &c); err != nil {
			return err
		}
		tb.Block = c

	// other types ...
	default:
		return fmt.Errorf("unknown block type: %s", base.Type)
	}

	return nil
}
