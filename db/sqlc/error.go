package db

import "fmt"

var (
	ErrDuplicateVote    = fmt.Errorf("duplicate vote")
	ErrInvalidVoteValue = fmt.Errorf("vote value must be either 1 of -1")
)
