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
	require.False(t, comment.ParentID.Valid)
	require.Zero(t, comment.Downvotes)
	require.Zero(t, comment.Upvotes)

	return comment
}

func TestCreateComment(t *testing.T) {
	createRandomComment(t)
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
		PUserID:   user.ID,
		PPostID:   post.ID,
		PBody:     util.RandomString(10),
		PParentID: pgtype.Int8{Int64: comment1.ID, Valid: true},
	}

	comment2, err := testStore.CreateComment(context.Background(), arg2)
	require.NoError(t, err)

	require.Equal(t, arg2.PUserID, comment2.UserID)
	require.Equal(t, arg2.PPostID, comment2.PostID)
	require.Equal(t, arg2.PBody, comment2.Body)
	require.Equal(t, int32(1), comment2.Depth)
	require.Equal(t, arg2.PParentID, comment2.ParentID)
	require.Zero(t, comment2.Downvotes)
	require.Zero(t, comment2.Upvotes)
}

func TestGetComment(t *testing.T) {
	post := createRandomPost(t)

	arg := CreateCommentParams{
		PUserID: post.UserID,
		PPostID: post.ID,
		PBody:   util.RandomString(10),
	}

	comment1, err := testStore.CreateComment(context.Background(), arg)
	require.NoError(t, err)

	comment2, err := testStore.GetComment(context.Background(), comment1.ID)
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
	// → #b = order by Popularity             0-#2                     0-#1                    0-#3
	// ↓ a- = Depth                     |      |      |           |     |     |           |     |     |
	//                                 1-#1   1-#2   1-#3        1-#2  1-#3  1-#1        1-#3  1-#2  1-#1

	post := createRandomPost(t)

	roots := make([]Comment, 3)

	root_upvotes := []int64{50, 50, 100}
	root_downvotes := []int64{50, 100, 50}

	for i := range 3 {
		var err error
		roots[i], err = testStore.CreateComment(context.Background(), CreateCommentParams{
			PUserID:    post.UserID,
			PPostID:    post.ID,
			PBody:      fmt.Sprintf("Root comment #%d", i),
			PUpvotes:   pgtype.Int8{Int64: root_upvotes[i], Valid: true},
			PDownvotes: pgtype.Int8{Int64: root_downvotes[i], Valid: true},
		})

		require.NoError(t, err)
	}

	replies := make([][]Comment, 3)

	reply_upvotes := [][]int64{
		{200, 100, 100},
		{100, 100, 200},
		{200, 100, 100},
	}

	reply_downvotes := [][]int64{
		{100, 200, 100},
		{200, 100, 100},
		{100, 100, 200},
	}

	for i := range 3 {
		replies[i] = make([]Comment, 3)
		for j := range 3 {
			var err error
			replies[i][j], err = testStore.CreateComment(context.Background(), CreateCommentParams{
				PUserID:    post.UserID,
				PPostID:    post.ID,
				PBody:      fmt.Sprintf("%d reply to the root comment #%d", j, i),
				PUpvotes:   pgtype.Int8{Int64: reply_upvotes[i][j], Valid: true},
				PDownvotes: pgtype.Int8{Int64: reply_downvotes[i][j], Valid: true},
				PParentID:  pgtype.Int8{Int64: roots[i].ID, Valid: true},
			})

			require.NoError(t, err)
		}
	}

	ordered_comments := []Comment{
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

	query_result, err := testStore.GetCommentsByPopularity(context.Background(), GetCommentsByPopularityParams{
		PPostID:    post.ID,
		PRootLimit: 3,
	})

	require.NoError(t, err)

	require.Equal(t, ordered_comments, query_result)
}

func TestVoteComment(t *testing.T) {
	comment1 := createRandomComment(t)

	user := createRandomUser(t)

	// there should be no vote initially
	vote1, err := testStore.GetCommentVote(context.Background(), GetCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment1.ID,
	})

	require.Empty(t, vote1)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// happy upvote case
	comment2, err := testStore.VoteComment(context.Background(), VoteCommentParams{
		PUserID:    user.ID,
		PCommentID: comment1.ID,
		PVote:      1,
	})

	require.NoError(t, err)
	require.Equal(t, comment1.Upvotes+1, comment2.Upvotes)

	vote2, err := testStore.GetCommentVote(context.Background(), GetCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment1.ID,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), vote2.Vote)

	// vote change to -1
	comment3, err := testStore.VoteComment(context.Background(), VoteCommentParams{
		PUserID:    user.ID,
		PCommentID: comment1.ID,
		PVote:      -1,
	})

	require.NoError(t, err)
	require.Equal(t, comment1.Downvotes+1, comment3.Downvotes)
	require.Equal(t, comment1.Upvotes, comment3.Upvotes)

	vote3, err := testStore.GetCommentVote(context.Background(), GetCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment1.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(-1), vote3.Vote)

	// check voting idempotency
	comment4, err := testStore.VoteComment(context.Background(), VoteCommentParams{
		PUserID:    user.ID,
		PCommentID: comment1.ID,
		PVote:      -1,
	})

	require.NoError(t, err)
	require.Equal(t, comment3.Downvotes, comment4.Downvotes)

	vote4, err := testStore.GetCommentVote(context.Background(), GetCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment1.ID,
	})
	require.NoError(t, err)
	require.Equal(t, vote3.Vote, vote4.Vote)
}

func TestDeleteCommentVote(t *testing.T) {
	comment1 := createRandomComment(t)

	user := createRandomUser(t)

	_, err := testStore.VoteComment(context.Background(), VoteCommentParams{
		PUserID:    user.ID,
		PCommentID: comment1.ID,
		PVote:      1,
	})

	require.NoError(t, err)

	vote1, err := testStore.GetCommentVote(context.Background(), GetCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment1.ID,
	})

	require.NotEmpty(t, vote1)
	require.NoError(t, err)
	require.Equal(t, int64(1), vote1.Vote)

	err = testStore.DeleteCommentVote(context.Background(), DeleteCommentVoteParams{
		PCommentID: comment1.ID,
		PUserID:    user.ID,
	})

	require.NoError(t, err)

	comment2, err := testStore.GetComment(context.Background(), comment1.ID)

	require.NoError(t, err)
	require.Equal(t, comment1.Upvotes, comment2.Upvotes)
	require.Equal(t, comment1.Downvotes, comment2.Downvotes)

	vote2, err := testStore.GetCommentVote(context.Background(), GetCommentVoteParams{
		UserID:    user.ID,
		CommentID: comment1.ID,
	})

	require.Empty(t, vote2)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

}

func TestUpdateComment(t *testing.T) {
	comment1 := createRandomComment(t)

	newBody := util.RandomString(10)

	comment2, err := testStore.UpdateComment(context.Background(), UpdateCommentParams{
		ID:   comment1.ID,
		Body: newBody,
	})

	require.NoError(t, err)

	require.Equal(t, newBody, comment2.Body)
}

func TestGetCommentsByPopularityInvalidPostID(t *testing.T) {
	comments, err := testStore.GetCommentsByPopularity(context.Background(), GetCommentsByPopularityParams{
		PPostID: -1,
	})

	require.NoError(t, err)
	require.True(t, len(comments) == 0)
}

func TestGetCommentsByPopularityInvalidLimit(t *testing.T) {
	_, err := testStore.GetCommentsByPopularity(context.Background(), GetCommentsByPopularityParams{
		PPostID:    1,
		PRootLimit: -1,
	})

	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "2201W", pgErr.Code)

}
