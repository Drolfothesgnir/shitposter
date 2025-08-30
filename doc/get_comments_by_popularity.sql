-- utility for extracting comments ordered by popularity
CREATE OR REPLACE FUNCTION get_comments_by_popularity(
  p_post_id BIGINT,
  p_root_limit INT,
  p_root_offset INT
) RETURNS SETOF comments
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
      ch.is_deleted, ch.deleted_at, ch.popularity,
      t.rnk || ch.rn AS rnk
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
    id, user_id, post_id, parent_id, depth,
    upvotes, downvotes, body, created_at, last_modified_at,
    is_deleted, deleted_at, popularity
  FROM cte
  -- utilising ordered index array
  ORDER BY rnk;
$$;
