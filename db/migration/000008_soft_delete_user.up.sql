-- little db normalization
-- now empty profile image URLs are '' instead of NULL
UPDATE users 
SET profile_img_url = NULL
WHERE profile_img_url = '';

-- required for correct sql to go mapping via sqlc;
CREATE DOMAIN optional_string TEXT;

-- soft_delete_user soft deletes user with possibility of restoration.
CREATE OR REPLACE FUNCTION soft_delete_user(
  p_user_id BIGINT
) RETURNS TABLE (
  id BIGINT,
  username TEXT,
  display_name TEXT,
  email TEXT,
  profile_img_url optional_string,
  is_deleted BOOLEAN,
  deleted_at TIMESTAMPTZ,
  last_modified_at TIMESTAMPTZ,
  success BOOLEAN -- TRUE if delete operation performed well or the user was already deleted
) AS $$
  WITH target AS (
    SELECT * FROM users
    WHERE id = p_user_id
  ),
  deleted_row AS (
    UPDATE users u
    SET
      is_deleted = TRUE,
	  -- user's will be displayed as '[deleted]' publicly
      display_name = '[deleted]',
      deleted_at = NOW(),
      last_modified_at = NOW(),
	  -- saving user's data for later 
	  -- in case there will be some account restoring feature
      archived_username = u.username,
      archived_email    = u.email,
	  -- replace username and email with dummy values to free
	  -- their values for avoiding duplicate errors
      username = CONCAT('deleted_user_', u.id),
      email    = CONCAT('deleted_', u.id, '@invalid.local'),
      profile_img_url = NULL
    FROM target t
	-- check u.is_deleted = FALSE and t.is_deleted = FALSE
	-- are similar right now, but u.is_deleted is preferred
	-- because it uses actual fresh user data
    WHERE u.id = t.id AND u.is_deleted = FALSE
    RETURNING u.*
  )
  SELECT 
    t.id AS id,
    COALESCE(d.username, t.username) AS username,
    COALESCE(d.display_name, t.display_name) AS display_name,
    COALESCE(d.email, t.email) AS email,
    CASE
      WHEN d.id IS NOT NULL THEN NULL            -- delete succeeded
      ELSE t.profile_img_url                     -- delete didn’t happen
    END AS profile_img_url,
    COALESCE(d.is_deleted, t.is_deleted) AS is_deleted,
    COALESCE(d.deleted_at, t.deleted_at) AS deleted_at,
    COALESCE(d.last_modified_at, t.last_modified_at) as last_modified_at,
	-- achieving idempotency
	(d.id IS NOT NULL OR t.is_deleted = TRUE) AS success
  FROM target t
  LEFT JOIN deleted_row d USING (id);
$$ LANGUAGE sql;