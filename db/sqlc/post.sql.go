// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: post.sql

package db

import (
	"context"
)

const createPost = `-- name: CreatePost :one
INSERT INTO posts (
  user_id, 
  title,
  topics,
  body
) VALUES (
  $1, $2, $3, $4
) RETURNING id, user_id, title, topics, body, upvotes, downvotes, created_at, last_modified_at
`

type CreatePostParams struct {
	UserID int64  `json:"user_id"`
	Title  string `json:"title"`
	Topics []byte `json:"topics"`
	Body   []byte `json:"body"`
}

func (q *Queries) CreatePost(ctx context.Context, arg CreatePostParams) (Post, error) {
	row := q.db.QueryRow(ctx, createPost,
		arg.UserID,
		arg.Title,
		arg.Topics,
		arg.Body,
	)
	var i Post
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Title,
		&i.Topics,
		&i.Body,
		&i.Upvotes,
		&i.Downvotes,
		&i.CreatedAt,
		&i.LastModifiedAt,
	)
	return i, err
}

const getPost = `-- name: GetPost :one
SELECT id, user_id, title, topics, body, upvotes, downvotes, created_at, last_modified_at FROM posts
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetPost(ctx context.Context, id int64) (Post, error) {
	row := q.db.QueryRow(ctx, getPost, id)
	var i Post
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Title,
		&i.Topics,
		&i.Body,
		&i.Upvotes,
		&i.Downvotes,
		&i.CreatedAt,
		&i.LastModifiedAt,
	)
	return i, err
}
