CREATE OR REPLACE FUNCTION delete_comment_vote(
	p_comment_id BIGINT,
	p_user_id BIGINT
) RETURNS void
LANGUAGE sql AS $$
WITH del AS (
	DELETE FROM comment_votes
	WHERE user_id = p_user_id AND comment_id = p_comment_id
	RETURNING vote
)
UPDATE comments c
SET
	upvotes = c.upvotes + (CASE WHEN d.vote = 1 THEN -1 ELSE 0 END),
	downvotes = c.downvotes + (CASE WHEN d.vote = -1 THEN -1 ELSE 0 END)
FROM del d
WHERE c.id = p_comment_id;
$$;