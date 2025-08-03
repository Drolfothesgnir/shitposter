package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestCreatePostVote(t *testing.T) {
	user := createRandomUser(t)

	post := createRandomPost(t)

	arg := CreatePostVoteParams{
		UserID: user.ID,
		PostID: post.ID,
		Vote:   1,
	}

	vote, err := testStore.CreatePostVote(context.Background(), arg)
	require.NoError(t, err)

	require.Equal(t, arg.UserID, vote.UserID)
	require.Equal(t, arg.PostID, vote.PostID)
	require.Equal(t, arg.Vote, vote.Vote)
	require.NotZero(t, vote.CreatedAt)
}

func TestChangePostVote(t *testing.T) {
	user := createRandomUser(t)

	post := createRandomPost(t)

	arg1 := CreatePostVoteParams{
		UserID: user.ID,
		PostID: post.ID,
		Vote:   1,
	}

	vote1, err := testStore.CreatePostVote(context.Background(), arg1)
	require.NoError(t, err)

	arg2 := ChangePostVoteParams{
		ID:   vote1.ID,
		Vote: -1,
	}

	vote2, err := testStore.ChangePostVote(context.Background(), arg2)
	require.NoError(t, err)

	require.Equal(t, vote1.ID, vote2.ID)
	require.Equal(t, vote1.Vote, -1*vote2.Vote)
	require.Equal(t, vote1.PostID, vote2.PostID)
	require.Equal(t, vote1.CreatedAt, vote2.CreatedAt)
	require.WithinDuration(t, time.Now(), vote2.LastModifiedAt, 10*time.Second)
}

func TestGetPostVote(t *testing.T) {
	user := createRandomUser(t)

	post := createRandomPost(t)

	arg := CreatePostVoteParams{
		UserID: user.ID,
		PostID: post.ID,
		Vote:   1,
	}

	vote1, err := testStore.CreatePostVote(context.Background(), arg)
	require.NoError(t, err)

	vote2, err := testStore.GetPostVoteByID(context.Background(), vote1.ID)
	require.NoError(t, err)

	require.Equal(t, vote1.ID, vote2.ID)
	require.Equal(t, vote1.PostID, vote2.PostID)
	require.Equal(t, vote1.CreatedAt, vote2.CreatedAt)
	require.Equal(t, vote1.Vote, vote2.Vote)
	require.WithinDuration(t, vote1.CreatedAt, vote2.CreatedAt, time.Second)
}

func TestDeletePostVote(t *testing.T) {
	post := createRandomPost(t)

	user := createRandomUser(t)

	arg := CreatePostVoteParams{
		UserID: user.ID,
		PostID: post.ID,
		Vote:   1,
	}

	vote1, err := testStore.CreatePostVote(context.Background(), arg)
	require.NoError(t, err)

	err = testStore.DeletePostVote(context.Background(), vote1.ID)
	require.NoError(t, err)

	_, err = testStore.GetPostVoteByID(context.Background(), vote1.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
