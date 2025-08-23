CREATE OR REPLACE FUNCTION insert_comment(
	p_user_id BIGINT,
	p_post_id BIGINT,
	p_parent_path LTREE,
	p_body TEXT,
	p_upvotes BIGINT DEFAULT 0,
	p_downvotes BIGINT DEFAULT 0
) RETURNS comments AS $$
DECLARE
	new_id BIGINT;
	new_path LTREE;
	new_depth INT;
	result comments;
BEGIN
	new_id := NEXTVAL('comments_id_seq');
	IF p_parent_path IS NULL THEN
		new_path := new_id::TEXT::LTREE;
		new_depth := 0;
	ELSE
		new_path := (p_parent_path::TEXT || '.' || new_id::TEXT)::LTREE;
    	new_depth := NLEVEL(p_parent_path);
	END IF;

	INSERT INTO comments (id, user_id, post_id, path, depth, body, upvotes, downvotes)
	VALUES (new_id, p_user_id, p_post_id, new_path, new_depth, p_body, p_upvotes, p_downvotes)
	RETURNING * INTO result;

	RETURN result;
END;

$$ LANGUAGE plpgsql;


-- utility for ordering comments recursively depth-first by popularity
CREATE OR REPLACE FUNCTION get_comments_by_popularity(
	p_post_id BIGINT,
	p_root_comments_limit BIGINT
) RETURNS SETOF comments AS $$
	DECLARE
		max_rank LTREE;
	BEGIN
		max_rank := p_root_comments_limit::TEXT::LTREE;
		RETURN QUERY
		WITH RECURSIVE cte AS (
			SELECT c.*, 
				-- rank used for the end sorting 
				(ROW_NUMBER() OVER(ORDER BY (c.upvotes - c.downvotes) DESC))::TEXT::LTREE AS rank
			FROM comments c
			WHERE c.depth = 0 AND c.post_id = p_post_id
		
			UNION ALL
		
			SELECT c.*, 
				-- concatenate the rank to the parent index to get the child rank
				t.rank || (ROW_NUMBER() OVER(ORDER BY (c.upvotes - c.downvotes) DESC))::TEXT AS rank
				
			FROM comments c, cte t
			-- checks if comment is a descendant of one of the previously found comments
			-- and if there is not too much root comments found
			WHERE 
				c.path <@ t.path AND 
				c.depth = t.depth + 1 AND 
				t.rank <= max_rank
		)
		SELECT 
			c.id, 
			c.user_id, 
			c.post_id, 
			c.path, 
			c.depth, 
			c.upvotes, 
			c.downvotes,
			c.body, 
			c.created_at, 
			c.last_modified_at
		FROM cte c
		WHERE rank <= max_rank
		ORDER BY rank;
	END;

$$ LANGUAGE plpgsql;