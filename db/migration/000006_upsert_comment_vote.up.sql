CREATE OR REPLACE FUNCTION upsert_comment_vote(
	p_user_id BIGINT,
	p_comment_id BIGINT,
	p_vote SMALLINT
) RETURNS TABLE (
	id BIGINT,
	user_id BIGINT,
	comment_id BIGINT,
	vote SMALLINT,
	created_at TIMESTAMPTZ,
	last_modified_at TIMESTAMPTZ,
	delta BOOLEAN,
	inserted_ok BOOLEAN
) AS $$
	WITH orig_vote AS ( -- check if the user has already voted for this comment and the actual vote
		SELECT vote AS val 
		FROM comment_votes
		WHERE 
			user_id = p_user_id AND 
			comment_id = p_comment_id
	),
	new_vote AS ( -- upserting new vote value
		INSERT INTO comment_votes (user_id, comment_id, vote, last_modified_at)
		VALUES (p_user_id, p_comment_id, p_vote, NOW())
		ON CONFLICT (user_id, comment_id) DO UPDATE SET
			vote = EXCLUDED.vote,
			last_modified_at = EXCLUDED.last_modified_at
		RETURNING *
	)
	SELECT
		nv.*,
		(ov.val IS DISTINCT FROM p_vote) AS delta, -- check if the new and the old votes differ
		(ov.val IS NULL) AS inserted_ok -- check if the vote is new and not an update
	FROM new_vote nv
  -- doing LEFT JOIN insetead of INNER one because
  -- if the orig_vote is empty then new_vote JOIN orig_vote will be empty too
	LEFT JOIN orig_vote ov ON TRUE; 
$$ LANGUAGE sql;