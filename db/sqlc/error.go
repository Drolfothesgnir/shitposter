package db

import "fmt"

var (
	ErrDuplicateVote = fmt.Errorf("duplicate vote")
)
