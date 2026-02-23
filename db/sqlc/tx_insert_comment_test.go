package db

import (
	"context"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// Root comment (no parent_id)
func TestInsertCommentTx_RootComment(t *testing.T) {
	ctx := context.Background()

	post := createRandomPost(t) // uses testStore internally

	arg := InsertCommentTxParams{
		UserID: post.UserID,
		PostID: post.ID,
		Body:   util.RandomString(10),
	}

	comment, err := testStore.InsertCommentTx(ctx, arg)
	require.NoError(t, err)
	require.NotZero(t, comment.ID)

	require.Equal(t, arg.UserID, comment.UserID)
	require.Equal(t, arg.PostID, comment.PostID)
	require.Equal(t, arg.Body, comment.Body)

	// root comment properties
	require.EqualValues(t, 0, comment.Depth)
	require.False(t, comment.ParentID.Valid)
	require.EqualValues(t, arg.Upvotes, comment.Upvotes)
	require.EqualValues(t, arg.Downvotes, comment.Downvotes)
}

// Child comment (with parent_id)
func TestInsertCommentTx_ChildComment(t *testing.T) {
	ctx := context.Background()

	// parent comment with depth = 0, no parent_id
	parent := createRandomComment(t)
	parentID := parent.ID

	arg := InsertCommentTxParams{
		UserID:    parent.UserID,
		PostID:    parent.PostID,
		Body:      util.RandomString(10),
		ParentID:  pgtype.Int8{Int64: parentID, Valid: true},
		Upvotes:   5,
		Downvotes: 1,
	}

	comment, err := testStore.InsertCommentTx(ctx, arg)
	require.NoError(t, err)
	require.NotZero(t, comment.ID)

	require.Equal(t, arg.UserID, comment.UserID)
	require.Equal(t, arg.PostID, comment.PostID)
	require.Equal(t, arg.Body, comment.Body)

	// depth = parent.depth + 1
	require.EqualValues(t, parent.Depth+1, comment.Depth)

	// parent_id is set correctly
	require.True(t, comment.ParentID.Valid)
	require.EqualValues(t, parentID, comment.ParentID.Int64)

	require.EqualValues(t, arg.Upvotes, comment.Upvotes)
	require.EqualValues(t, arg.Downvotes, comment.Downvotes)
}

// Non-existent parent: should return OpError with KindNotFound
func TestInsertCommentTx_ParentNotFound(t *testing.T) {
	ctx := context.Background()

	post := createRandomPost(t)
	nonExistingParentID := int64(9_999_999_999)

	arg := InsertCommentTxParams{
		UserID:   post.UserID,
		PostID:   post.ID,
		Body:     util.RandomString(10),
		ParentID: pgtype.Int8{Int64: nonExistingParentID, Valid: true},
	}

	_, err := testStore.InsertCommentTx(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opInsertComment, opErr.Op)
	require.Equal(t, KindNotFound, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)

	// you modelled this as "comment" related to missing "comment"
	require.Equal(t, entComment, opErr.RelatedEntity)
	require.EqualValues(t, nonExistingParentID, opErr.RelatedEntityID)

	// no field-level issue here
	require.Empty(t, opErr.FailingField)
}

// Parent belongs to a different post: KindRelation + failing field "post_id"
func TestInsertCommentTx_ParentPostIDMismatch(t *testing.T) {
	ctx := context.Background()

	// parent on post1
	parent := createRandomComment(t)

	// different post2
	post2 := createRandomPost(t)

	parentID := parent.ID

	arg := InsertCommentTxParams{
		UserID:   parent.UserID,
		PostID:   post2.ID, // intentionally different
		Body:     util.RandomString(10),
		ParentID: pgtype.Int8{Int64: parentID, Valid: true},
	}

	_, err := testStore.InsertCommentTx(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opInsertComment, opErr.Op)
	require.Equal(t, KindRelation, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)

	// we explicitly marked the problematic field
	require.Equal(t, "post_id", opErr.FailingField)

	// you modelled parent as the related comment
	require.Equal(t, entComment, opErr.RelatedEntity)
	require.EqualValues(t, parentID, opErr.RelatedEntityID)
}

// Invalid post_id (FK violation): KindRelation, entity=comment, related_entity=post
func TestInsertCommentTx_InvalidPostID(t *testing.T) {
	ctx := context.Background()

	user := createRandomUser(t)
	invalidPostID := int64(9_999_999_999) // no such post

	arg := InsertCommentTxParams{
		UserID: user.ID,
		PostID: invalidPostID,
		Body:   util.RandomString(10),
	}

	_, err := testStore.InsertCommentTx(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opInsertComment, opErr.Op)
	require.Equal(t, KindRelation, opErr.Kind)

	// you're creating a comment that relates to a missing post
	require.Equal(t, entComment, opErr.Entity)
	require.Equal(t, entPost, opErr.RelatedEntity)
	require.EqualValues(t, invalidPostID, opErr.RelatedEntityID)

	// no specific failing field here
	require.Empty(t, opErr.FailingField)
}

// Deleted parent: KindDeleted on comment, EntityID = parentID
func TestInsertCommentTx_DeletedParent(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)

	// soft-delete the parent comment
	_, err := testStore.softDeleteComment(ctx, comment.ID)
	require.NoError(t, err)

	arg := InsertCommentTxParams{
		UserID:   comment.UserID,
		PostID:   comment.PostID,
		Body:     util.RandomString(10),
		ParentID: pgtype.Int8{Int64: comment.ID, Valid: true},
	}

	_, err = testStore.InsertCommentTx(ctx, arg)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opInsertComment, opErr.Op)
	require.Equal(t, KindDeleted, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)
	require.EqualValues(t, comment.ID, opErr.EntityID)

	// deleted parent itself is the entity; no related entity/field necessary
	require.Empty(t, opErr.RelatedEntity)
	require.Zero(t, opErr.RelatedEntityID)
	require.Empty(t, opErr.FailingField)
}

