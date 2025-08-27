-- UPSERT of a post vote
CREATE OR REPLACE FUNCTION vote_post(
  p_user_id    bigint,
  p_post_id bigint,
  p_vote       int
) RETURNS posts
LANGUAGE sql AS $$
WITH
-- trying to update already existed vote
upd AS (
  UPDATE post_votes v
     SET 
      vote = p_vote,
      last_modified_at = NOW()
   WHERE v.user_id = p_user_id
     AND v.post_id = p_post_id
	 -- checking if two votes have different values
     AND v.vote IS DISTINCT FROM p_vote
  RETURNING
    v.post_id,
    (CASE WHEN p_vote = 1  THEN  1 ELSE -1 END) AS up_delta,
    (CASE WHEN p_vote = -1 THEN  1 ELSE -1 END) AS down_delta
),

ins AS (
  INSERT INTO post_votes(user_id, post_id, vote)
  VALUES (p_user_id, p_post_id, p_vote)
  -- if another transaction is already trying to create new vote do nothing
  ON CONFLICT (user_id, post_id) DO NOTHING
  RETURNING
    post_id,
    (p_vote = 1)::int  AS up_delta,
    (p_vote = -1)::int AS down_delta
),
-- extracting deltas from whatever operation succeeded
delta AS (
  SELECT * FROM upd
  UNION ALL
  SELECT * FROM ins
),
-- applying deltas to the posts counters
bump AS (
  UPDATE posts p
     SET upvotes   = p.upvotes   + COALESCE(d.up_delta, 0),
         downvotes = p.downvotes + COALESCE(d.down_delta, 0)
    FROM delta d
   WHERE p.id = d.post_id
  RETURNING p.*
)
-- returning updated post
SELECT *
FROM bump 
UNION ALL
SELECT p.*
FROM posts p
-- check in posts only if bump didn't return anything
WHERE p.id = p_post_id
	AND NOT EXISTS (SELECT 1 FROM bump);
$$;