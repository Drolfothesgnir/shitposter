-- name: CreateComment :one
SELECT * FROM insert_comment(
  p_user_id := $1,
  p_post_id := $2,
  p_parent_path := sqlc.narg('p_parent_path'),
  p_body := $3
);

-- name: GetComment :one
SELECT * FROM comments
WHERE id = $1 LIMIT 1;

-- name: UpvoteComment :one
UPDATE comments
SET upvotes = upvotes + 1
WHERE id = $1
RETURNING *;

-- name: DownvoteComment :one
UPDATE comments
SET downvotes = downvotes + 1
WHERE id = $1
RETURNING *;
