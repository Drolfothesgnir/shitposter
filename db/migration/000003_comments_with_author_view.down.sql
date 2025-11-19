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

DROP INDEX IF EXISTS comments_children_newest;
DROP INDEX IF EXISTS comments_roots_newest;
DROP INDEX IF EXISTS comments_children_oldest;
DROP INDEX IF EXISTS comments_roots_oldest;

DROP FUNCTION IF EXISTS get_newest_comments;
DROP FUNCTION IF EXISTS get_oldest_comments;
DROP FUNCTION IF EXISTS get_comments_by_popularity;

CREATE OR REPLACE FUNCTION get_comments_by_popularity(
  p_post_id BIGINT,
  p_root_limit INT,
  p_root_offset INT
) RETURNS SETOF comments
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  roots AS (
    SELECT c.*
    FROM comments c
    WHERE c.post_id = p_post_id AND c.parent_id IS NULL
    ORDER BY c.popularity DESC, c.id
    LIMIT p_root_limit
	  OFFSET p_root_offset
  ),
  cte (
    id, user_id, post_id, parent_id, depth,
    upvotes, downvotes, body, created_at, last_modified_at,
    is_deleted, deleted_at, popularity, rnk
  ) AS (
    SELECT
      r.id, r.user_id, r.post_id, r.parent_id, r.depth,
      r.upvotes, r.downvotes, r.body, r.created_at, r.last_modified_at,
      r.is_deleted, r.deleted_at, r.popularity,
      ARRAY[ROW_NUMBER() OVER (ORDER BY r.popularity DESC, r.id)]::BIGINT[] AS rnk
    FROM roots r

    UNION ALL

    SELECT
      ch.id, ch.user_id, ch.post_id, ch.parent_id, ch.depth,
      ch.upvotes, ch.downvotes, ch.body, ch.created_at, ch.last_modified_at,
      ch.is_deleted, ch.deleted_at, ch.popularity,
      t.rnk || ch.rn AS rnk
    FROM cte t
    JOIN LATERAL (
      SELECT c.*,
             ROW_NUMBER() OVER (ORDER BY c.popularity DESC, c.id) AS rn
      FROM comments c
      WHERE c.post_id = t.post_id
        AND c.parent_id = t.id
    ) ch ON TRUE
  )
  SELECT
    id, user_id, post_id, parent_id, depth,
    upvotes, downvotes, body, created_at, last_modified_at,
    is_deleted, deleted_at, popularity
  FROM cte
  ORDER BY rnk;
$$;

DROP VIEW IF EXISTS comments_with_author;