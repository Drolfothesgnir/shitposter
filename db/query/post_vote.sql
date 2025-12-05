-- name: getPostVote :one 
SELECT * from post_votes
WHERE user_id = $1 AND post_id = $2
LIMIT 1;