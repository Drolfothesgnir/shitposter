-- function to update webauthn passkey sign count.
CREATE OR REPLACE FUNCTION cred_update_sign_count(
	p_cred_id    BYTEA,
	p_sign_count BIGINT
) RETURNS TABLE (
	cred_exists BOOLEAN,
	advanced    BOOLEAN,
	prev_count  BIGINT
) AS $$
	WITH target AS (
		SELECT id, sign_count
		FROM webauthn_credentials
		WHERE id = p_cred_id
		FOR UPDATE -- crucial for avoiding race conditions
	),	
	updated AS (
		UPDATE webauthn_credentials c
		SET sign_count = GREATEST(p_sign_count, c.sign_count)
		FROM target t
		WHERE c.id = t.id
		RETURNING c.id, c.sign_count
	)
	SELECT
		(t.id IS NOT NULL) AS cred_exists,
		COALESCE(u.sign_count > t.sign_count, FALSE) AS advanced,
		COALESCE(t.sign_count, -1) AS prev_count
	 -- needed because when target will be null, 
	 -- simple target LEFT JOIN updated will return empty rows
	FROM (SELECT 1) d
	LEFT JOIN target t ON TRUE
	LEFT JOIN updated u ON t.id = u.id;
$$ LANGUAGE sql;