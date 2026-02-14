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
    WITH lock AS (
        SELECT pg_advisory_xact_lock(
            (
                ((p_user_id  & ((1::bigint << 32) - 1)) << 32)
              |  (p_comment_id & ((1::bigint << 32) - 1))
            )
        )
    ),
    orig_vote AS (
        SELECT cv.vote AS val
        FROM comment_votes cv
        CROSS JOIN lock
        WHERE cv.user_id    = p_user_id
          AND cv.comment_id = p_comment_id
    ),
    new_vote AS (
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
        COALESCE(ov.val, 0) AS original_vote
    FROM new_vote nv
    LEFT JOIN orig_vote ov ON TRUE;
$$ LANGUAGE sql;
