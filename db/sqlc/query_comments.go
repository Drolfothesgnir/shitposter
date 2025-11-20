package db

import (
	"context"
	"fmt"
)

const (
	CommentOrderPopular = "pop"
	CommentOrderNewest  = "new"
	CommentOrderOldest  = "old"
)

type CommentQuery struct {
	Order  string // pop | old | new
	PostID int64
	Limit  int32
	Offset int32
}

func (s *SQLStore) QueryComments(ctx context.Context, q CommentQuery) ([]CommentsWithAuthor, error) {
	switch q.Order {
	case CommentOrderPopular:
		return s.GetCommentsByPopularity(ctx, GetCommentsByPopularityParams{q.PostID, q.Limit, q.Offset})
	case CommentOrderNewest:
		return s.GetNewestComments(ctx, GetNewestCommentsParams{q.PostID, q.Limit, q.Offset})
	case CommentOrderOldest:
		return s.GetOldestComments(ctx, GetOldestCommentsParams{q.PostID, q.Limit, q.Offset})
	default:
		return nil, fmt.Errorf("invalid order: %s", q.Order)
	}
}
