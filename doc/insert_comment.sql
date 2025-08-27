CREATE OR REPLACE FUNCTION insert_comment(
	p_user_id BIGINT,
	p_post_id BIGINT,
	p_parent_id BIGINT,
	p_body TEXT,
	p_upvotes BIGINT DEFAULT 0,
	p_downvotes BIGINT DEFAULT 0
) RETURNS comments
-- using plpgsql because I have variables and control flow
LANGUAGE plpgsql AS $$
DECLARE
	v_depth INT;
	v_post BIGINT;
  v_upvotes BIGINT := COALESCE(p_upvotes, 0);
  v_downvotes BIGINT := COALESCE(p_downvotes, 0); 
	row_out comments;
BEGIN
	IF p_parent_id IS NULL THEN
		v_depth := 0;
	ELSE
		SELECT post_id, depth INTO v_post, v_depth
		FROM comments
		WHERE id = p_parent_id
    -- disabling other transactions from deleting the parent comment
		FOR KEY SHARE;

		IF NOT FOUND THEN
			RAISE EXCEPTION 'Parent % not found', p_parent_id
			USING ERRCODE = 'foreign_key_violation';
		END IF;

		IF v_post <> p_post_id THEN
			RAISE EXCEPTION 'Parent(%) belongs to post(%) but, new comment has post(%)',
			p_parent_id, v_post, p_post_id;
		END IF;

		v_depth := v_depth + 1;
	END IF;

	INSERT INTO comments (user_id, post_id, parent_id, depth, body, upvotes, downvotes)
	VALUES (p_user_id, p_post_id, p_parent_id, v_depth, p_body, v_upvotes, v_downvotes)
	RETURNING * INTO row_out;

	RETURN row_out;
END;
$$;
