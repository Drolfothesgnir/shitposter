package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type VotePostTxParams struct {
	UserID int64 `json:"user_id"`
	PostID int64 `json:"post_id"`
	Vote   int64 `json:"vote"`
}

type VotePostTxResult struct {
	Post     Post     `json:"post"`
	PostVote PostVote `json:"post_vote"`
}

func (store *SQLStore) VotePostTx(ctx context.Context, arg VotePostTxParams) (VotePostTxResult, error) {
	var result VotePostTxResult

	// Check if vote's value is correct: 1 or -1
	if arg.Vote != int64(1) && arg.Vote != int64(-1) {
		return result, ErrInvalidVoteValue
	}

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// Checking if user's vote for this post alread exist
		post_vote, err := q.GetPostVote(ctx, GetPostVoteParams{
			UserID: arg.UserID,
			PostID: arg.PostID,
		})

		// if it does not exist create new one
		if errors.Is(err, pgx.ErrNoRows) {
			result.PostVote, err = q.CreatePostVote(ctx, CreatePostVoteParams{
				UserID: arg.UserID,
				PostID: arg.PostID,
				Vote:   arg.Vote,
			})

			if err != nil {
				return err
			}

			// update post's Upvotes/Downvotes with new vote
			deltaDownvotes := 0
			deltaUpvotes := 0

			if arg.Vote == -1 {
				deltaDownvotes = 1
			} else {
				deltaUpvotes = 1
			}

			result.Post, err = q.UpdatePost(ctx, UpdatePostParams{
				ID:             arg.PostID,
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
		if post_vote.Vote == arg.Vote {
			return ErrDuplicateVote
		}

		// update vote's value
		result.PostVote, err = q.ChangePostVote(ctx, ChangePostVoteParams{
			ID:   post_vote.ID,
			Vote: arg.Vote,
		})

		if err != nil {
			return err
		}

		// if vote exists and its value is diffent from the one from the arg
		// then Post's Upvotes/Downvotes recalculated
		deltaDownvotes := -1
		deltaUpvotes := 1

		if arg.Vote == -1 {
			deltaDownvotes = 1
			deltaUpvotes = -1
		}

		result.Post, err = q.UpdatePost(ctx, UpdatePostParams{
			ID:             arg.PostID,
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
