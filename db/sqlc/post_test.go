package db

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
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
