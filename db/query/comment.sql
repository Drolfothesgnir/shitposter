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

-- name: UpdateComment :one
UPDATE comments
SET 
  upvotes = upvotes + COALESCE(sqlc.narg('delta_upvotes'), 0),
  downvotes = downvotes + COALESCE(sqlc.narg('delta_downvotes'), 0),
  body = COALESCE(sqlc.narg('body'), body)
WHERE id = $1
RETURNING *;
