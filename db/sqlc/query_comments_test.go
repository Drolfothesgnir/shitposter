package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// Small helper struct to hold IDs of our test comments.
type testCommentIDs struct {
	r1   int64
	r2   int64
	r3   int64
	r1a  int64
	r1a1 int64
	r1b  int64
	r2a  int64
}

// Helper: convert int64 to pgtype.Int8 with Valid=true.
func toInt8(v int64) pgtype.Int8 {
	return pgtype.Int8{
		Int64: v,
		Valid: true,
	}
}

// Seed a single post and a comment tree on it.
//
// Tree:
//
//	r1 (pop=10)
//	├─ r1a  (pop=7)
//	│   └─ r1a1 (pop=9)
//	└─ r1b  (pop=3)
//	r2 (pop=5)
//	└─ r2a  (pop=8)
//	r3 (pop=1)
//
// IMPORTANT: because of uniq_reply_per_user_parent (user_id, parent_id),
// we must NOT create two children of the same parent with the same user_id.
// So siblings under the same parent use different users.
func seedCommentTree(t *testing.T) (postID int64, ids testCommentIDs) {
	t.Helper()
	ctx := context.Background()

	// Post author (can be arbitrary for comments)
	post := createRandomPost(t)

	// Separate users for siblings to avoid uniq_reply_per_user_parent conflict.
	userR1 := createRandomUser(t)
	userR2 := createRandomUser(t)
	userR3 := createRandomUser(t)

	userR1ChildA := createRandomUser(t)
	userR1ChildB := createRandomUser(t)
	userR1AChild := createRandomUser(t)
	userR2Child := createRandomUser(t)

	// Root comments (parent_id = NULL)
	// NOTE: insertion order matters for oldest/newest tests.
	cR1, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:  userR1.ID,
		PostID:  post.ID,
		Body:    "r1",
		Upvotes: 10,
	})
	require.NoError(t, err)
	ids.r1 = cR1.ID

	cR2, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:  userR2.ID,
		PostID:  post.ID,
		Body:    "r2",
		Upvotes: 5,
	})
	require.NoError(t, err)
	ids.r2 = cR2.ID

	cR3, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:  userR3.ID,
		PostID:  post.ID,
		Body:    "r3",
		Upvotes: 1,
	})
	require.NoError(t, err)
	ids.r3 = cR3.ID

	// Child of r1: r1a (userR1ChildA)
	cR1A, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:   userR1ChildA.ID,
		PostID:   post.ID,
		Body:     "r1a",
		ParentID: toInt8(ids.r1),
		Upvotes:  7,
	})
	require.NoError(t, err)
	ids.r1a = cR1A.ID

	// Child of r1a: r1a1 (userR1AChild) – different parent_id, so user can be reused freely,
	// but we still just give it its own user for clarity.
	cR1A1, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:   userR1AChild.ID,
		PostID:   post.ID,
		Body:     "r1a1",
		ParentID: toInt8(ids.r1a),
		Upvotes:  9,
	})
	require.NoError(t, err)
	ids.r1a1 = cR1A1.ID

	// Second child of r1: r1b (userR1ChildB)
	// This would violate uniq_reply_per_user_parent if userID was the same
	// as for r1a, so we use a different user.
	cR1B, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:   userR1ChildB.ID,
		PostID:   post.ID,
		Body:     "r1b",
		ParentID: toInt8(ids.r1),
		Upvotes:  3,
	})
	require.NoError(t, err)
	ids.r1b = cR1B.ID

	// Child of r2: r2a (userR2Child)
	cR2A, err := testStore.InsertCommentTx(ctx, InsertCommentTxParams{
		UserID:   userR2Child.ID,
		PostID:   post.ID,
		Body:     "r2a",
		ParentID: toInt8(ids.r2),
		Upvotes:  8,
	})
	require.NoError(t, err)
	ids.r2a = cR2A.ID

	return post.ID, ids
}

// Build a small map[id]index for convenience.
func indexByID(comments []CommentsWithAuthor) map[int64]int {
	idx := make(map[int64]int, len(comments))
	for i, c := range comments {
		idx[c.ID] = i
	}
	return idx
}

// --- TESTS ---

// Invalid order string -> OpError with KindInvalid and failing field "order".
func TestQueryComments_InvalidOrder(t *testing.T) {
	ctx := context.Background()

	res, err := testStore.QueryComments(ctx, CommentQuery{
		Order:  "invalid-order",
		PostID: 123,
		Limit:  10,
		Offset: 0,
	})

	require.Empty(t, res)
	require.Error(t, err)

	var opErr *OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opQueryComments, opErr.Op)
	require.Equal(t, KindInvalid, opErr.Kind)
	require.Equal(t, entComment, opErr.Entity)
	require.Equal(t, "order", opErr.FailingField)
}

