package api

import (
	"fmt"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

type CommentNode struct {
	db.Comment
	Replies []*CommentNode `json:"replies,omitempty"`
}

func (comment *CommentNode) GetParentID() (int64, bool) {
	return comment.ParentID.Int64, comment.ParentID.Valid
}

// Utility for returning tree-like comments from plain ordered database query response.
// Comments from the database must be in depth-first order so the tree can be built from them.
//
// The tree will have n_roots number of roots
func PrepareCommentTree(orderedPlainComments []db.Comment, n_roots int) ([]*CommentNode, error) {
	result := make([]*CommentNode, 0, n_roots)
	stack := make([]*CommentNode, 0, 5) // 5 is the guess of typical comment thread depth

	for i := range orderedPlainComments {
		// taking &CommentNode instead of &comment is crucial to avoit address-of-loop-variable bug
		comment := &CommentNode{
			Comment: orderedPlainComments[i],
		}

		d := int(comment.Depth)
		// if comment is a root node append it to the result
		// and reset stack to contain new branch
		if d == 0 {
			result = append(result, comment)
			stack = append(stack[:0], comment)
			continue
		}
		// parent not yet seen (depth jump > 1)
		if d > len(stack) {
			return nil, fmt.Errorf("bad depth jump at %d (id=%d): %d > %d", i, comment.ID, d, len(stack))
		}

		// if comment is a reply then its parent must be previous node in the stack
		parent := stack[d-1]

		// check if comment has consistent depth
		if parent == nil {
			return nil, fmt.Errorf("corrupted data near index %d at depth %d", i, d)
		}

		// append child node to its parent replies
		parent.Replies = append(parent.Replies, comment)

		// if comment is a leaf append it to the end of the stack
		if d == len(stack) {
			stack = append(stack, comment)
		} else {
			// else if it's part of the other branch drop previous branch by cutting depth
			// and replacing its nodes with current comment
			stack = append(stack[:d], comment)
		}
	}

	return result, nil
}
