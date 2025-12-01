package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// Basic happy-path: first upvote.
func TestVoteCommentTx_FirstUpvote(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	user := createRandomUser(t)

	initialUp := comment.Upvotes
	initialDown := comment.Downvotes

	arg := VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	}

	updated, err := testStore.VoteCommentTx(ctx, arg)
	require.NoError(t, err)
	require.Equal(t, comment.ID, updated.ID)

	require.EqualValues(t, initialUp+1, updated.Upvotes)
	require.EqualValues(t, initialDown, updated.Downvotes)
}

// First downvote.
func TestVoteCommentTx_FirstDownvote(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	user := createRandomUser(t)

	initialUp := comment.Upvotes
	initialDown := comment.Downvotes

	arg := VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      -1,
	}

	updated, err := testStore.VoteCommentTx(ctx, arg)
	require.NoError(t, err)
	require.Equal(t, comment.ID, updated.ID)

	require.EqualValues(t, initialUp, updated.Upvotes)
	require.EqualValues(t, initialDown+1, updated.Downvotes)
}

// Change vote from +1 to -1.
func TestVoteCommentTx_ChangeUpToDown(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	user := createRandomUser(t)

	initialUp := comment.Upvotes
	initialDown := comment.Downvotes

	// First upvote
	_, err := testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	})
	require.NoError(t, err)

	// Then change to downvote
	updated, err := testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      -1,
	})
	require.NoError(t, err)

	// up: +1 then -1 -> обратно к initialUp
	require.EqualValues(t, initialUp, updated.Upvotes)
	// down: 0 then +1
	require.EqualValues(t, initialDown+1, updated.Downvotes)
}

// Change vote from -1 to +1.
func TestVoteCommentTx_ChangeDownToUp(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	user := createRandomUser(t)

	initialUp := comment.Upvotes
	initialDown := comment.Downvotes

	// First downvote
	_, err := testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      -1,
	})
	require.NoError(t, err)

	// Then change to upvote
	updated, err := testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	})
	require.NoError(t, err)

	// up: 0 -> +1
	require.EqualValues(t, initialUp+1, updated.Upvotes)
	// down: +1 -> 0
	require.EqualValues(t, initialDown, updated.Downvotes)
}

// Repeated vote should return OpError with KindConflict and not change counters.
func TestVoteCommentTx_DuplicateVote(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	user := createRandomUser(t)

	// First upvote
	first, err := testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	})
	require.NoError(t, err)

	// Second upvote (same value)
	_, err = testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	})
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opVoteComment, opErr.Op)
	require.Equal(t, KindConflict, opErr.Kind)
	require.Equal(t, entCommentVote, opErr.Entity)
	require.Equal(t, "vote", opErr.FailingField)

	// Reload comment and ensure counters unchanged from first result.
	reloaded, err := testStore.GetCommentWithLock(ctx, comment.ID)
	require.NoError(t, err)
	require.EqualValues(t, first.Upvotes, reloaded.Upvotes)
	require.EqualValues(t, first.Downvotes, reloaded.Downvotes)
}

// Invalid vote value should return OpError KindInvalid and not touch counters.
func TestVoteCommentTx_InvalidVoteValue(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	user := createRandomUser(t)

	initialUp := comment.Upvotes
	initialDown := comment.Downvotes

	_, err := testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      0, // invalid
	})
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opVoteComment, opErr.Op)
	require.Equal(t, KindInvalid, opErr.Kind)
	require.Equal(t, entCommentVote, opErr.Entity)
	require.Equal(t, "vote", opErr.FailingField)

	reloaded, err := testStore.GetCommentWithLock(ctx, comment.ID)
	require.NoError(t, err)
	require.EqualValues(t, initialUp, reloaded.Upvotes)
	require.EqualValues(t, initialDown, reloaded.Downvotes)
}

