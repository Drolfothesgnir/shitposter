package db

import "errors"

var (
	ErrDuplicateVote               = errors.New("duplicate vote")
	ErrInvalidVoteValue            = errors.New("vote value must be either 1 of -1")
	ErrParentCommentNotFound       = errors.New("parent comment not found")
	ErrParentCommentPostIDMismatch = errors.New("parent comment has different post id")
	ErrInvalidPostID               = errors.New("invalid post id")
	ErrParentCommentDeleted        = errors.New("parent comment is deleted")
)
