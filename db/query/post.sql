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

-- name: GetPostsByPopularity :many
SELECT * FROM posts
WHERE created_at >= (NOW() - sqlc.arg(interval)::INTERVAL)
ORDER BY (upvotes - downvotes) DESC
LIMIT $1
OFFSET $2;

-- name: GetOldestPosts :many
SELECT * FROM posts
ORDER BY created_at ASC
LIMIT $1
OFFSET $2;

-- name: GetNewestPosts :many
SELECT * FROM posts
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: VotePost :one
SELECT * FROM vote_post(
  p_user_id := $1,
  p_post_id := $2,
  p_vote := $3   
);

-- name: DeletePostVote :exec
SELECT delete_post_vote(
  p_post_id := $1,
  p_user_id := $2
);