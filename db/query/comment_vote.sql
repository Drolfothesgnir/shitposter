-- name: GetCommentVote :one 
SELECT * from comment_votes
WHERE user_id = $1 AND comment_id = $2
LIMIT 1;

-- name: UpsertCommentVote :one
SELECT
  id::BIGINT AS id ,
	user_id::BIGINT AS user_id ,
	comment_id::BIGINT AS comment_id ,
	vote::SMALLINT AS vote ,
	created_at::TIMESTAMPTZ AS created_at ,
	last_modified_at::TIMESTAMPTZ AS last_modified_at ,
	delta::BOOLEAN AS delta ,
	inserted_ok::BOOLEAN AS inserted_ok 
FROM upsert_comment_vote(
  p_user_id := $1,
  p_comment_id := $2,
  p_vote := $3
);