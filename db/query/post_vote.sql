-- name: CreatePostVote :one
INSERT INTO post_votes (
user_id,
post_id,
vote
) VALUES (
  $1, $2, $3
) RETURNING *;

-- name: GetPostVoteByID :one 
SELECT * FROM post_votes
WHERE id = $1 LIMIT 1;

-- name: ChangePostVote :one
UPDATE post_votes
SET 
  vote = $2,
  last_modified_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePostVote :exec
DELETE FROM post_votes
WHERE id = $1;

-- name: GetPostVote :one 
SELECT * from post_votes
WHERE user_id = $1 AND post_id = $2
LIMIT 1;