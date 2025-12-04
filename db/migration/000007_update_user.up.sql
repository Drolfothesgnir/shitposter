ALTER TABLE users ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00Z';

CREATE OR REPLACE FUNCTION update_user(
	p_user_id BIGINT,
	p_username TEXT,
	p_email TEXT,
	p_profile_img_url TEXT
) RETURNS TABLE (
	id BIGINT,
	username TEXT,
	email TEXT,
	profile_img_url TEXT,
	is_deleted BOOLEAN,
	last_modified_at TIMESTAMPTZ,
	updated BOOLEAN -- true if the update was successful
) AS $$
	WITH target AS (
		SELECT 
			id,
			username,
			email,
			profile_img_url,
			is_deleted,
			last_modified_at
		FROM users 
		WHERE id = p_user_id
	),
	updated_row AS (
		UPDATE users u
		SET
			username = COALESCE(p_username, t.username),
			email = COALESCE(p_email, t.email),
			profile_img_url = COALESCE(p_profile_img_url, t.profile_img_url),
			last_modified_at = NOW()
		FROM target t
		WHERE u.id = t.id AND u.is_deleted = false -- only allowed to update active users
		RETURNING u.*
	)
	SELECT 
		t.id AS id, -- no coalesce since id is not changed
		COALESCE(u.username, t.username) AS username,
		COALESCE(u.email, t.email) AS email,
		COALESCE(u.profile_img_url, t.profile_img_url) AS profile_img_url,
		t.is_deleted AS is_deleted, -- original deletion flag (user's status is not changed here)
		COALESCE(u.last_modified_at, t.last_modified_at) AS last_modified_at,
		(u.id IS NOT NULL) AS updated
	FROM target t
	LEFT JOIN updated_row u USING (id);

$$ LANGUAGE sql;