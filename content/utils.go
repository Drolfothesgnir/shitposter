package content

// Kind defines the type of the section, e.g. "default", "gallery", "faq"
type Kind string

const (
	KindDefault Kind = "default"
)

type Type string

const (
	TypeParagraph Type = "paragraph"
	TypeImage     Type = "image"
	TypeList      Type = "list"
	TypeCode      Type = "code"
	TypeQuote     Type = "quote"
	TypeDivider   Type = "divider"
)

type ContentItem interface {
	ContentType() Type
}

type Typed struct {
	Type Type `json:"type"` // Required.
}

func (t Typed) ContentType() Type { return t.Type }
