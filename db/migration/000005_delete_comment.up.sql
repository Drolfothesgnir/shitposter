-- This functions performs hard delete if the target comment has no replies
-- and returns result of this operation including data to proceed if deletion was not successfull
CREATE OR REPLACE FUNCTION delete_comment_leaf(
	p_comment_id BIGINT,
  p_user_id BIGINT,
  p_post_id BIGINT
) RETURNS TABLE (
	id BIGINT,
	user_id BIGINT,
	post_id BIGINT,
	is_deleted BOOLEAN,
	deleted_at TIMESTAMPTZ,
	has_children BOOLEAN,
	deleted_ok BOOLEAN
) AS $$
  -- target comment
	WITH target AS (
		SELECT 
			id, 
			user_id, 
			post_id, 
			is_deleted, 
			deleted_at
		FROM comments
		WHERE id = p_comment_id
	),
  -- search for the comment's replies
  -- we only need an indicator if there
  -- are some comments with parent id of the target's
	children AS (
		SELECT 1
		FROM comments ch
		WHERE ch.parent_id = p_comment_id
		LIMIT 1
	),
  -- actual deletion
	deleted_rows AS (
		DELETE FROM comments c
		USING target t
		WHERE c.id = t.id
			AND c.user_id = p_user_id -- check if the target comment belongs to the provided user
			AND c.post_id = p_post_id -- check if the target comment belongs to the provided post
			AND c.is_deleted = false -- check if the target comment is not deleted yet
			AND NOT EXISTS (SELECT 1 FROM children) -- check if the target comment has no replies
		RETURNING c.*
	)
  -- if no deletion happenned return the target's data
  -- otherwise return deletion result
	SELECT
		COALESCE(d.id, t.id) AS id,
		COALESCE(d.user_id, t.user_id) AS user_id,
		COALESCE(d.post_id, t.post_id) AS post_id,
		COALESCE(d.is_deleted, t.is_deleted) AS is_deleted,
		COALESCE(d.deleted_at, t.deleted_at) AS deleted_at,
		EXISTS (SELECT 1 FROM children) AS has_children,
		(d.id IS NOT NULL) AS deleted_ok
	FROM target t
	LEFT JOIN deleted_rows d ON t.id = d.id;
$$ LANGUAGE sql;