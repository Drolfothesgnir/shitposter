-- name: GetCommentVote :one 
SELECT * from comment_votes
WHERE user_id = $1 AND comment_id = $2
LIMIT 1;