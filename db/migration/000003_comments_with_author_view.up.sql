CREATE OR REPLACE VIEW comments_with_author AS
SELECT 
  c.*,
  u.display_name      AS user_display_name,
  u.profile_img_url   AS user_profile_img_url
FROM comments AS c
JOIN users AS u ON u.id = c.user_id;

DROP FUNCTION IF EXISTS get_comments_by_popularity;

-- utility for extracting comments ordered by popularity
CREATE OR REPLACE FUNCTION get_comments_by_popularity(
  p_post_id BIGINT,
  p_root_limit INT,
  p_root_offset INT
) RETURNS SETOF comments_with_author
-- STABLE is used for optimization. It tells to the engine that db will not be modified, only queried
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  -- getting root comments
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
	  -- creating array of order indexes for the final sort
	  -- it gives every comment its place in ordered by popularity list
      ARRAY[ROW_NUMBER() OVER (ORDER BY r.popularity DESC, r.id)]::BIGINT[] AS rnk
    FROM roots r

    UNION ALL

    -- getting children of the root comments
    SELECT
      ch.id, ch.user_id, ch.post_id, ch.parent_id, ch.depth,
      ch.upvotes, ch.downvotes, ch.body, ch.created_at, ch.last_modified_at,
      ch.is_deleted, ch.deleted_at, ch.popularity, t.rnk || ch.rn AS rnk
    FROM cte t
	-- using JOIN LATERAL because the condition needs data from multiple sources
    JOIN LATERAL (
      SELECT c.*,
			-- index in ordered by popularity list, same thing as for the root comments
             ROW_NUMBER() OVER (ORDER BY c.popularity DESC, c.id) AS rn
      FROM comments c
      WHERE c.post_id = t.post_id
        AND c.parent_id = t.id
    ) ch ON TRUE
  )
  SELECT
    c.id, c.user_id, c.post_id, c.parent_id, c.depth,
    c.upvotes, c.downvotes, c.body, c.created_at, c.last_modified_at,
    c.is_deleted, c.deleted_at, c.popularity,
    u.display_name    AS user_display_name,
    u.profile_img_url AS user_profile_img_url
  FROM cte c
  -- i'm joining comments with users at the end because it's faster than
  -- extracting row from the view on every iteration
  JOIN users u ON u.id = c.user_id
  -- utilising ordered index array
  ORDER BY rnk;
$$;

-- utility for extracting oldest comments 
CREATE OR REPLACE FUNCTION get_oldest_comments(
  p_post_id BIGINT,
  p_root_limit INT,
  p_root_offset INT
) RETURNS SETOF comments_with_author
-- STABLE is used for optimization. It tells to the engine that db will not be modified, only queried
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  -- getting root comments
  roots AS (
    SELECT c.*
    FROM comments c
    WHERE c.post_id = p_post_id AND c.parent_id IS NULL
    ORDER BY c.created_at ASC, c.id
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
	  -- creating array of order indexes for the final sort
	  -- it gives every comment its place in ordered by creation date list
      ARRAY[ROW_NUMBER() OVER (ORDER BY r.created_at ASC, r.id)]::BIGINT[] AS rnk
    FROM roots r

    UNION ALL

    -- getting children of the root comments
    SELECT
      ch.id, ch.user_id, ch.post_id, ch.parent_id, ch.depth,
      ch.upvotes, ch.downvotes, ch.body, ch.created_at, ch.last_modified_at,
      ch.is_deleted, ch.deleted_at, ch.popularity, t.rnk || ch.rn AS rnk
    FROM cte t
	-- using JOIN LATERAL because the condition needs data from multiple sources
    JOIN LATERAL (
      SELECT c.*,
			-- index in ordered by creation date list, same thing as for the root comments
             ROW_NUMBER() OVER (ORDER BY c.created_at ASC, c.id) AS rn
      FROM comments c
      WHERE c.post_id = t.post_id
        AND c.parent_id = t.id
    ) ch ON TRUE
  )
  SELECT
    c.id, c.user_id, c.post_id, c.parent_id, c.depth,
  	c.upvotes, c.downvotes, c.body, c.created_at, c.last_modified_at,
  	c.is_deleted, c.deleted_at, c.popularity,
  	u.display_name    AS user_display_name,
  	u.profile_img_url AS user_profile_img_url
  FROM cte c
  JOIN users u ON u.id = c.user_id
  -- utilising ordered index array
  ORDER BY rnk;
$$;

-- utility for extracting newest comments 
CREATE OR REPLACE FUNCTION get_newest_comments(
  p_post_id BIGINT,
  p_root_limit INT,
  p_root_offset INT
) RETURNS SETOF comments_with_author
-- STABLE is used for optimization. It tells to the engine that db will not be modified, only queried
LANGUAGE sql STABLE AS $$
  WITH RECURSIVE
  -- getting root comments
  roots AS (
    SELECT c.*
    FROM comments c
    WHERE c.post_id = p_post_id AND c.parent_id IS NULL
    ORDER BY c.created_at DESC, c.id
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
	  -- creating array of order indexes for the final sort
	  -- it gives every comment its place in ordered by creation date list
      ARRAY[ROW_NUMBER() OVER (ORDER BY r.created_at DESC, r.id)]::BIGINT[] AS rnk
    FROM roots r

    UNION ALL

    -- getting children of the root comments
    SELECT
      ch.id, ch.user_id, ch.post_id, ch.parent_id, ch.depth,
      ch.upvotes, ch.downvotes, ch.body, ch.created_at, ch.last_modified_at,
      ch.is_deleted, ch.deleted_at, ch.popularity, t.rnk || ch.rn AS rnk
    FROM cte t
	-- using JOIN LATERAL because the condition needs data from multiple sources
    JOIN LATERAL (
      SELECT c.*,
			-- index in ordered by creation date list, same thing as for the root comments
             ROW_NUMBER() OVER (ORDER BY c.created_at DESC, c.id) AS rn
      FROM comments c
      WHERE c.post_id = t.post_id
        AND c.parent_id = t.id
    ) ch ON TRUE
  )
  SELECT
    c.id, c.user_id, c.post_id, c.parent_id, c.depth,
  	c.upvotes, c.downvotes, c.body, c.created_at, c.last_modified_at,
  	c.is_deleted, c.deleted_at, c.popularity,
  	u.display_name    AS user_display_name,
  	u.profile_img_url AS user_profile_img_url
  FROM cte c
  JOIN users u ON u.id = c.user_id
  -- utilising ordered index array
  ORDER BY rnk;
$$;

-- roots by oldest
CREATE INDEX IF NOT EXISTS comments_roots_oldest
  ON comments (post_id, created_at ASC, id)
  WHERE parent_id IS NULL;

-- children by oldest
CREATE INDEX IF NOT EXISTS comments_children_oldest
  ON comments (post_id, parent_id, created_at ASC, id);

-- roots by newest
CREATE INDEX IF NOT EXISTS comments_roots_newest
  ON comments (post_id, created_at DESC, id)
  WHERE parent_id IS NULL;

-- children by newest
CREATE INDEX IF NOT EXISTS comments_children_newest
  ON comments (post_id, parent_id, created_at DESC, id);

DROP FUNCTION IF EXISTS insert_comment;