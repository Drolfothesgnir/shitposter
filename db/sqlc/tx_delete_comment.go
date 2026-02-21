package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const opDeleteComment = "delete-comment"

type DeleteCommentTxParams struct {
	CommentID int64 `json:"comment_id"`
	UserID    int64 `json:"user_id"`
	PostID    int64 `json:"post_id"`
}

type DeleteCommentTxResult struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	PostID      int64     `json:"post_id"`
	IsDeleted   bool      `json:"is_deleted"`
	DeletedAt   time.Time `json:"deleted_at"`
	HasChildren bool      `json:"has_children"`
	DeletedOk   bool      `json:"deleted_ok"`
	Success     bool      `json:"success"` // True if the delete operation is considered successful: hard delete, soft delete, or already deleted.
}

// DeleteCommentTx deletes a comment. Leaf comments are hard-deleted; comments with
// children are soft-deleted (body cleared, is_deleted flag set). Already-deleted
// comments are treated as a successful no-op.
// Returns KindNotFound if the comment does not exist, KindPermission if the comment
// belongs to another user, KindRelation if the comment belongs to a different post,
// KindCorrupted on unexpected inconsistent DB state, or KindInternal on database errors.
func (s *SQLStore) DeleteCommentTx(ctx context.Context, arg DeleteCommentTxParams) (DeleteCommentTxResult, error) {
	var row deleteCommentIfLeafRow
	var result DeleteCommentTxResult
	err := s.execTx(ctx, func(q *Queries) error {
		deleted, err := q.deleteCommentIfLeaf(ctx, deleteCommentIfLeafParams{
			PCommentID: arg.CommentID,
			PUserID:    arg.UserID,
			PPostID:    arg.PostID,
		})

		// if error is something other than "not found" return it
		// otherwise return "not found"
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return notFoundError(opDeleteComment, entComment, arg.CommentID)
			}
			return sqlError(
				opDeleteComment,
				opDetails{
					userID:    arg.UserID,
					postID:    arg.PostID,
					commentID: arg.CommentID,
					entity:    entComment,
				},
				err,
			)
		}

		// check if the target comment does not belong to the provided user
		if deleted.UserID != arg.UserID {
			return newOpError(
				opDeleteComment,
				KindPermission,
				entComment,
				fmt.Errorf("comment with id %d does not belong to user with id %d", arg.CommentID, arg.UserID),
				withRelated(entUser, arg.UserID),
				withUser(arg.UserID),
				withEntityID(arg.CommentID),
				withField("user_id"),
			)
		}

		// check if the target comment does not belong to the provided post
		if deleted.PostID != arg.PostID {
			return newOpError(
				opDeleteComment,
				KindRelation,
				entComment,
				fmt.Errorf("comment with id %d does not belong to post with id %d", arg.CommentID, arg.PostID),
				withRelated(entPost, arg.PostID),
				withEntityID(arg.CommentID),
				withField("post_id"),
			)
		}

		// if deletion performed successfully or
		// the target comment is already deleted return
		if deleted.DeletedOk || deleted.IsDeleted {
			row = deleted
			return nil
		}

		// if the comment has children perform soft delete
		if deleted.HasChildren {

			comment, err := q.softDeleteComment(ctx, arg.CommentID)
			if err != nil {
				return sqlError(
					opDeleteComment,
					opDetails{
						userID:    arg.UserID,
						postID:    arg.PostID,
						commentID: arg.CommentID,
						entity:    entComment,
					},
					err,
				)
			}

			row = deleteCommentIfLeafRow{
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
		return newOpError(
			opDeleteComment,
			KindCorrupted,
			entComment,
			fmt.Errorf("cannot delete comment with id %d", arg.CommentID),
		)
	})

	if err != nil {
		return result, err
	}

	result = DeleteCommentTxResult{
		ID:          row.ID,
		UserID:      row.UserID,
		PostID:      row.PostID,
		IsDeleted:   row.IsDeleted,
		DeletedAt:   row.DeletedAt,
		HasChildren: row.HasChildren,
		DeletedOk:   row.DeletedOk,
		Success:     row.DeletedOk || row.IsDeleted,
	}

	return result, nil
}
