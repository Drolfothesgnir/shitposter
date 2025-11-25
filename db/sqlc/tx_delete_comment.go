package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type DeleteCommentTxParams struct {
	CommentID int64 `json:"comment_id"`
	UserID    int64 `json:"user_id"`
	PostID    int64 `json:"post_id"`
}

type DeleteCommentTxResult struct {
	DeleteCommentIfLeafRow
	Success bool // True if the delete operation is considered successful: hard delete, soft delete, or already deleted.
}

func (s *SQLStore) DeleteCommentTx(ctx context.Context, arg DeleteCommentTxParams) (DeleteCommentTxResult, error) {
	var row DeleteCommentIfLeafRow
	var result DeleteCommentTxResult
	err := s.execTx(ctx, func(q *Queries) error {
		deleted, err := q.DeleteCommentIfLeaf(ctx, DeleteCommentIfLeafParams{
			PCommentID: arg.CommentID,
			PUserID:    arg.UserID,
			PPostID:    arg.PostID,
		})

		// if error is something other than "not found" return it
		// otherwise return "not found"
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrEntityNotFound
			}
			return err
		}

		// check if the target comment does not belong to the provided user
		if deleted.UserID != arg.UserID {
			return ErrEntityDoesNotBelongToUser
		}

		// check if the target comment does not belong to the provided post
		if deleted.PostID != arg.PostID {
			return ErrInvalidPostID
		}

		// if deletion performed successfully or
		// the target comment is already deleted return
		if deleted.DeletedOk || deleted.IsDeleted {
			row = deleted
			return nil
		}

		// if the comment has children perform soft delete
		if deleted.HasChildren {

			comment, err := q.SoftDeleteComment(ctx, arg.CommentID)
			if err != nil {
				return err
			}

			row = DeleteCommentIfLeafRow{
				ID:          comment.ID,
				UserID:      comment.UserID,
				PostID:      comment.PostID,
				IsDeleted:   comment.IsDeleted,
				DeletedAt:   comment.DeletedAt,
				HasChildren: true,
				DeletedOk:   true,
			}

			return nil
		}

		// else the data must be corrupted
		return ErrDataCorrupted
	})

	if err != nil {
		return result, err
	}

	result = DeleteCommentTxResult{
		DeleteCommentIfLeafRow: row,
		Success:                row.DeletedOk || row.IsDeleted,
	}

	return result, nil
}
