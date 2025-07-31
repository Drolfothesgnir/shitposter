package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateComment(t *testing.T) {
	post := createRandomPost(t)

	arg := CreateCommentParams{
		PUserID: post.UserID,
		PPostID: post.ID,
		PBody:   util.RandomString(10),
	}

	comment, err := testStore.CreateComment(context.Background(), arg)
	require.NoError(t, err)

	require.Equal(t, arg.PUserID, comment.UserID)
	require.Equal(t, arg.PPostID, comment.PostID)
	require.Equal(t, arg.PBody, comment.Body)
	require.Equal(t, int32(0), comment.Depth)
	require.Equal(t, fmt.Sprint(comment.ID), comment.Path)
	require.Zero(t, comment.Downvotes)
	require.Zero(t, comment.Upvotes)
}

func TestCreateReplyComment(t *testing.T) {
	post := createRandomPost(t)

	user := createRandomUser(t)

	arg1 := CreateCommentParams{
		PUserID: post.UserID,
		PPostID: post.ID,
		PBody:   util.RandomString(10),
	}

	comment1, err := testStore.CreateComment(context.Background(), arg1)
	require.NoError(t, err)

	arg2 := CreateCommentParams{
		PUserID:     user.ID,
		PPostID:     post.ID,
		PBody:       util.RandomString(10),
		PParentPath: pgtype.Text{String: fmt.Sprint(comment1.ID), Valid: true},
	}

	comment2, err := testStore.CreateComment(context.Background(), arg2)
	require.NoError(t, err)

	path := fmt.Sprintf("%d.%d", comment1.ID, comment2.ID)

	require.Equal(t, arg2.PUserID, comment2.UserID)
	require.Equal(t, arg2.PPostID, comment2.PostID)
	require.Equal(t, arg2.PBody, comment2.Body)
	require.Equal(t, int32(1), comment2.Depth)
	require.Equal(t, path, comment2.Path)
	require.Zero(t, comment2.Downvotes)
	require.Zero(t, comment2.Upvotes)
}
