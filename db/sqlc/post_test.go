package db

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomPost(t *testing.T) Post {

	user := createRandomUser(t)

	title := util.RandomString(10)

	topics1 := []string{util.RandomString(6), util.RandomString(6)}

	json_topics, err := json.Marshal(topics1)
	require.NoError(t, err)

	body1 := PostBody{ContentType: "html", Content: fmt.Sprintf("<h1>%s</h1>", title)}

	json_body, err := json.Marshal(body1)
	require.NoError(t, err)

	arg := CreatePostParams{
		UserID: user.ID,
		Title:  title,
		Topics: json_topics,
		Body:   json_body,
	}

	post, err := testStore.CreatePost(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, post)

	require.Equal(t, arg.Title, post.Title)
	require.Equal(t, arg.UserID, user.ID)

	var body2 PostBody

	err = json.Unmarshal(post.Body, &body2)
	require.NoError(t, err)

	require.Equal(t, body1.ContentType, body2.ContentType)
	require.Equal(t, body1.Content, body2.Content)

	var topics2 []string

	err = json.Unmarshal(post.Topics, &topics2)
	require.NoError(t, err)
	require.Equal(t, topics1, topics2)

	require.NotZero(t, post.ID)

	require.Zero(t, post.Downvotes)
	require.Zero(t, post.Upvotes)

	require.NotZero(t, user.CreatedAt)

	return post
}

func TestCreatePost(t *testing.T) {
	createRandomPost(t)
}

func TestGetNewestPosts(t *testing.T) {
	n := 10
	posts := make([]Post, n)

	user := createRandomUser(t)

	var err error

	body := PostBody{ContentType: "html", Content: "<h1>Hello world</h1>"}
	json_body, err := json.Marshal(body)
	require.NoError(t, err)

	topics := []string{util.RandomString(3)}
	json_topics, err := json.Marshal(topics)
	require.NoError(t, err)

	for i := range n {
		// inserting posts into array in reverse order
		posts[n-i-1], err = testStore.CreatePost(context.Background(), CreatePostParams{
			UserID: user.ID,
			Title:  util.RandomString(10),
			Body:   json_body,
			Topics: json_topics,
		})
		require.NoError(t, err)
	}

	// get all n new posts first
	query_result1, err := testStore.GetNewestPosts(context.Background(), GetNewestPostsParams{
		Limit:  int32(n),
		Offset: 0,
	})

	require.NoError(t, err)
	require.Equal(t, posts, query_result1)

	// get first 5 posts
	query_result2, err := testStore.GetNewestPosts(context.Background(), GetNewestPostsParams{
		Limit:  int32(5),
		Offset: 0,
	})

	require.NoError(t, err)
	require.Equal(t, posts[:5], query_result2)

	// get second 5 posts
	query_result3, err := testStore.GetNewestPosts(context.Background(), GetNewestPostsParams{
		Limit:  int32(5),
		Offset: 5,
	})

	require.NoError(t, err)
	require.Equal(t, posts[5:], query_result3)
}

