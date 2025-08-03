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

-- name: UpdatePost :one
UPDATE posts
SET 
  title = COALESCE(sqlc.narg('title'), title),
  body = COALESCE(sqlc.narg('body'), body),
  topics = COALESCE(sqlc.narg('topics'), topics),
  upvotes = upvotes + COALESCE(sqlc.narg('delta_upvotes'), 0),
  downvotes = downvotes + COALESCE(sqlc.narg('delta_downvotes'), 0),
  last_modified_at = NOW()
WHERE id = $1
RETURNING *;