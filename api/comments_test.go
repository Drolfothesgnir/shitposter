package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

// helper functions for creating roots and replies
func root(id int64, postID int64) db.CommentsWithAuthor {
	return db.CommentsWithAuthor{
		ID:              id,
		UserID:          100 + id,
		PostID:          postID,
		ParentID:        pgtype.Int8{Int64: 0, Valid: false},
		Depth:           0,
		Body:            "root",
		CreatedAt:       time.Time{},
		UserDisplayName: fmt.Sprintf("user_%d", id),
	}
}

func child(id, parentID int64, depth int32, postID int64) db.CommentsWithAuthor {
	return db.CommentsWithAuthor{
		ID:              id,
		UserID:          100 + id,
		PostID:          postID,
		ParentID:        pgtype.Int8{Int64: parentID, Valid: true},
		Depth:           depth,
		Body:            "child",
		CreatedAt:       time.Time{},
		UserDisplayName: fmt.Sprintf("user_%d", id),
	}
}

func TestPrepareCommentTree_SimpleTree(t *testing.T) {
	// Tree:
	// 1
	// ├── 2
	// │   ├── 3
	// │   └── 4
	// └── 5
	ordered := []db.CommentsWithAuthor{
		root(1, 10),
		child(2, 1, 1, 10),
		child(3, 2, 2, 10),
		child(4, 2, 2, 10),
		child(5, 1, 1, 10),
	}

	nodes, err := PrepareCommentTree(ordered, 1)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	r1 := nodes[0]
	require.Equal(t, int64(1), r1.ID)
	require.Len(t, r1.Replies, 2)

	// 1 -> children [2,5]
	require.Equal(t, int64(2), r1.Replies[0].ID)
	require.Equal(t, int64(5), r1.Replies[1].ID)

	// 2 -> children [3,4]
	require.Len(t, r1.Replies[0].Replies, 2)
	require.Equal(t, int64(3), r1.Replies[0].Replies[0].ID)
	require.Equal(t, int64(4), r1.Replies[0].Replies[1].ID)

	// 5 -> no children
	require.Empty(t, r1.Replies[1].Replies)
}

func TestPrepareCommentTree_MultipleRoots(t *testing.T) {
	// two independent roots, each with its own reply
	// 10
	// └── 11
	// 20
	// ├── 21
	// └── 22
	ordered := []db.CommentsWithAuthor{
		root(10, 77),
		child(11, 10, 1, 77),

		root(20, 77),
		child(21, 20, 1, 77),
		child(22, 20, 1, 77),
	}

	nodes, err := PrepareCommentTree(ordered, 2)
	require.NoError(t, err)
	require.Len(t, nodes, 2)

	// first root
	r1 := nodes[0]
	require.Equal(t, int64(10), r1.ID)
	require.Len(t, r1.Replies, 1)
	require.Equal(t, int64(11), r1.Replies[0].ID)
	require.Empty(t, r1.Replies[0].Replies)

	// second root
	r2 := nodes[1]
	require.Equal(t, int64(20), r2.ID)
	require.Len(t, r2.Replies, 2)
	require.Equal(t, int64(21), r2.Replies[0].ID)
	require.Equal(t, int64(22), r2.Replies[1].ID)
}

func TestPrepareCommentTree_DepthJumpError(t *testing.T) {
	// Error: depth jump from 0 to 2 (parent depth=1 is not seen yet)
	ordered := []db.CommentsWithAuthor{
		root(1, 42),
		// wrong: depth=2, but stack has only root (len(stack)=1)
		{
			ID:        99,
			UserID:    123,
			PostID:    42,
			ParentID:  pgtype.Int8{Int64: 1, Valid: true},
			Depth:     2,
			Body:      "bad jump",
			CreatedAt: time.Time{},
		},
	}

	nodes, err := PrepareCommentTree(ordered, 1)
	require.Nil(t, nodes)
	require.Error(t, err)
	require.Contains(t, err.Error(), "bad depth jump")
}

func TestPrepareCommentTree_SiblingBranchCut(t *testing.T) {
	// Checking correct stack "cut" while moving to the neigbor branch
	// Schema:
	// 1
	// ├── 2
	// │   └── 3
	// └── 4
	ordered := []db.CommentsWithAuthor{
		root(1, 100),
		child(2, 1, 1, 100),
		child(3, 2, 2, 100),
		// moving to the neighbor 4 (depth=1), stack must be cut from depth 2 to 1
		child(4, 1, 1, 100),
	}

	nodes, err := PrepareCommentTree(ordered, 1)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	r1 := nodes[0]
	require.Equal(t, int64(1), r1.ID)
	require.Len(t, r1.Replies, 2)

	require.Equal(t, int64(2), r1.Replies[0].ID)
	require.Len(t, r1.Replies[0].Replies, 1)
	require.Equal(t, int64(3), r1.Replies[0].Replies[0].ID)

	require.Equal(t, int64(4), r1.Replies[1].ID)
	require.Empty(t, r1.Replies[1].Replies)
}

func TestPrepareCommentTree_EmptyInput(t *testing.T) {
	nodes, err := PrepareCommentTree(nil, 0)
	require.NoError(t, err)
	require.Empty(t, nodes)
}
