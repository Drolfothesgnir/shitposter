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

// 1) If comment is a leaf and owned by the user, it should be hard-deleted.
func TestDeleteCommentTx_LeafHardDelete(t *testing.T) {
	ctx := context.Background()

	// create a root comment with no children
	comment := createRandomComment(t)

	// sanity check: it exists
	gotBefore, err := testStore.GetComment(ctx, comment.ID)
	require.NoError(t, err)
	require.Equal(t, comment.ID, gotBefore.ID)

	// act: delete as owner, correct post id
	result, err := testStore.DeleteCommentTx(ctx, DeleteCommentTxParams{
		CommentID: comment.ID,
		UserID:    comment.UserID,
		PostID:    comment.PostID,
	})
	require.NoError(t, err)

	// transaction-level success flag
	require.True(t, result.Success)
	// hard delete: row is gone from DB
	_, err = testStore.GetComment(ctx, comment.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// semantic properties from result:
	require.Equal(t, comment.ID, result.ID)
	require.Equal(t, comment.UserID, result.UserID)
	require.Equal(t, comment.PostID, result.PostID)
	require.False(t, result.IsDeleted)   // hard delete, not soft
	require.False(t, result.HasChildren) // leaf
	require.True(t, result.DeletedOk)    // hard delete succeeded inside SQL
}

//  2. If the comment has children, it should be soft-deleted,
//     children remain untouched, and operation is still considered success.
func TestDeleteCommentTx_NonLeafSoftDelete(t *testing.T) {
	ctx := context.Background()

	// create parent and child
	parent := createRandomComment(t)
	child := createChildComment(t, parent)

	// sanity: both exist and not deleted
	gotParentBefore, err := testStore.GetComment(ctx, parent.ID)
	require.NoError(t, err)
	require.False(t, gotParentBefore.IsDeleted)

	gotChildBefore, err := testStore.GetComment(ctx, child.ID)
	require.NoError(t, err)
	require.False(t, gotChildBefore.IsDeleted)

	// act: delete the parent as rightful owner
	result, err := testStore.DeleteCommentTx(ctx, DeleteCommentTxParams{
		CommentID: parent.ID,
		UserID:    parent.UserID,
		PostID:    parent.PostID,
	})
	require.NoError(t, err)

	// overall operation marked as successful
	require.True(t, result.Success)
	require.True(t, result.HasChildren)

	// parent should be soft-deleted
	gotParentAfter, err := testStore.GetComment(ctx, parent.ID)
	require.NoError(t, err)
	require.True(t, gotParentAfter.IsDeleted)
	require.Equal(t, "[deleted]", gotParentAfter.Body)
	require.True(t, gotParentAfter.DeletedAt.After(gotParentBefore.CreatedAt))

	// child should stay and NOT be deleted
	gotChildAfter, err := testStore.GetComment(ctx, child.ID)
	require.NoError(t, err)
	require.Equal(t, child.ID, gotChildAfter.ID)
	require.False(t, gotChildAfter.IsDeleted)
	require.Equal(t, gotChildBefore.Body, gotChildAfter.Body)
}

// 3) Non-existing comment: should return ErrEntityNotFound and !Success.
func TestDeleteCommentTx_NonExistingComment_NotFound(t *testing.T) {
	ctx := context.Background()

	const nonexistentID int64 = 9_999_999_999

	// user and post ids here don't matter, record doesn't exist anyway
	result, err := testStore.DeleteCommentTx(ctx, DeleteCommentTxParams{
		CommentID: nonexistentID,
		UserID:    1,
		PostID:    1,
	})

	require.Error(t, err)
	require.ErrorIs(t, err, ErrEntityNotFound)
	require.False(t, result.Success)
}

// 4) Repeated delete of the same comment: first succeeds, second returns ErrEntityNotFound.
func TestDeleteCommentTx_LeafDeleteThenNotFound(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)

	// first call - hard delete as leaf
	result1, err := testStore.DeleteCommentTx(ctx, DeleteCommentTxParams{
		CommentID: comment.ID,
		UserID:    comment.UserID,
		PostID:    comment.PostID,
	})
	require.NoError(t, err)
	require.True(t, result1.Success)

	// sanity: really gone
	_, err = testStore.GetComment(ctx, comment.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// second call - now it should be "not found", not silent success
	result2, err := testStore.DeleteCommentTx(ctx, DeleteCommentTxParams{
		CommentID: comment.ID,
		UserID:    comment.UserID,
		PostID:    comment.PostID,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrEntityNotFound)
	require.False(t, result2.Success)
}

// 5) Trying to delete someone else's comment should return ErrEntityDoesNotBelongToUser.
func TestDeleteCommentTx_ForeignUserForbidden(t *testing.T) {
	ctx := context.Background()

	ownerComment := createRandomComment(t)
	foreignUser := createRandomUser(t)

	// sanity
	require.NotEqual(t, ownerComment.UserID, foreignUser.ID)

	result, err := testStore.DeleteCommentTx(ctx, DeleteCommentTxParams{
		CommentID: ownerComment.ID,
		UserID:    foreignUser.ID,      // not the owner
		PostID:    ownerComment.PostID, // correct post
	})

	require.Error(t, err)
	require.ErrorIs(t, err, ErrEntityDoesNotBelongToUser)
	require.False(t, result.Success)

	// ensure original comment still exists and not deleted
	got, err := testStore.GetComment(ctx, ownerComment.ID)
	require.NoError(t, err)
	require.False(t, got.IsDeleted)
}

// 6) Trying to delete comment with wrong post ID should return ErrInvalidPostID.
func TestDeleteCommentTx_WrongPostID(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	anotherPost := createRandomPost(t)

	// sanity
	require.NotEqual(t, comment.PostID, anotherPost.ID)

	result, err := testStore.DeleteCommentTx(ctx, DeleteCommentTxParams{
		CommentID: comment.ID,
		UserID:    comment.UserID,
		PostID:    anotherPost.ID, // wrong post
	})

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidPostID)
	require.False(t, result.Success)

	// ensure comment still exists and not deleted
	got, err := testStore.GetComment(ctx, comment.ID)
	require.NoError(t, err)
	require.False(t, got.IsDeleted)
}
