package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeletePostVoteTx(t *testing.T) {
	Post := createRandomPost(t)

	arg1 := VotePostTxParams{
		UserID: Post.UserID,
		PostID: Post.ID,
		Vote:   1,
	}

	tx_result1, err := testStore.VotePostTx(context.Background(), arg1)
	require.NoError(t, err)

	require.Equal(t, int64(1), tx_result1.Post.Upvotes)

	arg2 := DeletePostVoteTxParams{
		VoteID: tx_result1.PostVote.ID,
	}

	tx_result2, err := testStore.DeletePostVoteTx(context.Background(), arg2)

	require.NoError(t, err)

	require.Equal(t, int64(0), tx_result2.Post.Upvotes)
}
