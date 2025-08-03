package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type DeletePostVoteTxParams struct {
	VoteID int64 `json:"vote_id"`
}

type DeletePostVoteTxResult struct {
	Post Post `json:"post"`
}

func (store *SQLStore) DeletePostVoteTx(ctx context.Context, arg DeletePostVoteTxParams) (DeletePostVoteTxResult, error) {
	var result DeletePostVoteTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		vote, err := q.GetPostVoteByID(ctx, arg.VoteID)
		if err != nil {
			return err
		}

		result.Post, err = q.UpdatePost(ctx, UpdatePostParams{
			ID:             vote.PostID,
			DeltaUpvotes:   pgtype.Int8{Int64: -1, Valid: vote.Vote == 1},
			DeltaDownvotes: pgtype.Int8{Int64: -1, Valid: vote.Vote == -1},
		})

		if err != nil {
			return err
		}

		err = q.DeletePostVote(ctx, arg.VoteID)

		return err
	})

	return result, err
}
