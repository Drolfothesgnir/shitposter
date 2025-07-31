-- name: CreatePost :one
INSERT INTO posts (
  user_id, 
  title,
  topics,
  body
) VALUES (
  $1, $2, $3, $4
) RETURNING *;

-- name: GetPost :one
SELECT * FROM posts
WHERE id = $1 LIMIT 1;