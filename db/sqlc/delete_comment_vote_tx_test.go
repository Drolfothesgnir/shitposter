package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteCommentVoteTx(t *testing.T) {
	comment := createRandomComment(t)

	arg1 := VoteCommentTxParams{
		UserID:    comment.UserID,
		CommentID: comment.ID,
		Vote:      1,
	}

	tx_result1, err := testStore.VoteCommentTx(context.Background(), arg1)
	require.NoError(t, err)

	require.Equal(t, int64(1), tx_result1.Comment.Upvotes)

	arg2 := DeleteCommentVoteTxParams{
		VoteID: tx_result1.CommentVote.ID,
	}

	tx_result2, err := testStore.DeleteCommentVoteTx(context.Background(), arg2)

	require.NoError(t, err)

	require.Equal(t, int64(0), tx_result2.Comment.Upvotes)
}
