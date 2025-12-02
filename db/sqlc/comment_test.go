package db

import (
	"context"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/stretchr/testify/require"
)

func createRandomComment(t *testing.T) Comment {
	t.Helper()

	ctx := context.Background()
	post := createRandomPost(t)

	arg := createCommentParams{
		UserID: post.UserID,
		PostID: post.ID,
		Body:   util.RandomString(10),
	}

	comment, err := testStore.createComment(ctx, arg)
	require.NoError(t, err)

	require.Equal(t, arg.UserID, comment.UserID)
	require.Equal(t, arg.PostID, comment.PostID)
	require.Equal(t, arg.Body, comment.Body)
	require.Equal(t, int32(0), comment.Depth)
	require.False(t, comment.ParentID.Valid)
	require.Zero(t, comment.Downvotes)
	require.Zero(t, comment.Upvotes)

	return comment
}

func TestCreateComment(t *testing.T) {
	createRandomComment(t)
}
