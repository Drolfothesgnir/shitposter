-- name: CreateComment :one
SELECT * FROM insert_comment(
  p_user_id := $1,
  p_post_id := $2,
  p_parent_path := sqlc.narg('p_parent_path'),
  p_body := $3,
  p_upvotes := $4,
  p_downvotes := $5
);

-- name: GetComment :one
SELECT * FROM comments
WHERE id = $1 LIMIT 1;

-- name: UpdateComment :one
UPDATE comments
SET 
  upvotes = upvotes + COALESCE(sqlc.narg('delta_upvotes'), 0),
  downvotes = downvotes + COALESCE(sqlc.narg('delta_downvotes'), 0),
  body = COALESCE(sqlc.narg('body'), body),
  last_modified_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetCommentsByPopularity :many
SELECT * FROM get_comments_by_popularity(
  p_post_id := $1,
  p_root_comments_limit := $2
);