-- function to update webauthn passkey sign count.
CREATE OR REPLACE FUNCTION record_credential_use(
	p_cred_id    BYTEA,
	p_sign_count BIGINT
) RETURNS TABLE (
	cred_exists   BOOLEAN,
	prev_count    BIGINT,
	is_suspicious BOOLEAN
) AS $$
	WITH target AS (
		SELECT id, sign_count
		FROM webauthn_credentials
		WHERE id = p_cred_id
		FOR UPDATE -- crucial for avoiding race conditions
	),
	-- Policy:
	--   - the only tolerated non-advancing counter case is 0 -> 0
	--   - otherwise the incoming sign count must strictly increase
	--   - any non-increasing value outside the 0 -> 0 case is suspicious
	policy AS (
		SELECT 
			(p_sign_count <= t.sign_count) AND 
			(p_sign_count <> 0 OR t.sign_count <> 0) AS is_suspicious
		FROM target t
	),
	updated AS (
		UPDATE webauthn_credentials c
		SET 
			sign_count = 
			CASE
				WHEN p.is_suspicious IS FALSE THEN p_sign_count
				ELSE c.sign_count
			END,
			last_used_at = 
			CASE
				WHEN p.is_suspicious IS FALSE THEN NOW()
				ELSE c.last_used_at
			END
		FROM target t
		LEFT JOIN policy p ON TRUE
		WHERE c.id = t.id
		RETURNING c.id, c.sign_count
	)
	SELECT
		(t.id IS NOT NULL) AS cred_exists,
		COALESCE(t.sign_count, -1) AS prev_count,
		COALESCE(p.is_suspicious, FALSE) AS is_suspicious
	 -- needed because when target will be null, 
	 -- simple target LEFT JOIN updated will return empty rows
	FROM (SELECT 1) d
	LEFT JOIN target t ON TRUE
	LEFT JOIN policy p ON TRUE
	LEFT JOIN updated u ON t.id = u.id;
$$ LANGUAGE sql;
