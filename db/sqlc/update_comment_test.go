package db

import (
	"context"
	"testing"
	"time"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/stretchr/testify/require"
)

// Happy path: owner updates his own comment on the correct post.
func TestUpdateComment_Success(t *testing.T) {
	ctx := context.Background()

	// Create a random comment (with its own user and post).
	original := createRandomComment(t)

	newBody := util.RandomString(20)

	arg := UpdateCommentParams{
		UserID:    original.UserID,
		PostID:    original.PostID,
		CommentID: original.ID,
		Body:      newBody,
	}

	before := time.Now()

	res, err := testStore.UpdateComment(ctx, arg)
	require.NoError(t, err)

	// Returned payload
	require.EqualValues(t, original.ID, res.ID)
	require.Equal(t, newBody, res.Body)
	require.False(t, res.LastModifiedAt.Before(before)) // updated_at >= before

	// Check DB state
	updated, err := testStore.getCommentWithLock(ctx, original.ID)
	require.NoError(t, err)

	require.Equal(t, newBody, updated.Body)
	require.False(t, updated.LastModifiedAt.Before(before))
}

// Non-existent comment: should return KindNotFound with entity=comment.
func TestUpdateComment_NotFound(t *testing.T) {
	ctx := context.Background()

	user := createRandomUser(t)
	post := createRandomPost(t)

	nonExistingCommentID := int64(9_999_999_999)

	arg := UpdateCommentParams{
		UserID:    user.ID,
		PostID:    post.ID,
		CommentID: nonExistingCommentID,
		Body:      "does not matter",
	}

	_, err := testStore.UpdateComment(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opUpdateComment, opErr.Op)
	require.Equal(t, KindNotFound, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)
	require.EqualValues(t, nonExistingCommentID, opErr.EntityID)
}

// Comment exists but belongs to a different user.
// With your new precedence this should be KindPermission (before deleted/relation).
func TestUpdateComment_PermissionDenied(t *testing.T) {
	ctx := context.Background()

	// Owner and his comment
	ownerComment := createRandomComment(t)

	// Different user trying to update owner's comment
	otherUser := createRandomUser(t)

	originalBody := ownerComment.Body

	arg := UpdateCommentParams{
		UserID:    otherUser.ID, // wrong user
		PostID:    ownerComment.PostID,
		CommentID: ownerComment.ID,
		Body:      util.RandomString(20),
	}

	_, err := testStore.UpdateComment(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opUpdateComment, opErr.Op)
	require.Equal(t, KindPermission, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)
	require.EqualValues(t, ownerComment.ID, opErr.EntityID)
	require.EqualValues(t, otherUser.ID, opErr.UserID)

	// Ensure DB was not updated
	reloaded, err := testStore.getCommentWithLock(ctx, ownerComment.ID)
	require.NoError(t, err)
	require.Equal(t, originalBody, reloaded.Body)
}

// Comment is soft-deleted: owner tries to update, but update is not allowed.
// With the new precedence, deleted comes after permission, so we use the owner here.
func TestUpdateComment_DeletedComment(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)

	// Soft delete the comment first
	deleted, err := testStore.softDeleteComment(ctx, comment.ID)
	require.NoError(t, err)
	require.True(t, deleted.IsDeleted)

	arg := UpdateCommentParams{
		UserID:    comment.UserID, // correct owner
		PostID:    comment.PostID, // correct post
		CommentID: comment.ID,
		Body:      util.RandomString(20),
	}

	_, err = testStore.UpdateComment(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opUpdateComment, opErr.Op)
	require.Equal(t, KindDeleted, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)
	require.EqualValues(t, comment.ID, opErr.EntityID)

	// Body should stay "[deleted]" after failed update
	reloaded, err := testStore.getCommentWithLock(ctx, comment.ID)
	require.NoError(t, err)

	require.True(t, reloaded.IsDeleted)
	require.Equal(t, "[deleted]", reloaded.Body)
}

// Comment belongs to a different post: user is correct, but post_id is wrong.
// With the new precedence, relation is checked after permission and deletion.
func TestUpdateComment_PostMismatch(t *testing.T) {
	ctx := context.Background()

	// Original comment
	comment := createRandomComment(t)

	// Different post (does not own the comment)
	otherPost := createRandomPost(t)

	originalBody := comment.Body

	arg := UpdateCommentParams{
		UserID:    comment.UserID, // correct owner
		PostID:    otherPost.ID,   // wrong post_id
		CommentID: comment.ID,
		Body:      util.RandomString(20),
	}

	_, err := testStore.UpdateComment(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opUpdateComment, opErr.Op)
	require.Equal(t, KindRelation, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)
	require.EqualValues(t, comment.ID, opErr.EntityID)
	require.Equal(t, entPost, opErr.RelatedEntity)
	require.EqualValues(t, otherPost.ID, opErr.RelatedEntityID)

	// Ensure DB was not updated
	reloaded, err := testStore.getCommentWithLock(ctx, comment.ID)
	require.NoError(t, err)
	require.Equal(t, originalBody, reloaded.Body)
}
