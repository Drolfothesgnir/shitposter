package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// Test for proper 10 positive votes and 5 negative votes
func TestVotePostTx(t *testing.T) {
	n_pos := 10
	n_neg := 5
	Post1 := createRandomPost(t)

	errs := make(chan error)
	results := make(chan VotePostTxResult)

	for range n_pos {
		go func() {
			user := createRandomUser(t)

			result, err := testStore.VotePostTx(context.Background(), VotePostTxParams{
				UserID: user.ID,
				PostID: Post1.ID,
				Vote:   1,
			})

			errs <- err
			results <- result
		}()
	}

	for range n_pos {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result.PostVote)
		require.NotEmpty(t, result.Post)

		require.Equal(t, Post1.ID, result.Post.ID)
		require.Equal(t, Post1.Body, result.Post.Body)
		require.Equal(t, Post1.ID, result.PostVote.PostID)
		require.Equal(t, int64(1), result.PostVote.Vote)
	}

	for range n_neg {
		go func() {
			user := createRandomUser(t)

			result, err := testStore.VotePostTx(context.Background(), VotePostTxParams{
				UserID: user.ID,
				PostID: Post1.ID,
				Vote:   -1,
			})

			errs <- err
			results <- result
		}()
	}

	for range n_neg {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		require.Equal(t, Post1.ID, result.Post.ID)
		require.Equal(t, Post1.Body, result.Post.Body)
		require.Equal(t, Post1.ID, result.PostVote.PostID)
		require.Equal(t, int64(-1), result.PostVote.Vote)
	}

	Post2, err := testStore.GetPost(context.Background(), Post1.ID)
	require.NoError(t, err)

	require.Equal(t, int64(n_pos-n_neg), Post2.Upvotes-Post2.Downvotes)
}

func TestDuplicateVote(t *testing.T) {
	Post := createRandomPost(t)

	user := createRandomUser(t)

	_, err := testStore.VotePostTx(context.Background(), VotePostTxParams{
		UserID: user.ID,
		PostID: Post.ID,
		Vote:   1,
	})

	require.NoError(t, err)

	tx_result2, err := testStore.VotePostTx(context.Background(), VotePostTxParams{
		UserID: user.ID,
		PostID: Post.ID,
		Vote:   1,
	})

	require.Empty(t, tx_result2)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDuplicateVote)
}

func TestChangeVote(t *testing.T) {
	Post := createRandomPost(t)

	user := createRandomUser(t)

	tx_result1, err := testStore.VotePostTx(context.Background(), VotePostTxParams{
		UserID: user.ID,
		PostID: Post.ID,
		Vote:   1,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), tx_result1.Post.Upvotes)
	require.Zero(t, tx_result1.Post.Downvotes)

	tx_result2, err := testStore.VotePostTx(context.Background(), VotePostTxParams{
		UserID: user.ID,
		PostID: Post.ID,
		Vote:   -1,
	})
	fmt.Printf("upvotes: %d", tx_result2.Post.Upvotes)
	require.NoError(t, err)
	require.Equal(t, int64(1), tx_result2.Post.Downvotes)
	require.Zero(t, tx_result2.Post.Upvotes)
	require.Equal(t, int64(-1), tx_result2.PostVote.Vote)
}

func TestBadPostID(t *testing.T) {
	user := createRandomUser(t)

	tx_result, err := testStore.VotePostTx(context.Background(), VotePostTxParams{
		UserID: user.ID,
		PostID: -1,
		Vote:   1,
	})

	require.Empty(t, tx_result)
	var pgErr *pgconn.PgError
	require.Error(t, err)
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23503", pgErr.Code)
	require.Equal(t, "post_votes_post_id_fkey", pgErr.ConstraintName)
}

func TestBadVote(t *testing.T) {
	Post := createRandomPost(t)
	tx_result, err := testStore.VotePostTx(context.Background(), VotePostTxParams{
		UserID: Post.UserID,
		PostID: Post.ID,
		Vote:   5,
	})

	require.Empty(t, tx_result)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidVoteValue)
}
