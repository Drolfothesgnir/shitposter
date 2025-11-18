package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type InsertCommentTxParams struct {
	UserID    int64       `json:"user_id"`
	PostID    int64       `json:"post_id"`
	Body      string      `json:"body"`
	ParentID  pgtype.Int8 `json:"parent_id"`
	Upvotes   int64       `json:"upvotes"`
	Downvotes int64       `json:"downvotes"`
}

// TODO: Currently user is allowed to reply to his own comments to the infinite depth.
// I should limit this behavoiur.
func (s *SQLStore) InsertCommentTx(ctx context.Context, arg InsertCommentTxParams) (Comment, error) {
	var result Comment

	err := s.execTx(ctx, func(q *Queries) error {
		// in case when the comment is a root comment, depth will be 0
		var depth int32

		if arg.ParentID.Valid {
			// getting parent comment and locking it for updating parent comment's ID
			// though currently other functions do not alter entity ids
			parent, err := q.GetCommentWithLock(ctx, arg.ParentID.Int64)

			// if there is no parent comment when parent id is provided abort with error
			if err == pgx.ErrNoRows {
				return ErrParentCommentNotFound
			}

			// return generic error
			if err != nil {
				return err
			}

			// if parent comment's post_id and provided post_id differs abort
			if parent.PostID != arg.PostID {
				return ErrParentCommentPostIDMismatch
			}

			// check if comment is alive
			if parent.IsDeleted {
				return ErrParentCommentDeleted
			}

			// if everything is ok child will have parent's depth + 1
			depth = parent.Depth + 1
		}

		comment, err := q.CreateComment(ctx, CreateCommentParams{
			UserID:    arg.UserID,
			PostID:    arg.PostID,
			Body:      arg.Body,
			ParentID:  arg.ParentID,
			Depth:     depth,
			Upvotes:   arg.Upvotes,
			Downvotes: arg.Downvotes,
		})

		// instead of making another trip to the db to check if parent post exists
		// i'm trying to insert the new comment and check if there is foreign key violation,
		// that is if the parent post is missing
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23503" && pgErr.ConstraintName == "comments_post_id_fkey" {
				return ErrInvalidPostID
			}
		}

		if err != nil {
			return err
		}

		result = comment

		return nil
	})

	return result, err
}
