package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// DeleteCommentTx hard-deletes a comment if it is a leaf,
// soft-deletes it if it has children,
// and is a no-op if the comment does not exist.
func (s *SQLStore) DeleteCommentTx(ctx context.Context, commentID int64) error {
	return s.execTx(ctx, func(q *Queries) error {
		// 1. Try to delete as leaf
		_, err := q.DeleteCommentIfLeaf(ctx, commentID)
		switch {
		case err == nil:
			// Deleted as leaf
			return nil

		case !errors.Is(err, pgx.ErrNoRows):
			// Real error
			return err
		}

		// 2. Either not a leaf or doesn't exist: try soft delete
		_, err = q.SoftDeleteComment(ctx, commentID)
		if errors.Is(err, pgx.ErrNoRows) {
			// if the error is "not found" comment is not in the db
			// which is considered a success
			return nil
		}

		return err
	})
}
