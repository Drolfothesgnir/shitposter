CREATE OR REPLACE FUNCTION delete_post_vote(
	p_post_id BIGINT,
	p_user_id BIGINT
) RETURNS void
LANGUAGE sql AS $$
WITH del AS (
	DELETE FROM post_votes
	WHERE user_id = p_user_id AND post_id = p_post_id
	RETURNING vote
)
UPDATE posts p
SET
	upvotes = p.upvotes + (CASE WHEN d.vote = 1 THEN -1 ELSE 0 END),
	downvotes = p.downvotes + (CASE WHEN d.vote = -1 THEN -1 ELSE 0 END)
FROM del d
WHERE p.id = p_post_id;
$$;
