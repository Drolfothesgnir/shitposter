package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// helper to create a child comment for a given parent comment
func createChildComment(t *testing.T, parent Comment) Comment {
	t.Helper()

	arg := CreateCommentParams{
		UserID:   parent.UserID,
		PostID:   parent.PostID,
		ParentID: pgtype.Int8{Int64: parent.ID, Valid: true},
		Body:     "child comment body",
	}

	comment, err := testStore.CreateComment(context.Background(), arg)
	require.NoError(t, err)
	require.NotZero(t, comment.ID)
	require.Equal(t, parent.ID, comment.ParentID.Int64)
	require.True(t, comment.ParentID.Valid)

	return comment
}

// 1) If comment is leaf it should be deleted completely
func TestDeleteCommentTx_LeafHardDelete(t *testing.T) {
	ctx := context.Background()

	// create a root comment with no children
	comment := createRandomComment(t)

	// sanity check: it exists
	gotBefore, err := testStore.GetComment(ctx, comment.ID)
	require.NoError(t, err)
	require.Equal(t, comment.ID, gotBefore.ID)

	// act
	err = testStore.DeleteCommentTx(ctx, comment.ID)
	require.NoError(t, err)

	// after: the comment should not be present in the db
	_, err = testStore.GetComment(ctx, comment.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

// 2. If the comment has children it should be soft-deleted,
// so children remain untouched
func TestDeleteCommentTx_NonLeafSoftDelete(t *testing.T) {
	ctx := context.Background()

	// create parent and child
	parent := createRandomComment(t)
	child := createChildComment(t, parent)

	// sanity: both exist
	gotParentBefore, err := testStore.GetComment(ctx, parent.ID)
	require.NoError(t, err)
	require.False(t, gotParentBefore.IsDeleted)

	gotChildBefore, err := testStore.GetComment(ctx, child.ID)
	require.NoError(t, err)
	require.False(t, gotChildBefore.IsDeleted)

	// act: delete the parent
	err = testStore.DeleteCommentTx(ctx, parent.ID)
	require.NoError(t, err)

	// parent should be soft-deleted
	gotParentAfter, err := testStore.GetComment(ctx, parent.ID)
	require.NoError(t, err)
	require.True(t, gotParentAfter.IsDeleted)
	require.Equal(t, "[deleted]", gotParentAfter.Body)
	// deleted_at is changed from the default value
	// require.False(t, gotParentAfter.DeletedAt.IsZero())

	// child should stay and NOT be deleted
	gotChildAfter, err := testStore.GetComment(ctx, child.ID)
	require.NoError(t, err)
	require.Equal(t, child.ID, gotChildAfter.ID)
	require.False(t, gotChildAfter.IsDeleted)
	require.Equal(t, gotChildBefore.Body, gotChildAfter.Body)
}

// 3) Operation must be idempotent / no-op for a non-existing comment
func TestDeleteCommentTx_NonExistingComment_NoError(t *testing.T) {
	ctx := context.Background()

	// choosing deliberately non-existent id
	const nonexistentID int64 = 9_999_999_999

	// act: first call
	err := testStore.DeleteCommentTx(ctx, nonexistentID)
	require.NoError(t, err)

	// act: second call - must not fall as well
	err = testStore.DeleteCommentTx(ctx, nonexistentID)
	require.NoError(t, err)
}

// 4) Optional: idempotency for alreade deleted leaf
func TestDeleteCommentTx_LeafIdempotent(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)

	// first call - remove as leaf
	err := testStore.DeleteCommentTx(ctx, comment.ID)
	require.NoError(t, err)

	// sanity: deleted for real
	_, err = testStore.GetComment(ctx, comment.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// second call - should be without error (no-op)
	err = testStore.DeleteCommentTx(ctx, comment.ID)
	require.NoError(t, err)
}
