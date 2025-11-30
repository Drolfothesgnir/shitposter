package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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
			// getting parent comment and preventing its deletion from other queries
			parentID := arg.ParentID.Int64
			parent, err := q.GetCommentWithLock(ctx, parentID)

			// if there is no parent comment when parent id is provided abort with error
			if err == pgx.ErrNoRows {
				return withEntityID(
					baseError(
						"insert-comment",
						"comment",
						KindNotFound,
						fmt.Errorf("cannot reply to the comment with id: %d, the comment doesn't exist", parentID),
					),
					parentID,
				)
			}

			// return generic error
			if err != nil {
				return sqlError(
					"insert-comment",
					opDetails{
						userID:    arg.UserID,
						postID:    arg.PostID,
						commentID: parentID,
						entity:    "comment",
					},
					err,
				)
			}

			// if parent comment's post_id and provided post_id differs abort
			if parent.PostID != arg.PostID {

				return withEntityID(
					baseError(
						"insert-comment",
						"comment",
						KindRelation,
						fmt.Errorf(
							"cannot reply to comment %d for post %d: parent comment belongs to post %d",
							parentID,
							arg.PostID,
							parent.PostID,
						),
					),
					parentID,
				)
			}

			// check if comment is alive
			if parent.IsDeleted {
				parentID := parent.ID

				return withEntityID(
					baseError(
						"insert-comment",
						"comment",
						KindDeleted,
						fmt.Errorf("cannot reply to a deleted comment with id: %d", parentID),
					),
					parentID,
				)
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

		if err != nil {
			return sqlError(
				"insert-comment",
				opDetails{
					userID:    arg.UserID,
					postID:    arg.PostID,
					commentID: arg.ParentID.Int64,
					entity:    "comment",
				},
				err,
			)
		}

		result = comment

		return nil
	})

	return result, err
}
