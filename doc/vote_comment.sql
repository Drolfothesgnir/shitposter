-- UPSERT of a comment vote
CREATE OR REPLACE FUNCTION vote_comment(
  p_user_id    bigint,
  p_comment_id bigint,
  p_vote       int
) RETURNS comments
LANGUAGE sql AS $$
WITH
-- trying to update already existed vote
upd AS (
  UPDATE comment_votes v
     SET 
      vote = p_vote,
      last_modified_at = NOW()
   WHERE v.user_id = p_user_id
     AND v.comment_id = p_comment_id
	 -- checking if two votes have different values
     AND v.vote IS DISTINCT FROM p_vote
  RETURNING
    v.comment_id,
    (CASE WHEN p_vote = 1  THEN  1 ELSE -1 END) AS up_delta,
    (CASE WHEN p_vote = -1 THEN  1 ELSE -1 END) AS down_delta
),

ins AS (
  INSERT INTO comment_votes(user_id, comment_id, vote)
  VALUES (p_user_id, p_comment_id, p_vote)
  -- if another transaction is already trying to create new vote do nothing
  ON CONFLICT (user_id, comment_id) DO NOTHING
  RETURNING
    comment_id,
    (p_vote = 1)::int  AS up_delta,
    (p_vote = -1)::int AS down_delta
),
-- extracting deltas from whatever operation succeeded
delta AS (
  SELECT * FROM upd
  UNION ALL
  SELECT * FROM ins
),
-- applying deltas to the comments counters
bump AS (
  UPDATE comments c
     SET upvotes   = c.upvotes   + COALESCE(d.up_delta, 0),
         downvotes = c.downvotes + COALESCE(d.down_delta, 0)
    FROM delta d
   WHERE c.id = d.comment_id
  RETURNING c.*
)
-- returning updated comment
SELECT *
FROM bump 
UNION ALL
SELECT c.*
FROM comments c
-- check in comments only if bump didn't return anything
WHERE c.id = p_comment_id
	AND NOT EXISTS (SELECT 1 FROM bump);
$$;
