-- name: CreateComment :one
SELECT * FROM insert_comment(
  p_user_id := $1,
  p_post_id := $2,
  p_body := $3,
  p_parent_id := sqlc.narg('p_parent_id'),
  p_upvotes := sqlc.narg('p_upvotes'),
  p_downvotes := sqlc.narg('p_downvotes')
);

-- name: GetComment :one
SELECT * FROM comments
WHERE id = $1 LIMIT 1;

-- name: UpdateComment :one
UPDATE comments
SET
  body = $2,
  last_modified_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetCommentsByPopularity :many
SELECT * FROM get_comments_by_popularity(
  p_post_id := $1,
  p_root_limit := $2,
  p_root_offset := $3
);

-- name: VoteComment :one
SELECT * FROM vote_comment(
  p_user_id := $1,
  p_comment_id := $2,
  p_vote := $3   
);

-- name: DeleteCommentVote :exec
SELECT delete_comment_vote(
  p_comment_id := $1,
  p_user_id := $2
);

-- name: DeleteComment :one
UPDATE comments
SET 
  body = '[deleted]',
  is_deleted = true,
  deleted_at = NOW(),
  last_modified_at = NOW()
WHERE id = $1
RETURNING *;
