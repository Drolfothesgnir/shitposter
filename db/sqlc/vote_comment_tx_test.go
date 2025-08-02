package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// Test for proper 10 positive votes and 5 negative votes
func TestVoteCommentTx(t *testing.T) {
	n_pos := 10
	n_neg := 5
	comment1 := createRandomComment(t)

	errs := make(chan error)
	results := make(chan VoteCommentTxResult)

	for range n_pos {
		go func() {
			user := createRandomUser(t)

			result, err := testStore.VoteCommentTx(context.Background(), VoteCommentTxParams{
				UserID:    user.ID,
				CommentID: comment1.ID,
				Vote:      1,
			})

			errs <- err
			results <- result
		}()
	}

	for range n_pos {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result.CommentVote)
		require.NotEmpty(t, result.Comment)

		require.Equal(t, comment1.ID, result.Comment.ID)
		require.Equal(t, comment1.Body, result.Comment.Body)
		require.Equal(t, comment1.ID, result.CommentVote.CommentID)
		require.Equal(t, int64(1), result.CommentVote.Vote)
	}

	for range n_neg {
		go func() {
			user := createRandomUser(t)

			result, err := testStore.VoteCommentTx(context.Background(), VoteCommentTxParams{
				UserID:    user.ID,
				CommentID: comment1.ID,
				Vote:      -1,
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

		require.Equal(t, comment1.ID, result.Comment.ID)
		require.Equal(t, comment1.Body, result.Comment.Body)
		require.Equal(t, comment1.ID, result.CommentVote.CommentID)
		require.Equal(t, int64(-1), result.CommentVote.Vote)
	}

	comment2, err := testStore.GetComment(context.Background(), comment1.ID)
	require.NoError(t, err)

	require.Equal(t, int64(n_pos-n_neg), comment2.Upvotes-comment2.Downvotes)
}

func TestDuplicateVote(t *testing.T) {
	comment := createRandomComment(t)

	user := createRandomUser(t)

	_, err := testStore.VoteCommentTx(context.Background(), VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	})

	require.NoError(t, err)

	tx_result2, err := testStore.VoteCommentTx(context.Background(), VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	})

	require.Empty(t, tx_result2)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDuplicateVote)
}

func TestChangeVote(t *testing.T) {
	comment := createRandomComment(t)

	user := createRandomUser(t)

	tx_result1, err := testStore.VoteCommentTx(context.Background(), VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), tx_result1.Comment.Upvotes)
	require.Zero(t, tx_result1.Comment.Downvotes)

	tx_result2, err := testStore.VoteCommentTx(context.Background(), VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      -1,
	})
	fmt.Printf("upvotes: %d", tx_result2.Comment.Upvotes)
	require.NoError(t, err)
	require.Equal(t, int64(1), tx_result2.Comment.Downvotes)
	require.Zero(t, tx_result2.Comment.Upvotes)
	require.Equal(t, int64(-1), tx_result2.CommentVote.Vote)
}

func TestBadCommentID(t *testing.T) {
	user := createRandomUser(t)

	tx_result, err := testStore.VoteCommentTx(context.Background(), VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: -1,
		Vote:      1,
	})

	require.Empty(t, tx_result)
	var pgErr *pgconn.PgError
	require.Error(t, err)
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23503", pgErr.Code)
	require.Equal(t, "comment_votes_comment_id_fkey", pgErr.ConstraintName)
}

func TestBadVote(t *testing.T) {
	comment := createRandomComment(t)
	tx_result, err := testStore.VoteCommentTx(context.Background(), VoteCommentTxParams{
		UserID:    comment.UserID,
		CommentID: comment.ID,
		Vote:      5,
	})

	require.Empty(t, tx_result)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidVoteValue)
}
