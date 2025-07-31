package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestCreateCommentVote(t *testing.T) {
	user := createRandomUser(t)

	comment := createRandomComment(t)

	arg := CreateCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	}

	vote, err := testStore.CreateCommentVote(context.Background(), arg)
	require.NoError(t, err)

	require.Equal(t, arg.UserID, vote.UserID)
	require.Equal(t, arg.CommentID, vote.CommentID)
	require.Equal(t, arg.Vote, vote.Vote)
	require.NotZero(t, vote.CreatedAt)
}

func TestChangeCommentVote(t *testing.T) {
	user := createRandomUser(t)

	comment := createRandomComment(t)

	arg1 := CreateCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	}

	vote1, err := testStore.CreateCommentVote(context.Background(), arg1)
	require.NoError(t, err)

	arg2 := ChangeCommentVoteParams{
		ID:   vote1.ID,
		Vote: -1,
	}

	vote2, err := testStore.ChangeCommentVote(context.Background(), arg2)
	require.NoError(t, err)

	require.Equal(t, vote1.ID, vote2.ID)
	require.Equal(t, vote1.Vote, -1*vote2.Vote)
	require.Equal(t, vote1.CommentID, vote2.CommentID)
	require.Equal(t, vote1.CreatedAt, vote2.CreatedAt)
	require.WithinDuration(t, time.Now(), vote2.LastModifiedAt, 10*time.Second)
}

func TestGetCommentVote(t *testing.T) {
	user := createRandomUser(t)

	comment := createRandomComment(t)

	arg := CreateCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	}

	vote1, err := testStore.CreateCommentVote(context.Background(), arg)
	require.NoError(t, err)

	vote2, err := testStore.GetCommentVote(context.Background(), vote1.ID)
	require.NoError(t, err)

	require.Equal(t, vote1.ID, vote2.ID)
	require.Equal(t, vote1.CommentID, vote2.CommentID)
	require.Equal(t, vote1.CreatedAt, vote2.CreatedAt)
	require.Equal(t, vote1.Vote, vote2.Vote)
	require.WithinDuration(t, vote1.CreatedAt, vote2.CreatedAt, time.Second)
}

func TestDeleteCommentVote(t *testing.T) {
	comment := createRandomComment(t)

	user := createRandomUser(t)

	arg := CreateCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	}

	vote1, err := testStore.CreateCommentVote(context.Background(), arg)
	require.NoError(t, err)

	err = testStore.DeleteCommentVote(context.Background(), vote1.ID)
	require.NoError(t, err)

	_, err = testStore.GetCommentVote(context.Background(), vote1.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
