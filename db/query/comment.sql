-- name: CreateComment :one
INSERT INTO comments (
  user_id,
  post_id,
  body,
  parent_id,
  depth,
  upvotes,
  downvotes
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetCommentWithLock :one
SELECT * FROM comments
WHERE id = $1 
FOR KEY SHARE
LIMIT 1;

-- name: DeleteCommentIfLeaf :one
DELETE FROM comments c
WHERE c.id = $1
AND NOT EXISTS (
  SELECT 1 FROM comments ch
  WHERE ch.parent_id = c.id
) RETURNING *;

-- name: GetComment :one
SELECT * FROM comments
WHERE id = $1 LIMIT 1;

-- name: UpdateComment :one
SELECT
    id::bigint AS id,
    user_id::bigint AS user_id,
    post_id::bigint AS post_id,
    is_deleted::boolean AS is_deleted,
    body::text AS body,
    last_modified_at::timestamptz AS last_modified_at,
    updated::boolean AS updated
FROM update_comment(
  p_comment_id := $1,
  p_user_id := $2,
  p_post_id := $3,
  p_body := $4
);

-- name: GetCommentsByPopularity :many
SELECT * FROM get_comments_by_popularity(
  p_post_id := $1,
  p_root_limit := $2,
  p_root_offset := $3
);

-- name: GetOldestComments :many
SELECT * FROM get_oldest_comments(
  p_post_id := $1,
  p_root_limit := $2,
  p_root_offset := $3
);

-- name: GetNewestComments :many
SELECT * FROM get_newest_comments(
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

-- name: SoftDeleteComment :one
UPDATE comments
SET 
  body = '[deleted]',
  is_deleted = true,
  deleted_at = NOW(),
  last_modified_at = NOW()
WHERE id = $1
RETURNING *;
