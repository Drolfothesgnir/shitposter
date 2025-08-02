package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type VoteCommentTxParams struct {
	UserID    int64 `json:"user_id"`
	CommentID int64 `json:"comment_id"`
	Vote      int64 `json:"vote"`
}

type VoteCommentTxResult struct {
	Comment     Comment     `json:"comment"`
	CommentVote CommentVote `json:"comment_vote"`
}

func (store *SQLStore) VoteCommentTx(ctx context.Context, arg VoteCommentTxParams) (VoteCommentTxResult, error) {
	var result VoteCommentTxResult

	// Check if vote's value is correct: 1 or -1
	if arg.Vote != int64(1) && arg.Vote != int64(-1) {
		return result, ErrInvalidVoteValue
	}

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// Checking if user's vote for this comment alread exist
		comment_vote, err := q.GetExistingVote(ctx, GetExistingVoteParams{
			UserID:    arg.UserID,
			CommentID: arg.CommentID,
		})

		// if it does not exist create new one
		if errors.Is(err, pgx.ErrNoRows) {
			result.CommentVote, err = q.CreateCommentVote(ctx, CreateCommentVoteParams{
				UserID:    arg.UserID,
				CommentID: arg.CommentID,
				Vote:      arg.Vote,
			})

			if err != nil {
				return err
			}

			// update comments Upvotes/Downvotes with new vote
			deltaDownvotes := 0
			deltaUpvotes := 0

			if arg.Vote == -1 {
				deltaDownvotes = 1
			} else {
				deltaUpvotes = 1
			}

			result.Comment, err = q.UpdateComment(ctx, UpdateCommentParams{
				ID:             arg.CommentID,
				DeltaUpvotes:   pgtype.Int8{Int64: int64(deltaUpvotes), Valid: true},
				DeltaDownvotes: pgtype.Int8{Int64: int64(deltaDownvotes), Valid: true},
			})

			if err != nil {
				return err
			}

			return nil
		}

		// if the vote exist and it has same value, 1 or -1,
		// then it's a duplicate vote and error is returned
		if comment_vote.Vote == arg.Vote {
			return ErrDuplicateVote
		}

		// update vote's value
		result.CommentVote, err = q.ChangeCommentVote(ctx, ChangeCommentVoteParams{
			ID:   comment_vote.ID,
			Vote: arg.Vote,
		})

		if err != nil {
			return err
		}

		// if vote exists and its value is diffent from the one from the arg
		// then comment's Upvotes/Downvotes recalculated
		deltaDownvotes := -1
		deltaUpvotes := 1

		if arg.Vote == -1 {
			deltaDownvotes = 1
			deltaUpvotes = -1
		}

		result.Comment, err = q.UpdateComment(ctx, UpdateCommentParams{
			ID:             arg.CommentID,
			DeltaUpvotes:   pgtype.Int8{Int64: int64(deltaUpvotes), Valid: true},
			DeltaDownvotes: pgtype.Int8{Int64: int64(deltaDownvotes), Valid: true},
		})

		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}
