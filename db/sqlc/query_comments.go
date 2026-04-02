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

// CommentOrder defines possible sorting criterion for the [QueryComments] method.
// Can be [CommentOrderPopular], [CommentOrderNewest] or [CommentOrderOldest].
type CommentOrder string

const (
	CommentOrderPopular CommentOrder = "pop"
	CommentOrderNewest  CommentOrder = "new"
	CommentOrderOldest  CommentOrder = "old"
)

var CommentOrderMethods = []CommentOrder{
	CommentOrderPopular,
	CommentOrderNewest,
	CommentOrderOldest,
}

type CommentQuery struct {
	Order  CommentOrder // pop | old | new
	PostID int64
	Limit  int32
	Offset int32
}

// ValidateCommentOrderMethod returns *[OpError] if the provided order method is not allowed.
func ValidateCommentOrderMethod(orderMethod CommentOrder) error {
	if !slices.Contains(CommentOrderMethods, orderMethod) {
		// creating pretty readable list from order methods array
		methodsList := make([]string, len(CommentOrderMethods))
		for i, m := range CommentOrderMethods {
			methodsList[i] = strconv.Quote(string(m))
		}

		listStr := strings.Join(methodsList, ", ")

		opErr := newOpError(
			opQueryComments,
			KindInvalid,
			entComment,
			fmt.Errorf("invalid order \"%s\". can be one of %s", orderMethod, listStr),
			withField("order"),
		)
		return opErr
	}

	return nil
}

// QueryComments returns a paginated set of comments for a post, ordered by
// popularity ("pop"), newest first ("new"), or oldest first ("old").
// Returns an empty slice when the post has no comments or does not exist.
// Returns KindInvalid if the order value is invalid, or KindInternal on database errors.
func (s *SQLStore) QueryComments(ctx context.Context, q CommentQuery) ([]CommentsWithAuthor, error) {
	var result []CommentsWithAuthor

	err := ValidateCommentOrderMethod(q.Order)
	if err != nil {
		return result, err
	}

	switch q.Order {
	case CommentOrderPopular:
		result, err = s.getCommentsByPopularity(ctx, getCommentsByPopularityParams{q.PostID, q.Limit, q.Offset})
	case CommentOrderNewest:
		result, err = s.getNewestComments(ctx, getNewestCommentsParams{q.PostID, q.Limit, q.Offset})
	case CommentOrderOldest:
		result, err = s.getOldestComments(ctx, getOldestCommentsParams{q.PostID, q.Limit, q.Offset})
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
			opDetails{postID: fmt.Sprint(q.PostID), entity: entComment},
			err,
		)
		return result, opErr
	}

	return result, nil
}
