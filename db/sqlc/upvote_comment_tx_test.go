package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpvoteCommentTx(t *testing.T) {
	n := 10
	comment1 := createRandomComment(t)

	errs := make(chan error)
	results := make(chan UpvoteCommentTxResult)

	for range n {
		go func() {
			user := createRandomUser(t)

			result, err := testStore.UpvoteCommentTx(context.Background(), UpvoteCommentTxParams{
				UserID:    user.ID,
				CommentID: comment1.ID,
			})

			errs <- err
			results <- result
		}()
	}

	for range n {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		require.Equal(t, comment1.ID, result.Comment.ID)
		require.Equal(t, comment1.Body, result.Comment.Body)
		require.Equal(t, comment1.ID, result.CommentVote.CommentID)
		require.Equal(t, int64(1), result.CommentVote.Vote)
	}

	comment2, err := testStore.GetComment(context.Background(), comment1.ID)
	require.NoError(t, err)

	require.Equal(t, int64(n), comment2.Upvotes)
}
