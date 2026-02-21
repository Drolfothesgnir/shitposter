package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const opUpdateComment = "update-comment"

type UpdateCommentParams struct {
	UserID    int64
	PostID    int64
	CommentID int64
	Body      string
}

type UpdateCommentResult struct {
	ID             int64     `json:"id"`
	Body           string    `json:"body"`
	LastModifiedAt time.Time `json:"last_modified_at"`
}

// UpdateComment updates the body of a comment identified by CommentID.
// The caller must own the comment and the comment must belong to the given post.
// Returns KindNotFound if the comment does not exist, KindPermission if the comment
// belongs to another user, KindDeleted if the comment is soft-deleted, KindRelation
// if the comment belongs to a different post, or KindInternal on database errors.
func (s *SQLStore) UpdateComment(ctx context.Context, arg UpdateCommentParams) (UpdateCommentResult, error) {
	updateResult, err := s.updateComment(ctx, updateCommentParams{
		PUserID:    arg.UserID,
		PPostID:    arg.PostID,
		PCommentID: arg.CommentID,
		PBody:      arg.Body,
	})

	// 1. Comment doesn't exist
	if errors.Is(err, pgx.ErrNoRows) {
		return UpdateCommentResult{}, notFoundError(opUpdateComment, entComment, arg.CommentID)
	}

	// 2. Internal error
	if err != nil {
		opErr := sqlError(
			opUpdateComment,
			opDetails{
				userID:    arg.UserID,
				postID:    arg.PostID,
				commentID: arg.CommentID,
				entity:    entComment,
			},
			err,
		)

		return UpdateCommentResult{}, opErr
	}

	// 3. Update performed successfully
	if updateResult.Updated {
		res := UpdateCommentResult{
			ID:             updateResult.ID,
			Body:           updateResult.Body,
			LastModifiedAt: updateResult.LastModifiedAt,
		}

		return res, nil
	}

	// 4. Update didn't happen, but the target comment exists.
	//
	//    At this point updateResult comes from the update_comment(...) SQL function.
	//    The contract of that function is:
	//      - 0 rows returned            -> comment not found (pgx.ErrNoRows)
	//      - 1 row with Updated = true  -> update applied
	//      - 1 row with Updated = false -> comment exists, but one of the predicates failed:
	//           * user_id   != arg.UserID   (caller is not the owner)
	//           * post_id   != arg.PostID   (comment belongs to a different post)
	//           * is_deleted = true         (comment has been soft-deleted)
	//
	//    For the "not updated but exists" case we map the state to domain errors
	//    in the following precedence order:
	//      1) the comment does not belong to the caller       -> KindPermission
	//      2) the comment is soft-deleted                     -> KindDeleted
	//      3) the comment belongs to a different post         -> KindRelation
	//
	//    If none of the checks below trigger, we fall back to KindInternal
	//    to avoid silently swallowing an impossible state if the SQL logic
	//    changes in the future.

	// 4.1 The comment belongs to someone else - most significant reason
	if updateResult.UserID != arg.UserID {
		opErr := newOpError(
			opUpdateComment,
			KindPermission,
			entComment,
			fmt.Errorf("comment with id %d does not belong to user with id %d", arg.CommentID, arg.UserID),
			withEntityID(arg.CommentID),
			withUser(arg.UserID),
		)

		return UpdateCommentResult{}, opErr
	}

	// 4.2 The comment is deleted
	if updateResult.IsDeleted {
		opErr := newOpError(
			opUpdateComment,
			KindDeleted,
			entComment,
			fmt.Errorf("comment with id %d is deleted and cannot be updated", arg.CommentID),
			withEntityID(arg.CommentID),
		)

		return UpdateCommentResult{}, opErr
	}

	// 4.3 The comment belongs to a different post
	if updateResult.PostID != arg.PostID {
		opErr := newOpError(
			opUpdateComment,
			KindRelation,
			entComment,
			fmt.Errorf("comment with id %d does not belong to post with id %d", arg.CommentID, arg.PostID),
			withEntityID(arg.CommentID),
			withRelated(entPost, arg.PostID),
		)

		return UpdateCommentResult{}, opErr
	}

	// 4.4 Guarding fallback: in theory you can't fall into this case,
	//    but if the business logic above will change it's better to return error.
	opErr := newOpError(
		opUpdateComment,
		KindInternal,
		entComment,
		fmt.Errorf("comment update failed for unknown reason (comment_id=%d, user_id=%d, post_id=%d)", arg.CommentID, arg.UserID, arg.PostID),
		withEntityID(arg.CommentID),
	)

	return UpdateCommentResult{}, opErr
}
