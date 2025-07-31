-- name: CreateComment :one
SELECT * FROM insert_comment(
  p_user_id := $1,
  p_post_id := $2,
  p_parent_path := sqlc.narg('p_parent_path'),
  p_body := $3
);