// Reply at exactly the maximum allowed depth should be rejected: KindConstraint
func TestInsertCommentTx_MaxDepthExceeded(t *testing.T) {
	ctx := context.Background()

	// temporarily set max depth to 3 (allowed depths: 0, 1, 2)
	original := testStore.config.CommentMaxNestingDepth
	testStore.config.CommentMaxNestingDepth = 3
	t.Cleanup(func() { testStore.config.CommentMaxNestingDepth = original })

	post := createRandomPost(t)

	// build a chain: depth 0 -> 1 -> 2
	root, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID: post.UserID,
		PostID: post.ID,
		Body:   "depth-0",
	})
	require.NoError(t, err)
	require.EqualValues(t, 0, root.Depth)

	child1, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:   post.UserID,
		PostID:   post.ID,
		Body:     "depth-1",
		ParentID: pgtype.Int8{Int64: root.ID, Valid: true},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, child1.Depth)

	child2, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:   post.UserID,
		PostID:   post.ID,
		Body:     "depth-2",
		ParentID: pgtype.Int8{Int64: child1.ID, Valid: true},
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, child2.Depth)

	// attempting depth 3 should be rejected
	_, err = testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:   post.UserID,
		PostID:   post.ID,
		Body:     "depth-3-should-fail",
		ParentID: pgtype.Int8{Int64: child2.ID, Valid: true},
	})
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)

	require.Equal(t, opInsertComment, opErr.Op)
	require.Equal(t, KindConstraint, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)
	require.EqualValues(t, child2.ID, opErr.EntityID)
}

// Reply at one level below the max depth should succeed
func TestInsertCommentTx_MaxDepthAllowed(t *testing.T) {
	ctx := context.Background()

	// max depth = 2 means allowed depths are 0 and 1
	original := testStore.config.CommentMaxNestingDepth
	testStore.config.CommentMaxNestingDepth = 2
	t.Cleanup(func() { testStore.config.CommentMaxNestingDepth = original })

	post := createRandomPost(t)

	root, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID: post.UserID,
		PostID: post.ID,
		Body:   "depth-0",
	})
	require.NoError(t, err)
	require.EqualValues(t, 0, root.Depth)

	// depth 1 should be allowed (1 < 2)
	child, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:   post.UserID,
		PostID:   post.ID,
		Body:     "depth-1-ok",
		ParentID: pgtype.Int8{Int64: root.ID, Valid: true},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, child.Depth)

	// depth 2 should be rejected (2 >= 2)
	_, err = testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:   post.UserID,
		PostID:   post.ID,
		Body:     "depth-2-should-fail",
		ParentID: pgtype.Int8{Int64: child.ID, Valid: true},
	})
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, KindConstraint, opErr.Kind)
}