// Invalid user ID (FK violation) -> OpError KindRelation, entity=comment-vote, related=user.
func TestVoteCommentTx_InvalidUserID(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	invalidUserID := int64(9_999_999_999)

	_, err := testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    invalidUserID,
		CommentID: comment.ID,
		Vote:      1,
	})
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opVoteComment, opErr.Op)
	require.Equal(t, KindRelation, opErr.Kind)
	require.Equal(t, entCommentVote, opErr.Entity)
	require.Equal(t, entUser, opErr.RelatedEntity)
	require.EqualValues(t, invalidUserID, opErr.RelatedEntityID)

	// Ensure counters didn't change.
	reloaded, err := testStore.GetCommentWithLock(ctx, comment.ID)
	require.NoError(t, err)
	require.EqualValues(t, comment.Upvotes, reloaded.Upvotes)
	require.EqualValues(t, comment.Downvotes, reloaded.Downvotes)
}

// Invalid comment ID (FK violation) -> OpError KindRelation, entity=comment-vote, related=comment.
func TestVoteCommentTx_InvalidCommentID(t *testing.T) {
	ctx := context.Background()

	user := createRandomUser(t)
	invalidCommentID := int64(9_999_999_999)

	_, err := testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: invalidCommentID,
		Vote:      1,
	})
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opVoteComment, opErr.Op)
	require.Equal(t, KindRelation, opErr.Kind)
	require.Equal(t, entCommentVote, opErr.Entity)
	require.Equal(t, entComment, opErr.RelatedEntity)
	require.EqualValues(t, invalidCommentID, opErr.RelatedEntityID)
}

// Soft-deleted comment cannot be voted -> OpError KindDeleted, entity=comment, EntityID=commentID.
func TestVoteCommentTx_DeletedComment(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	user := createRandomUser(t)

	// Soft delete the comment.
	_, err := testStore.SoftDeleteComment(ctx, comment.ID)
	require.NoError(t, err)

	_, err = testStore.VoteCommentTx(ctx, VoteCommentTxParams{
		UserID:    user.ID,
		CommentID: comment.ID,
		Vote:      1,
	})
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opVoteComment, opErr.Op)
	require.Equal(t, KindDeleted, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)
	require.EqualValues(t, comment.ID, opErr.EntityID)

	// Ensure counters are unchanged after failed vote.
	reloaded, err := testStore.GetCommentWithLock(ctx, comment.ID)
	require.NoError(t, err)
	require.EqualValues(t, comment.Upvotes, reloaded.Upvotes)
	require.EqualValues(t, comment.Downvotes, reloaded.Downvotes)
}

// Concurrent voting by same user on same comment should result in
// exactly one effective vote (thanks to advisory lock and tx logic).
func TestVoteCommentTx_ConcurrentSameUserSameComment(t *testing.T) {
	ctx := context.Background()

	comment := createRandomComment(t)
	user := createRandomUser(t)

	initialUp := comment.Upvotes
	initialDown := comment.Downvotes

	const n = 10
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		go func() {
			_, err := testStore.VoteCommentTx(ctx, VoteCommentTxParams{
				UserID:    user.ID,
				CommentID: comment.ID,
				Vote:      1,
			})
			if !isNilOrConflictVote(err) {
				// We expect either nil or a conflict about the vote.
				errCh <- err
			} else {
				errCh <- nil
			}
		}()
	}

	for i := 0; i < n; i++ {
		err := <-errCh
		require.NoError(t, err)
	}

	reloaded, err := testStore.GetCommentWithLock(ctx, comment.ID)
	require.NoError(t, err)

	// Exactly one upvote should be applied.
	require.EqualValues(t, initialUp+1, reloaded.Upvotes)
	require.EqualValues(t, initialDown, reloaded.Downvotes)
}

// helper for concurrent test: treat KIND conflict on vote as non-fatal.
func isNilOrConflictVote(err error) bool {
	if err == nil {
		return true
	}
	var opErr *OpError
	if errors.As(err, &opErr) {
		return opErr.Op == opVoteComment &&
			opErr.Kind == KindConflict &&
			opErr.Entity == entCommentVote &&
			opErr.FailingField == "vote"
	}
	return false
}
