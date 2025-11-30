package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomComment(t *testing.T) Comment {
	t.Helper()

	ctx := context.Background()
	post := createRandomPost(t)

	arg := CreateCommentParams{
		UserID: post.UserID,
		PostID: post.ID,
		Body:   util.RandomString(10),
	}

	comment, err := testStore.CreateComment(ctx, arg)
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

func TestCreateReplyComment(t *testing.T) {
	ctx := context.Background()

	post := createRandomPost(t)
	user := createRandomUser(t)

	arg1 := InsertCommentTxParams{
		UserID: post.UserID,
		PostID: post.ID,
		Body:   util.RandomString(10),
	}

	comment1, err := testStore.InsertCommentTx(ctx, arg1)
	require.NoError(t, err)

	arg2 := InsertCommentTxParams{
		UserID:   user.ID,
		PostID:   post.ID,
		Body:     util.RandomString(10),
		ParentID: pgtype.Int8{Int64: comment1.ID, Valid: true},
	}

	comment2, err := testStore.InsertCommentTx(ctx, arg2)
	require.NoError(t, err)

	require.Equal(t, arg2.UserID, comment2.UserID)
	require.Equal(t, arg2.PostID, comment2.PostID)
	require.Equal(t, arg2.Body, comment2.Body)
	require.Equal(t, int32(1), comment2.Depth)
	require.Equal(t, arg2.ParentID, comment2.ParentID)
	require.Zero(t, comment2.Downvotes)
	require.Zero(t, comment2.Upvotes)
}

func TestGetComment(t *testing.T) {
	ctx := context.Background()

	post := createRandomPost(t)

	arg := CreateCommentParams{
		UserID: post.UserID,
		PostID: post.ID,
		Body:   util.RandomString(10),
	}

	comment1, err := testStore.CreateComment(ctx, arg)
	require.NoError(t, err)

	comment2, err := testStore.GetComment(ctx, comment1.ID)
	require.NoError(t, err)

	require.Equal(t, comment1.UserID, comment2.UserID)
	require.Equal(t, comment1.PostID, comment2.PostID)
	require.Equal(t, comment1.Body, comment2.Body)
	require.Equal(t, comment1.Depth, comment2.Depth)
	require.Equal(t, comment1.ParentID, comment2.ParentID)
	require.Equal(t, comment1.Downvotes, comment2.Downvotes)
	require.Equal(t, comment1.Upvotes, comment2.Upvotes)
}

func TestGetCommentsByPopularity(t *testing.T) {
	ctx := context.Background()

	// → #b = order by Popularity             0-#2                     0-#1                    0-#3
	// ↓ a- = Depth                     |      |      |           |     |     |           |     |     |
	//                                 1-#1   1-#2   1-#3        1-#2  1-#3  1-#1        1-#3  1-#2  1-#1

	users := make([]User, 12)
	for i := 0; i < 12; i++ {
		users[i] = createRandomUser(t)
	}

	post := createRandomPost(t)

	roots := make([]CommentsWithAuthor, 3)

	rootUpvotes := []int64{50, 50, 100}
	rootDownvotes := []int64{50, 100, 50}

	for i := 0; i < 3; i++ {
		c, err := testStore.CreateComment(ctx, CreateCommentParams{
			UserID:    users[i].ID,
			PostID:    post.ID,
			Body:      fmt.Sprintf("Root comment #%d", i),
			Upvotes:   rootUpvotes[i],
			Downvotes: rootDownvotes[i],
		})
		require.NoError(t, err)

		root := CommentsWithAuthor{
			ID:                c.ID,
			UserID:            c.UserID,
			PostID:            c.PostID,
			ParentID:          c.ParentID,
			Depth:             c.Depth,
			Upvotes:           c.Upvotes,
			Downvotes:         c.Downvotes,
			Body:              c.Body,
			CreatedAt:         c.CreatedAt,
			LastModifiedAt:    c.LastModifiedAt,
			IsDeleted:         c.IsDeleted,
			DeletedAt:         c.DeletedAt,
			Popularity:        c.Popularity,
			UserDisplayName:   users[i].DisplayName,
			UserProfileImgUrl: users[i].ProfileImgUrl,
		}
		roots[i] = root
	}

	replies := make([][]CommentsWithAuthor, 3)

	replyUpvotes := [][]int64{
		{200, 100, 100},
		{100, 100, 200},
		{200, 100, 100},
	}

	replyDownvotes := [][]int64{
		{100, 200, 100},
		{200, 100, 100},
		{100, 100, 200},
	}

	for i := 0; i < 3; i++ {
		replies[i] = make([]CommentsWithAuthor, 3)
		for j := 0; j < 3; j++ {
			userIdx := 3 + i*3 + j
			c, err := testStore.CreateComment(ctx, CreateCommentParams{
				UserID:    users[userIdx].ID,
				PostID:    post.ID,
				Body:      fmt.Sprintf("%d reply to the root comment #%d", j, i),
				Upvotes:   replyUpvotes[i][j],
				Downvotes: replyDownvotes[i][j],
				ParentID:  pgtype.Int8{Int64: roots[i].ID, Valid: true},
			})
			require.NoError(t, err)

			reply := CommentsWithAuthor{
				ID:                c.ID,
				UserID:            c.UserID,
				PostID:            c.PostID,
				ParentID:          c.ParentID,
				Depth:             c.Depth,
				Upvotes:           c.Upvotes,
				Downvotes:         c.Downvotes,
				Body:              c.Body,
				CreatedAt:         c.CreatedAt,
				LastModifiedAt:    c.LastModifiedAt,
				IsDeleted:         c.IsDeleted,
				DeletedAt:         c.DeletedAt,
				Popularity:        c.Popularity,
				UserDisplayName:   users[userIdx].DisplayName,
				UserProfileImgUrl: users[userIdx].ProfileImgUrl,
			}

			replies[i][j] = reply
		}
	}

	orderedComments := []CommentsWithAuthor{
		roots[2],
		replies[2][0],
		replies[2][1],
		replies[2][2],
		roots[0],
		replies[0][0],
		replies[0][2],
		replies[0][1],
		roots[1],
		replies[1][2],
		replies[1][1],
		replies[1][0],
	}

	queryResult, err := testStore.GetCommentsByPopularity(ctx, GetCommentsByPopularityParams{
		PPostID:    post.ID,
		PRootLimit: 3,
	})
	require.NoError(t, err)
	require.Equal(t, orderedComments, queryResult)
}

// Note: VoteCommentTx now has its own dedicated tests that assert on OpError,
// so the old all-in-one TestVoteCommentTx has been removed here.

func TestUpdateComment_Success(t *testing.T) {
	ctx := context.Background()

	comment1 := createRandomComment(t)
	newBody := util.RandomString(10)

	result, err := testStore.UpdateComment(ctx, UpdateCommentParams{
		PCommentID: comment1.ID,
		PUserID:    comment1.UserID,
		PPostID:    comment1.PostID,
		PBody:      newBody,
	})
	require.NoError(t, err)
	require.True(t, result.Updated)
	require.False(t, result.IsDeleted)
	require.Equal(t, newBody, result.Body)
	require.Equal(t, comment1.ID, result.ID)
	require.Equal(t, comment1.UserID, result.UserID)
	require.Equal(t, comment1.PostID, result.PostID)

	// double-check in DB
	comment2, err := testStore.GetComment(ctx, comment1.ID)
	require.NoError(t, err)
	require.Equal(t, newBody, comment2.Body)
}

func TestUpdateComment_NonExistingComment(t *testing.T) {
	ctx := context.Background()

	invalidID := int64(-1)

	_, err := testStore.UpdateComment(ctx, UpdateCommentParams{
		PCommentID: invalidID,
		PUserID:    1,
		PPostID:    1,
		PBody:      "whatever",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestUpdateComment_DeletedComment(t *testing.T) {
	ctx := context.Background()

	comment1 := createRandomComment(t)

	// soft-delete the comment first
	deleted, err := testStore.SoftDeleteComment(ctx, comment1.ID)
	require.NoError(t, err)
	require.True(t, deleted.IsDeleted)

	newBody := util.RandomString(10)

	result, err := testStore.UpdateComment(ctx, UpdateCommentParams{
		PCommentID: comment1.ID,
		PUserID:    comment1.UserID,
		PPostID:    comment1.PostID,
		PBody:      newBody,
	})
	require.NoError(t, err)

	// update must NOT happen
	require.False(t, result.Updated)
	require.True(t, result.IsDeleted)
	require.Equal(t, "[deleted]", result.Body)
	require.Equal(t, comment1.ID, result.ID)
	require.Equal(t, comment1.UserID, result.UserID)
	require.Equal(t, comment1.PostID, result.PostID)

	// DB state must stay "[deleted]"
	comment2, err := testStore.GetComment(ctx, comment1.ID)
	require.NoError(t, err)
	require.True(t, comment2.IsDeleted)
	require.Equal(t, "[deleted]", comment2.Body)
}

func TestUpdateComment_WrongUser(t *testing.T) {
	ctx := context.Background()

	comment1 := createRandomComment(t)
	otherUser := createRandomUser(t)
	newBody := util.RandomString(10)

	result, err := testStore.UpdateComment(ctx, UpdateCommentParams{
		PCommentID: comment1.ID,
		PUserID:    otherUser.ID, // NOT the author
		PPostID:    comment1.PostID,
		PBody:      newBody,
	})
	require.NoError(t, err)

	// update must NOT happen
	require.False(t, result.Updated)
	require.False(t, result.IsDeleted)
	require.Equal(t, comment1.ID, result.ID)
	require.Equal(t, comment1.UserID, result.UserID)
	require.Equal(t, comment1.PostID, result.PostID)
	require.Equal(t, comment1.Body, result.Body)

	// DB must still have original body
	comment2, err := testStore.GetComment(ctx, comment1.ID)
	require.NoError(t, err)
	require.Equal(t, comment1.Body, comment2.Body)
}

func TestUpdateComment_WrongPost(t *testing.T) {
	ctx := context.Background()

	comment1 := createRandomComment(t)
	otherPost := createRandomPost(t)
	newBody := util.RandomString(10)

	result, err := testStore.UpdateComment(ctx, UpdateCommentParams{
		PCommentID: comment1.ID,
		PUserID:    comment1.UserID,
		PPostID:    otherPost.ID, // wrong post
		PBody:      newBody,
	})
	require.NoError(t, err)

	// update must NOT happen
	require.False(t, result.Updated)
	require.False(t, result.IsDeleted)
	require.Equal(t, comment1.ID, result.ID)
	require.Equal(t, comment1.UserID, result.UserID)
	require.Equal(t, comment1.PostID, result.PostID)
	require.Equal(t, comment1.Body, result.Body)

	// DB must still have original body
	comment2, err := testStore.GetComment(ctx, comment1.ID)
	require.NoError(t, err)
	require.Equal(t, comment1.Body, comment2.Body)
}

func TestGetCommentsByPopularityInvalidPostID(t *testing.T) {
	ctx := context.Background()

	comments, err := testStore.GetCommentsByPopularity(ctx, GetCommentsByPopularityParams{
		PPostID: -1,
	})
	require.NoError(t, err)
	require.Len(t, comments, 0)
}

func TestGetCommentsByPopularityInvalidLimit(t *testing.T) {
	ctx := context.Background()

	_, err := testStore.GetCommentsByPopularity(ctx, GetCommentsByPopularityParams{
		PPostID:    1,
		PRootLimit: -1,
	})
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "2201W", pgErr.Code)
}

func TestDeleteComment(t *testing.T) {
	ctx := context.Background()

	comment1 := createRandomComment(t)

	comment2, err := testStore.SoftDeleteComment(ctx, comment1.ID)
	require.NoError(t, err)

	require.True(t, comment2.IsDeleted)
	require.Equal(t, "[deleted]", comment2.Body)
	require.True(t, comment2.DeletedAt.After(comment2.CreatedAt))
	require.True(t, comment2.LastModifiedAt.After(comment2.CreatedAt))
}