func TestVotePost(t *testing.T) {
	post1 := createRandomPost(t)

	user := createRandomUser(t)

	// there should be no vote initially
	vote1, err := testStore.GetPostVote(context.Background(), GetPostVoteParams{
		UserID: user.ID,
		PostID: post1.ID,
	})

	require.Empty(t, vote1)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// happy upvote case
	post2, err := testStore.VotePost(context.Background(), VotePostParams{
		PUserID: user.ID,
		PPostID: post1.ID,
		PVote:   1,
	})

	require.NoError(t, err)
	require.Equal(t, post1.Upvotes+1, post2.Upvotes)

	vote2, err := testStore.GetPostVote(context.Background(), GetPostVoteParams{
		UserID: user.ID,
		PostID: post1.ID,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), vote2.Vote)

	// vote change to -1
	post3, err := testStore.VotePost(context.Background(), VotePostParams{
		PUserID: user.ID,
		PPostID: post1.ID,
		PVote:   -1,
	})

	require.NoError(t, err)
	require.Equal(t, post1.Downvotes+1, post3.Downvotes)
	require.Equal(t, post1.Upvotes, post3.Upvotes)

	vote3, err := testStore.GetPostVote(context.Background(), GetPostVoteParams{
		UserID: user.ID,
		PostID: post1.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(-1), vote3.Vote)

	// check voting idempotency
	post4, err := testStore.VotePost(context.Background(), VotePostParams{
		PUserID: user.ID,
		PPostID: post1.ID,
		PVote:   -1,
	})

	require.NoError(t, err)
	require.Equal(t, post3.Downvotes, post4.Downvotes)

	vote4, err := testStore.GetPostVote(context.Background(), GetPostVoteParams{
		UserID: user.ID,
		PostID: post1.ID,
	})
	require.NoError(t, err)
	require.Equal(t, vote3.Vote, vote4.Vote)
}

func TestUpdatePost(t *testing.T) {
	post1 := createRandomPost(t)

	newTitle := util.RandomString(10)

	body1, err := GetPostBodyFromJSON(post1.Body)
	require.NoError(t, err)

	newBody := PostBody{
		ContentType: body1.ContentType,
		Content:     "<h2>Test</h2>",
	}

	newBodyJson, err := json.Marshal(newBody)
	require.NoError(t, err)

	topics1, err := GetPostTopicsFromJSON(post1.Topics)
	require.NoError(t, err)

	newTopic := util.RandomString(5)

	newTopics := append(topics1, newTopic)
	newTopicsJson, err := json.Marshal(newTopics)
	require.NoError(t, err)

	post2, err := testStore.UpdatePost(context.Background(), UpdatePostParams{
		ID:     post1.ID,
		Title:  pgtype.Text{String: newTitle, Valid: true},
		Body:   newBodyJson,
		Topics: newTopicsJson,
	})
	require.NoError(t, err)

	require.Equal(t, post1.ID, post2.ID)
	require.True(t, post2.LastModifiedAt.After(post1.LastModifiedAt))

	body2, err := GetPostBodyFromJSON(post2.Body)
	require.NoError(t, err)

	require.Equal(t, &newBody, body2)

	topics2, err := GetPostTopicsFromJSON(post2.Topics)
	require.NoError(t, err)
	require.Equal(t, newTopics, topics2)

	require.Equal(t, newTitle, post2.Title)
}

func TestDeletePostVote(t *testing.T) {
	post1 := createRandomPost(t)

	user := createRandomUser(t)

	_, err := testStore.VotePost(context.Background(), VotePostParams{
		PUserID: user.ID,
		PPostID: post1.ID,
		PVote:   1,
	})

	require.NoError(t, err)

	vote1, err := testStore.GetPostVote(context.Background(), GetPostVoteParams{
		UserID: user.ID,
		PostID: post1.ID,
	})

	require.NotEmpty(t, vote1)
	require.NoError(t, err)
	require.Equal(t, int64(1), vote1.Vote)

	err = testStore.DeletePostVote(context.Background(), DeletePostVoteParams{
		PPostID: post1.ID,
		PUserID: user.ID,
	})

	require.NoError(t, err)

	post2, err := testStore.GetPost(context.Background(), post1.ID)

	require.NoError(t, err)
	require.Equal(t, post1.Upvotes, post2.Upvotes)
	require.Equal(t, post1.Downvotes, post2.Downvotes)

	vote2, err := testStore.GetPostVote(context.Background(), GetPostVoteParams{
		UserID: user.ID,
		PostID: post1.ID,
	})

	require.Empty(t, vote2)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

}

func TestDeletePost(t *testing.T) {
	user := createRandomUser(t)
	post1 := createRandomPost(t)

	comment1, err := testStore.CreateComment(context.Background(), CreateCommentParams{
		PUserID: user.ID,
		PPostID: post1.ID,
		PBody:   util.RandomString(6),
	})

	require.NoError(t, err)

	comment2, err := testStore.CreateComment(context.Background(), CreateCommentParams{
		PUserID: user.ID,
		PPostID: post1.ID,
		PBody:   util.RandomString(6),
	})

	require.NoError(t, err)

	err = testStore.DeletePost(context.Background(), post1.ID)
	require.NoError(t, err)

	_, err = testStore.GetPost(context.Background(), post1.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	_, err = testStore.GetComment(context.Background(), comment1.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	_, err = testStore.GetComment(context.Background(), comment2.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