// Post without comments -> empty slice, no error.
func TestQueryComments_NoComments(t *testing.T) {
	ctx := context.Background()
	post := createRandomPost(t)

	res, err := testStore.QueryComments(ctx, CommentQuery{
		Order:  CommentOrderPopular,
		PostID: post.ID,
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, res, 0)
}

// Popularity ordering + DFS over the tree.
func TestQueryComments_PopularityDFS(t *testing.T) {
	ctx := context.Background()
	postID, ids := seedCommentTree(t)

	res, err := testStore.QueryComments(ctx, CommentQuery{
		Order:  CommentOrderPopular,
		PostID: postID,
		Limit:  10, // more than enough
		Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, res, 7)

	// Expected DFS order with roots sorted by popularity:
	// roots by popularity: r1(10), r2(5), r3(1)
	// children by popularity:
	//   r1: r1a(7), r1b(3)
	//   r1a: r1a1(9)
	//   r2: r2a(8)
	//
	// DFS:
	//   r1, r1a, r1a1, r1b, r2, r2a, r3
	expectedOrder := []int64{
		ids.r1,
		ids.r1a,
		ids.r1a1,
		ids.r1b,
		ids.r2,
		ids.r2a,
		ids.r3,
	}

	var got []int64
	for _, c := range res {
		got = append(got, c.ID)
	}
	require.Equal(t, expectedOrder, got)

	idx := indexByID(res)

	// DFS: parent must be before children.
	require.Less(t, idx[ids.r1], idx[ids.r1a])
	require.Less(t, idx[ids.r1], idx[ids.r1b])
	require.Less(t, idx[ids.r1a], idx[ids.r1a1])
	require.Less(t, idx[ids.r2], idx[ids.r2a])

	// Roots ordered by popularity: r1 > r2 > r3
	require.Less(t, idx[ids.r1], idx[ids.r2])
	require.Less(t, idx[ids.r2], idx[ids.r3])

	// Children of r1 by popularity: r1a(7) before r1b(3)
	require.Less(t, idx[ids.r1a], idx[ids.r1b])
}

// Oldest ordering (by created_at ASC) + DFS.
func TestQueryComments_OldestDFS(t *testing.T) {
	ctx := context.Background()
	postID, ids := seedCommentTree(t)

	res, err := testStore.QueryComments(ctx, CommentQuery{
		Order:  CommentOrderOldest,
		PostID: postID,
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, res, 7)

	idx := indexByID(res)

	// DFS: parent before children.
	require.Less(t, idx[ids.r1], idx[ids.r1a])
	require.Less(t, idx[ids.r1a], idx[ids.r1a1])
	require.Less(t, idx[ids.r1], idx[ids.r1b])
	require.Less(t, idx[ids.r2], idx[ids.r2a])

	// Roots in insertion/oldest order: r1 (first), r2, r3.
	require.Less(t, idx[ids.r1], idx[ids.r2])
	require.Less(t, idx[ids.r2], idx[ids.r3])

	// Children of r1 in oldest order: r1a created before r1b.
	require.Less(t, idx[ids.r1a], idx[ids.r1b])
}

// Newest ordering (by created_at DESC) + DFS.
func TestQueryComments_NewestDFS(t *testing.T) {
	ctx := context.Background()
	postID, ids := seedCommentTree(t)

	res, err := testStore.QueryComments(ctx, CommentQuery{
		Order:  CommentOrderNewest,
		PostID: postID,
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, res, 7)

	idx := indexByID(res)

	// DFS: parent before children.
	require.Less(t, idx[ids.r1], idx[ids.r1a])
	require.Less(t, idx[ids.r1a], idx[ids.r1a1])
	require.Less(t, idx[ids.r1], idx[ids.r1b])
	require.Less(t, idx[ids.r2], idx[ids.r2a])

	// Roots in newest order: r3 (last inserted) first, then r2, then r1.
	require.Less(t, idx[ids.r3], idx[ids.r2])
	require.Less(t, idx[ids.r2], idx[ids.r1])

	// Children of r1 in newest order:
	// r1b inserted after r1a, so r1b should come before r1a.
	require.Less(t, idx[ids.r1b], idx[ids.r1a])

	// And r1a before its own child r1a1.
	require.Less(t, idx[ids.r1a], idx[ids.r1a1])
}

// Limit/Offset should apply to root comments (with their whole subtrees).
func TestQueryComments_LimitOffsetRoots(t *testing.T) {
	ctx := context.Background()
	postID, ids := seedCommentTree(t)

	// Roots by popularity: r1(10), r2(5), r3(1).
	// root_offset=1, root_limit=1 -> we should get only r2 and its subtree (r2, r2a).
	res, err := testStore.QueryComments(ctx, CommentQuery{
		Order:  CommentOrderPopular,
		PostID: postID,
		Limit:  1,
		Offset: 1,
	})
	require.NoError(t, err)

	var got []int64
	for _, c := range res {
		got = append(got, c.ID)
	}

	require.Len(t, got, 2)
	require.ElementsMatch(t, []int64{ids.r2, ids.r2a}, got)
}
