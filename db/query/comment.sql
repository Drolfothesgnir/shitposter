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
SELECT
  id::BIGINT AS id,
	user_id::BIGINT AS user_id,
	post_id::BIGINT AS post_id,
	is_deleted::BOOLEAN AS is_deleted,
	deleted_at::TIMESTAMPTZ AS deleted_at,
	has_children::BOOLEAN AS has_children,
	deleted_ok::BOOLEAN AS deleted_ok -- True if either soft or hard delete was successful
FROM delete_comment_leaf(
  p_comment_id := $1,
  p_user_id := $2,
  p_post_id := $3
);

-- name: GetComment :one
SELECT * FROM comments
WHERE id = $1 LIMIT 1;

-- name: UpdateComment :one
SELECT
    id::BIGINT AS id,
    user_id::BIGINT AS user_id,
    post_id::BIGINT AS post_id,
    is_deleted::BOOLEAN AS is_deleted,
    body::TEXT AS body,
    last_modified_at::TIMESTAMPTZ AS last_modified_at,
    updated::BOOLEAN AS updated
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

-- name: SoftDeleteComment :one
UPDATE comments
SET 
  body = '[deleted]',
  is_deleted = true,
  deleted_at = NOW(),
  last_modified_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCommentPopularity :one
UPDATE comments
SET
  upvotes = upvotes + sqlc.arg('upvotes_delta')::SMALLINT,
  downvotes = downvotes + sqlc.arg('downvotes_delta')::SMALLINT,
  last_modified_at = NOW()
WHERE id = $1 AND is_deleted = false
RETURNING *;

-- name: getCommentWithAuthor :one
SELECT * FROM comments_with_author
WHERE id = $1;