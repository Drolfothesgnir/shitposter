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
    -- We take a per-(user_id, comment_id) advisory lock to serialize
    -- concurrent voting attempts by the same user on the same comment.
    WITH lock AS (
        SELECT pg_advisory_xact_lock(
            (
                -- high 32 bits: lower 32 bits of user_id
                ((p_user_id  & ((1::bigint << 32) - 1)) << 32)
                -- low 32 bits: lower 32 bits of comment_id
              |  (p_comment_id & ((1::bigint << 32) - 1))
            )
        )
    ),
    orig_vote AS (
        -- Read the previous vote (if any).
        -- CROSS JOIN lock forces `lock` to execute first.
        SELECT cv.vote AS val
        FROM comment_votes cv
        CROSS JOIN lock
        WHERE cv.user_id    = p_user_id
          AND cv.comment_id = p_comment_id
    ),
    new_vote AS (
        -- Upsert the new vote.
        -- We SELECT from `lock` and LEFT JOIN `orig_vote` just to create
        -- a dependency chain: lock -> orig_vote -> new_vote.
        INSERT INTO comment_votes (user_id, comment_id, vote, last_modified_at)
        SELECT
            p_user_id,
            p_comment_id,
            p_vote,
            NOW()
        FROM lock l
        LEFT JOIN orig_vote ov ON TRUE
        ON CONFLICT (user_id, comment_id) DO UPDATE SET
            vote             = EXCLUDED.vote,
            last_modified_at = EXCLUDED.last_modified_at
        RETURNING *
    )
    SELECT
        nv.*,
        -- 0 = no previous vote (nice for Go-side arithmetic)
        COALESCE(ov.val, 0) AS original_vote
    FROM new_vote nv
    LEFT JOIN orig_vote ov ON TRUE;
$$ LANGUAGE sql;

DROP FUNCTION IF EXISTS vote_comment(BIGINT, BIGINT, INT);

DROP FUNCTION IF EXISTS delete_comment_vote(BIGINT, BIGINT);