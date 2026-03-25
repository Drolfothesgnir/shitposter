DROP FUNCTION IF EXISTS delete_comment_leaf(BIGINT, BIGINT, BIGINT);

-- This functions performs hard delete if the target comment has no replies
-- and returns result of this operation including data to proceed if deletion was not successfull
CREATE OR REPLACE FUNCTION delete_comment_leaf(
    p_comment_id BIGINT,
    p_user_id    BIGINT,
    p_post_id    BIGINT
) RETURNS TABLE (
	id           BIGINT,
	user_id      BIGINT,
	post_id      BIGINT,
	is_deleted   BOOLEAN,
	deleted_at   TIMESTAMPTZ,
	has_children BOOLEAN,
	deleted_ok   BOOLEAN
) AS $$
DECLARE
    v_id comments.id%TYPE;
    v_user_id comments.user_id%TYPE;
    v_post_id comments.post_id%TYPE; 
    v_is_deleted comments.is_deleted%TYPE;
    v_deleted_at comments.deleted_at%TYPE;

    v_has_children BOOLEAN;

    v_deleted_row comments%ROWTYPE;
BEGIN
    -- target comment
    SELECT 
        c.id, 
        c.user_id, 
        c.post_id, 
        c.is_deleted, 
        c.deleted_at
    INTO
        v_id,
        v_user_id,
        v_post_id,
        v_is_deleted,
        v_deleted_at
    FROM comments c
    WHERE c.id = p_comment_id
    FOR UPDATE;

    -- to avoid null values in the response in case the target row was not found
    -- the explicit RETURN is used to return no rows and raise ErrNoRows
    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- search for the comment's replies
    -- we only need an indicator if there
    -- are some comments with parent id of the target's
    SELECT EXISTS (
        SELECT 1
		FROM comments ch
		WHERE ch.parent_id = p_comment_id
    ) INTO v_has_children;

    -- actual deletion
    DELETE FROM comments c
    WHERE c.id = v_id
        AND c.user_id = p_user_id -- check if the target comment belongs to the provided user
        AND c.post_id = p_post_id -- check if the target comment belongs to the provided post
        AND c.is_deleted = FALSE -- check if the target comment is not deleted yet
        AND v_has_children = FALSE -- check if the target comment has no replies
    RETURNING c.* INTO v_deleted_row;

    -- if no deletion happenned return the target's data
    -- otherwise return deletion result
	RETURN QUERY SELECT
		COALESCE(v_deleted_row.id, v_id) AS id,
		COALESCE(v_deleted_row.user_id, v_user_id) AS user_id,
		COALESCE(v_deleted_row.post_id, v_post_id) AS post_id,
		COALESCE(v_deleted_row.is_deleted, v_is_deleted) AS is_deleted,
		COALESCE(v_deleted_row.deleted_at, v_deleted_at) AS deleted_at,
		v_has_children AS has_children,
		(v_deleted_row.id IS NOT NULL) AS deleted_ok;
END;
$$ LANGUAGE plpgsql;