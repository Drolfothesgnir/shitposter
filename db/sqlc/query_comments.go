package db

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const opQueryComments = "query-comments"

const (
	CommentOrderPopular = "pop"
	CommentOrderNewest  = "new"
	CommentOrderOldest  = "old"
)

var CommentOrderMethods = []string{
	CommentOrderPopular,
	CommentOrderNewest,
	CommentOrderOldest,
}

type CommentQuery struct {
	Order  string // pop | old | new
	PostID int64
	Limit  int32
	Offset int32
}

func (s *SQLStore) QueryComments(ctx context.Context, q CommentQuery) ([]CommentsWithAuthor, error) {
	var result []CommentsWithAuthor
	var err error

	if !slices.Contains(CommentOrderMethods, q.Order) {
		// creating pretty readable list from order methods array
		methodsList := make([]string, len(CommentOrderMethods))
		for i, m := range CommentOrderMethods {
			methodsList[i] = strconv.Quote(m)
		}

		listStr := strings.Join(methodsList, ", ")

		opErr := newOpError(
			opQueryComments,
			KindInvalid,
			entComment,
			fmt.Errorf("invalid order \"%s\". can be one of %s", q.Order, listStr),
			withField("order"),
		)
		return result, opErr
	}

	switch q.Order {
	case CommentOrderPopular:
		result, err = s.GetCommentsByPopularity(ctx, GetCommentsByPopularityParams{q.PostID, q.Limit, q.Offset})
	case CommentOrderNewest:
		result, err = s.GetNewestComments(ctx, GetNewestCommentsParams{q.PostID, q.Limit, q.Offset})
	case CommentOrderOldest:
		result, err = s.GetOldestComments(ctx, GetOldestCommentsParams{q.PostID, q.Limit, q.Offset})
	}

	// even if the post doesn't have comments
	// or doesn't exist in the first place
	// return empty slice
	if errors.Is(err, pgx.ErrNoRows) {
		return []CommentsWithAuthor{}, nil
	}

	if err != nil {
		opErr := sqlError(
			opQueryComments,
			opDetails{postID: q.PostID, entity: entComment},
			err,
		)
		return result, opErr
	}

	return result, nil
}
