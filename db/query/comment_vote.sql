-- name: CreateCommentVote :one
INSERT INTO comment_votes (
user_id,
comment_id,
vote
) VALUES (
  $1, $2, $3
) RETURNING *;

-- name: GetCommentVoteByID :one 
SELECT * FROM comment_votes
WHERE id = $1 LIMIT 1;

-- name: ChangeCommentVote :one
UPDATE comment_votes
SET 
  vote = $2,
  last_modified_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCommentVote :exec
DELETE FROM comment_votes
WHERE id = $1;

-- name: GetCommentVote :one 
SELECT * from comment_votes
WHERE user_id = $1 AND comment_id = $2
LIMIT 1;