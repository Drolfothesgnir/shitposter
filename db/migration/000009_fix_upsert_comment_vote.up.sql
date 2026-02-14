CREATE OR REPLACE FUNCTION upsert_comment_vote(
    p_user_id    BIGINT,
    p_comment_id BIGINT,
    p_vote       SMALLINT
) RETURNS TABLE (
    id               BIGINT,
    user_id          BIGINT,
    comment_id       BIGINT,
    vote             SMALLINT,
    created_at       TIMESTAMPTZ,
    last_modified_at TIMESTAMPTZ,
    original_vote    SMALLINT
) AS $$
-- Prefer table column names over plpgsql output variable names
-- when they collide (e.g. comment_votes.user_id vs the RETURNS TABLE "user_id").
#variable_conflict use_column
DECLARE
    v_original_vote SMALLINT;
BEGIN
    -- 1. Acquire per-(user, comment) advisory lock (held until transaction end).
    PERFORM pg_advisory_xact_lock(
        ((p_user_id  & ((1::bigint << 32) - 1)) << 32)
      |  (p_comment_id & ((1::bigint << 32) - 1))
    );

    -- 2. Read previous vote with a fresh snapshot (plpgsql gives each
    --    statement its own snapshot under READ COMMITTED, so this sees
    --    any vote committed before the lock was acquired).
    SELECT cv.vote INTO v_original_vote
      FROM comment_votes cv
     WHERE cv.user_id    = p_user_id
       AND cv.comment_id = p_comment_id;

    -- 3. Upsert the new vote and return the row together with the
    --    original vote value (0 when there was no prior vote).
    RETURN QUERY
    INSERT INTO comment_votes (user_id, comment_id, vote, last_modified_at)
    VALUES (p_user_id, p_comment_id, p_vote, NOW())
    ON CONFLICT (user_id, comment_id) DO UPDATE SET
        vote             = EXCLUDED.vote,
        last_modified_at = EXCLUDED.last_modified_at
    RETURNING
        comment_votes.id,
        comment_votes.user_id,
        comment_votes.comment_id,
        comment_votes.vote,
        comment_votes.created_at,
        comment_votes.last_modified_at,
        COALESCE(v_original_vote, 0::SMALLINT);
END;
$$ LANGUAGE plpgsql;
