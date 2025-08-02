package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type DeleteCommentVoteTxParams struct {
	VoteID int64 `json:"vote_id"`
}

type DeleteCommentVoteTxResult struct {
	Comment Comment `json:"comment"`
}

func (store *SQLStore) DeleteCommentVoteTx(ctx context.Context, arg DeleteCommentVoteTxParams) (DeleteCommentVoteTxResult, error) {
	var result DeleteCommentVoteTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		vote, err := q.GetCommentVote(ctx, arg.VoteID)
		if err != nil {
			return err
		}

		result.Comment, err = q.UpdateComment(ctx, UpdateCommentParams{
			ID:             vote.CommentID,
			DeltaUpvotes:   pgtype.Int8{Int64: -1, Valid: vote.Vote == 1},
			DeltaDownvotes: pgtype.Int8{Int64: -1, Valid: vote.Vote == -1},
		})

		if err != nil {
			return err
		}

		err = q.DeleteCommentVote(ctx, arg.VoteID)

		return err
	})

	return result, err
}
