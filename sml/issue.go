package sml

import "fmt"

type Kind int

const (
	KindInternal Kind = iota
	KindConfig
	KindUnknownNodeType
	KindAttributeNotAllowed
	KindAttributeInvalidPayload
)

type Issue struct {
	Kind Kind
	Err  error
}

func (i *Issue) Error() string {
	return fmt.Sprintf("SML Issue; Kind=[] ")
}
