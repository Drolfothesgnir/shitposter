-- Function to update a comment after checking the input params
-- and the db state in one query.
CREATE OR REPLACE FUNCTION update_comment(
    p_comment_id BIGINT,
    p_user_id BIGINT,
    p_post_id BIGINT,
    p_body TEXT
) RETURNS TABLE (
    id BIGINT,
    user_id BIGINT,
    post_id BIGINT,
    is_deleted BOOLEAN,
    body TEXT,
    last_modified_at TIMESTAMPTZ,
    updated BOOLEAN
) AS $$
    WITH target AS (
        SELECT 
            id, 
            user_id, 
            post_id, 
            is_deleted, 
            body, 
            last_modified_at
        FROM comments
        WHERE id = p_comment_id
    ),
    updated AS (
        UPDATE comments c
        SET body = p_body, last_modified_at = now()
        FROM target t
        WHERE c.id = t.id
          AND t.user_id = p_user_id -- checks if the client tries to update his own comment
          AND t.post_id = p_post_id -- checks if the target comment belongs to a correct post
          AND t.is_deleted = false -- checks if the target comment is not deleted
        RETURNING c.*
    )
    SELECT
        COALESCE(u.id, t.id) AS id,
        COALESCE(u.user_id, t.user_id) AS user_id,
        COALESCE(u.post_id, t.post_id) AS post_id,
        COALESCE(u.is_deleted, t.is_deleted) AS is_deleted,
        COALESCE(u.body, t.body) AS body,
        COALESCE(u.last_modified_at, t.last_modified_at) AS last_modified_at,
        (u.id IS NOT NULL) AS updated
    FROM target t
    LEFT JOIN updated u ON t.id = u.id;
$$ LANGUAGE sql;
