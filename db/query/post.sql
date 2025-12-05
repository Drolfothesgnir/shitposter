-- name: createPost :one
INSERT INTO posts (
  user_id, 
  title,
  topics,
  body
) VALUES (
  $1, $2, $3, $4
) RETURNING *;

-- name: getPostWithAuthor :one
SELECT * FROM posts_with_author
WHERE id = $1
LIMIT 1;

-- name: getPost :one
SELECT * FROM posts
WHERE id = $1
LIMIT 1;

-- name: updatePost :one
UPDATE posts
SET 
  title = COALESCE(sqlc.narg('title'), title),
  body = COALESCE(sqlc.narg('body'), body),
  topics = COALESCE(sqlc.narg('topics'), topics),
  last_modified_at = NOW()
WHERE id = $1
RETURNING *;

-- name: getPostsByPopularity :many
SELECT * FROM posts_with_author
WHERE p.created_at >= (NOW() - sqlc.arg(interval)::INTERVAL)
ORDER BY (p.upvotes - p.downvotes) DESC
LIMIT $1
OFFSET $2;

-- name: getOldestPosts :many
SELECT * FROM posts_with_author
ORDER BY created_at ASC, id ASC
LIMIT $1 OFFSET $2;

-- name: getNewestPosts :many
SELECT * FROM posts_with_author
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: votePost :one
SELECT * FROM vote_post(
  p_user_id := $1,
  p_post_id := $2,
  p_vote := $3   
);

-- name: deletePostVote :exec
SELECT delete_post_vote(
  p_post_id := $1,
  p_user_id := $2
);

-- name: deletePost :exec
DELETE FROM posts
WHERE id = $1;